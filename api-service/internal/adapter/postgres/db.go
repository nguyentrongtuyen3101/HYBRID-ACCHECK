// Package postgres hiện thực các repository port bằng PostgreSQL (pgx).
package postgres

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"sort"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"reqsentinel/api/internal/domain"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// Khoá advisory để nhiều replica start cùng lúc không chạy migration chồng nhau.
const migrationLockID = 72620011

type querier interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// DB bọc connection pool; repository lấy transaction hiện tại (nếu có) từ context.
type DB struct {
	Pool *pgxpool.Pool
}

func Connect(ctx context.Context, url string) (*DB, error) {
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		return nil, fmt.Errorf("parse DATABASE_URL: %w", err)
	}
	cfg.ConnConfig.RuntimeParams["search_path"] = "public"
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}
	return &DB{Pool: pool}, nil
}

func (db *DB) Close() { db.Pool.Close() }

func (db *DB) Ping(ctx context.Context) error { return db.Pool.Ping(ctx) }

type txKey struct{}

func (db *DB) q(ctx context.Context) querier {
	if tx, ok := ctx.Value(txKey{}).(pgx.Tx); ok {
		return tx
	}
	return db.Pool
}

// WithinTx hiện thực app.TxManager; gọi lồng nhau thì dùng lại transaction ngoài.
func (db *DB) WithinTx(ctx context.Context, fn func(ctx context.Context) error) error {
	if _, ok := ctx.Value(txKey{}).(pgx.Tx); ok {
		return fn(ctx)
	}
	return pgx.BeginFunc(ctx, db.Pool, func(tx pgx.Tx) error {
		return fn(context.WithValue(ctx, txKey{}, tx))
	})
}

// Migrate chạy lần lượt các file migrations/*.sql chưa được áp dụng.
func (db *DB) Migrate(ctx context.Context, log *slog.Logger) error {
	conn, err := db.Pool.Acquire(ctx)
	if err != nil {
		return err
	}
	defer conn.Release()

	if _, err := conn.Exec(ctx, "SELECT pg_advisory_lock($1)", migrationLockID); err != nil {
		return err
	}
	defer conn.Exec(context.Background(), "SELECT pg_advisory_unlock($1)", migrationLockID)

	if _, err := conn.Exec(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
		version    text PRIMARY KEY,
		applied_at timestamptz NOT NULL DEFAULT now()
	)`); err != nil {
		return err
	}
	files, err := fs.Glob(migrationsFS, "migrations/*.sql")
	if err != nil {
		return err
	}
	sort.Strings(files)
	for _, file := range files {
		var applied bool
		if err := conn.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM schema_migrations WHERE version = $1)", file).
			Scan(&applied); err != nil {
			return err
		}
		if applied {
			continue
		}
		sqlBytes, err := migrationsFS.ReadFile(file)
		if err != nil {
			return err
		}
		err = pgx.BeginFunc(ctx, conn, func(tx pgx.Tx) error {
			if _, err := tx.Exec(ctx, string(sqlBytes)); err != nil {
				return fmt.Errorf("migration %s: %w", file, err)
			}
			_, err := tx.Exec(ctx, "INSERT INTO schema_migrations (version) VALUES ($1)", file)
			return err
		})
		if err != nil {
			return err
		}
		log.Info("applied migration", "file", file)
	}
	_, err = conn.Exec(ctx, enableRowLevelSecurity)
	return err
}

// Supabase (và các nền tảng có PostgREST) tự mở REST API cho schema public qua role anon/authenticated.
// Bật RLS không kèm policy trên mọi bảng của app để các role đó không đọc/ghi được gì.
// App kết nối bằng role sở hữu bảng nên không bị RLS chặn (chỉ FORCE RLS mới áp lên owner).
// Chạy sau mỗi lần migrate nên bảng thêm ở migration sau cũng được bật.
const enableRowLevelSecurity = `DO $$
DECLARE t record;
BEGIN
	FOR t IN
		SELECT c.relname FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE n.nspname = 'public' AND c.relkind = 'r' AND NOT c.relrowsecurity
		  AND pg_get_userbyid(c.relowner) = current_user
	LOOP
		EXECUTE format('ALTER TABLE public.%I ENABLE ROW LEVEL SECURITY', t.relname);
	END LOOP;
END $$`

// ---------- helpers ----------

func notFound(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrNotFound
	}
	return err
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func affected(tag pgconn.CommandTag, err error) error {
	if err == nil && tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return err
}

type scanner interface{ Scan(...any) error }

// scanTail cho phép dùng lại hàm scan chuẩn khi câu SELECT có thêm cột phía sau.
type scanTail struct {
	scanner
	extra []any
}

func (s scanTail) Scan(dest ...any) error { return s.scanner.Scan(append(dest, s.extra...)...) }

func collect[T any](rows pgx.Rows, scan func(scanner) (T, error)) ([]T, error) {
	defer rows.Close()
	out := []T{}
	for rows.Next() {
		v, err := scan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}
