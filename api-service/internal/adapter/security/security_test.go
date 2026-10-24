package security

import (
	"testing"
	"time"

	"reqsentinel/api/internal/domain"
)

const secret = "unit-test-secret-unit-test-secret-1234"

func TestTokenRoundTrip(t *testing.T) {
	issuer := NewJWTIssuer(secret, time.Hour)
	token, exp, err := issuer.Issue("user-123", domain.SystemRoleUser)
	if err != nil {
		t.Fatal(err)
	}
	if time.Until(exp) <= 0 {
		t.Fatal("expiry should be in the future")
	}
	sub, err := issuer.Verify(token)
	if err != nil || sub != "user-123" {
		t.Fatalf("Verify = %q, %v", sub, err)
	}
}

func TestTokenRejected(t *testing.T) {
	issuer := NewJWTIssuer(secret, time.Hour)
	token, _, _ := issuer.Issue("user-123", domain.SystemRoleUser)

	other := NewJWTIssuer("another-secret-another-secret-123456", time.Hour)
	if _, err := other.Verify(token); err == nil {
		t.Fatal("token signed with a different secret must be rejected")
	}
	expired := NewJWTIssuer(secret, -time.Minute)
	old, _, _ := expired.Issue("user-123", domain.SystemRoleUser)
	if _, err := issuer.Verify(old); err == nil {
		t.Fatal("expired token must be rejected")
	}
	if _, err := issuer.Verify(token + "x"); err == nil {
		t.Fatal("tampered token must be rejected")
	}
}

func TestPassword(t *testing.T) {
	var h BcryptHasher
	hash, err := h.Hash("correct horse")
	if err != nil {
		t.Fatal(err)
	}
	if !h.Compare(&hash, "correct horse") || h.Compare(&hash, "wrong horse") {
		t.Fatal("Compare mismatch")
	}
	if h.Compare(nil, "anything") {
		t.Fatal("user without password must not log in")
	}
}

func TestOpaqueTokensAndUUID(t *testing.T) {
	raw, hash := OpaqueTokens{}.New()
	if len(raw) < 40 || (OpaqueTokens{}).Hash(raw) != hash {
		t.Fatalf("token/hash mismatch: %q %q", raw, hash)
	}
	id := NewUUID()
	if len(id) != 36 || id[14] != '4' || id == NewUUID() {
		t.Fatalf("bad uuid %q", id)
	}
}

func TestLoginLimiter(t *testing.T) {
	l := NewLoginLimiter(3, time.Minute)
	for i := 0; i < 3; i++ {
		if l.Blocked("k") {
			t.Fatalf("blocked too early at attempt %d", i)
		}
		l.Fail("k")
	}
	if !l.Blocked("k") {
		t.Fatal("should be blocked after 3 failures")
	}
	l.Reset("k")
	if l.Blocked("k") {
		t.Fatal("reset should unblock")
	}
}
