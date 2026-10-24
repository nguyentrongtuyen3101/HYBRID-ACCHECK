// Package security hiện thực các port bảo mật: hash mật khẩu, JWT, token ngẫu nhiên, sinh id.
//
// Đổi sang IdP ngoài (Keycloak, Auth0...) sau này: viết 1 TokenIssuer/verifier OIDC mới
// trả về subject, map sang users.external_subject — use case và HTTP không phải sửa.
package security

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"reqsentinel/api/internal/domain"
)

// ---------- Mật khẩu ----------

type BcryptHasher struct{}

func (BcryptHasher) Hash(pw string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.DefaultCost)
	return string(b), err
}

// dummyHash dùng khi user không tồn tại, để thời gian phản hồi giống như sai mật khẩu.
var dummyHash, _ = bcrypt.GenerateFromPassword([]byte("reqsentinel-dummy-password"), bcrypt.DefaultCost)

func (BcryptHasher) Compare(hash *string, pw string) bool {
	if hash == nil || *hash == "" {
		bcrypt.CompareHashAndPassword(dummyHash, []byte(pw))
		return false
	}
	return bcrypt.CompareHashAndPassword([]byte(*hash), []byte(pw)) == nil
}

// ---------- JWT ----------

const Issuer = "reqsentinel"

type JWTIssuer struct {
	secret []byte
	ttl    time.Duration
}

func NewJWTIssuer(secret string, ttl time.Duration) *JWTIssuer {
	return &JWTIssuer{secret: []byte(secret), ttl: ttl}
}

func (t *JWTIssuer) Issue(userID string, role domain.SystemRole) (string, time.Time, error) {
	now := time.Now()
	exp := now.Add(t.ttl)
	claims := jwt.MapClaims{
		"sub":  userID,
		"role": string(role), // chỉ để client hiển thị; server luôn đọc role từ DB
		"iss":  Issuer,
		"iat":  now.Unix(),
		"exp":  exp.Unix(),
	}
	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(t.secret)
	return signed, exp, err
}

var ErrInvalidToken = errors.New("invalid or expired token")

func (t *JWTIssuer) Verify(token string) (string, error) {
	parsed, err := jwt.Parse(token, func(*jwt.Token) (any, error) { return t.secret, nil },
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(Issuer),
		jwt.WithExpirationRequired(),
	)
	if err != nil || !parsed.Valid {
		return "", ErrInvalidToken
	}
	sub, err := parsed.Claims.GetSubject()
	if err != nil || sub == "" {
		return "", ErrInvalidToken
	}
	return sub, nil
}

// ---------- Token ngẫu nhiên (mời thành viên, reset mật khẩu) ----------

type OpaqueTokens struct{}

func (OpaqueTokens) New() (string, string) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic(fmt.Sprintf("crypto/rand: %v", err))
	}
	raw := base64.RawURLEncoding.EncodeToString(b)
	return raw, OpaqueTokens{}.Hash(raw)
}

func (OpaqueTokens) Hash(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

// ---------- UUID v4 ----------

func NewUUID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(fmt.Sprintf("crypto/rand: %v", err))
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	h := hex.EncodeToString(b[:])
	return h[0:8] + "-" + h[8:12] + "-" + h[12:16] + "-" + h[16:20] + "-" + h[20:32]
}
