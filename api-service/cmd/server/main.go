// Composition root: nơi duy nhất nối adapter (postgres, http, ml, rabbitmq...) vào use case.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"reqsentinel/api/internal/adapter/excel"
	httpadapter "reqsentinel/api/internal/adapter/http"
	"reqsentinel/api/internal/adapter/mlclient"
	"reqsentinel/api/internal/adapter/postgres"
	"reqsentinel/api/internal/adapter/rabbitmq"
	"reqsentinel/api/internal/adapter/security"
	"reqsentinel/api/internal/adapter/worker"
	"reqsentinel/api/internal/app"
	"reqsentinel/api/internal/app/admin"
	"reqsentinel/api/internal/app/analysis"
	"reqsentinel/api/internal/app/auth"
	"reqsentinel/api/internal/app/conflicts"
	"reqsentinel/api/internal/app/projects"
	"reqsentinel/api/internal/app/reports"
	"reqsentinel/api/internal/app/requirements"
	"reqsentinel/api/internal/app/users"
	"reqsentinel/api/internal/config"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("config", "err", err)
		os.Exit(1)
	}
	log := newLogger(cfg.LogLevel)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := run(ctx, cfg, log); err != nil {
		log.Error("fatal", "err", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, cfg config.Config, log *slog.Logger) error {
	db, err := postgres.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer db.Close()
	if err := db.Migrate(ctx, log); err != nil {
		return err
	}

	// ---------- Adapter ----------
	var (
		userRepo    = postgres.NewUserRepo(db)
		projectRepo = postgres.NewProjectRepo(db)
		memberRepo  = postgres.NewMemberRepo(db)
		runRepo     = postgres.NewRunRepo(db)
		issueRepo   = postgres.NewIssueRepo(db)
		auditRepo   = postgres.NewAuditRepo(db, log)
		storyRepo   = postgres.NewStoryRepo(db)
		pmRepo      = postgres.NewPMRepo(db)
		termRepo    = postgres.NewTermRepo(db)
		versionRepo = postgres.NewPMVersionRepo(db)
		jobRepo     = postgres.NewJobRepo(db)
		outbox      = postgres.NewOutbox(db)
		ml          = mlclient.New(cfg.MLServiceURL, cfg.MLServiceToken, cfg.MLTimeout)
		clock       = app.Clock(time.Now)
		newID       = app.IDGenerator(security.NewUUID)
		hasher      = security.BcryptHasher{}
		opaque      = security.OpaqueTokens{}
		sheets      = excel.Sheets{}
		authorizer  = app.Authorizer{Projects: projectRepo, Members: memberRepo}
	)

	// ---------- Use case ----------
	svc := httpadapter.Services{
		Auth: auth.New(auth.Deps{Users: userRepo, Resets: postgres.NewPasswordResetRepo(db), Hasher: hasher,
			Tokens: security.NewJWTIssuer(cfg.JWTSecret, cfg.TokenTTL), Opaque: opaque,
			Limiter: security.NewLoginLimiter(10, 15*time.Minute), Events: outbox, Tx: db, Audit: auditRepo,
			Clock: clock, NewID: newID, AllowSignup: cfg.AllowSignup, WebURL: cfg.WebURL}),
		Users: users.New(users.Deps{Users: userRepo, Projects: projectRepo, Hasher: hasher, Audit: auditRepo}),
		Projects: projects.New(projects.Deps{Projects: projectRepo, Members: memberRepo,
			Invitations: postgres.NewInvitationRepo(db), Users: userRepo, Stories: storyRepo, PM: pmRepo,
			Auth: authorizer, Opaque: opaque, Events: outbox, Tx: db, Audit: auditRepo, Clock: clock, NewID: newID,
			WebURL: cfg.WebURL}),
		Requirements: requirements.New(requirements.Deps{Stories: storyRepo, PM: pmRepo, Terms: termRepo,
			PMVersions: versionRepo, Auth: authorizer, Sheets: sheets, Tx: db, Audit: auditRepo}),
		Analysis: analysis.New(analysis.Deps{Analyzer: ml, Runs: runRepo, Issues: issueRepo, Jobs: jobRepo,
			Stories: storyRepo, PM: pmRepo, PMVersions: versionRepo, Terms: termRepo, Projects: projectRepo,
			Members: memberRepo, Auth: authorizer, Events: outbox, Tx: db, Audit: auditRepo, Clock: clock,
			NewID: newID, Log: log}),
		Conflicts: conflicts.New(conflicts.Deps{Issues: issueRepo, Members: memberRepo, Auth: authorizer,
			Events: outbox, Tx: db, Audit: auditRepo, Clock: clock, NewID: newID}),
		Reports: reports.New(reports.Deps{Stories: storyRepo, PM: pmRepo, Terms: termRepo, Runs: runRepo,
			Issues: issueRepo, Auth: authorizer, Sheets: sheets, Clock: clock}),
		Admin: admin.New(admin.Deps{Stats: postgres.NewStatsRepo(db, runRepo, issueRepo), Audit: auditRepo}),
	}

	if err := svc.Users.BootstrapAdmin(ctx, cfg.AdminEmail, cfg.AdminPassword, cfg.AdminName, log); err != nil {
		return err
	}
	if err := svc.Analysis.RecoverInterruptedJobs(ctx); err != nil {
		return err
	}

	// ---------- Tiến trình nền ----------
	var bg sync.WaitGroup
	bgCtx, cancelBg := context.WithCancel(ctx)
	defer func() { cancelBg(); bg.Wait() }()
	bg.Add(1)
	go func() { defer bg.Done(); worker.RunJobs(bgCtx, svc.Analysis, cfg.JobWorkers, 2*time.Second, log) }()
	if cfg.RabbitMQURL != "" {
		bg.Add(1)
		go func() { defer bg.Done(); rabbitmq.NewRelay(cfg.RabbitMQURL, outbox, log).Run(bgCtx) }()
	} else {
		log.Warn("RABBITMQ_URL not set: events stay in the outbox, notifications are disabled")
	}

	// ---------- HTTP ----------
	handler := httpadapter.NewServer(httpadapter.Config{AppName: cfg.AppName, AppVersion: cfg.AppVersion,
		CORSAllowedOrigins: cfg.CORSAllowedOrigins, TrustProxy: cfg.TrustProxy}, svc, db, ml, log).Handler()
	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      cfg.MLTimeout + 30*time.Second, // đủ thời gian chờ ML service
		IdleTimeout:       120 * time.Second,
	}
	errCh := make(chan error, 1)
	go func() {
		log.Info("API listening", "addr", srv.Addr, "ml_service", cfg.MLServiceURL)
		errCh <- srv.ListenAndServe()
	}()
	select {
	case err := <-errCh:
		if !errors.Is(err, http.ErrServerClosed) {
			return err
		}
	case <-ctx.Done():
		log.Info("shutting down")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		return srv.Shutdown(shutdownCtx)
	}
	return nil
}

func newLogger(level string) *slog.Logger {
	lvl := map[string]slog.Level{"DEBUG": slog.LevelDebug, "WARN": slog.LevelWarn, "WARNING": slog.LevelWarn,
		"ERROR": slog.LevelError}[strings.ToUpper(level)]
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: lvl}))
}
