package auth

import (
	"crypto/rsa"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	TenantID uint64 `json:"tenant_id"`
	jwt.RegisteredClaims
}

// LoadRSAPublicKeyFromEnv reads a PEM public key from an env var.
// It supports either a normal multi-line PEM, or a single-line PEM with \n escapes.
func LoadRSAPublicKeyFromEnv(envKey string) (*rsa.PublicKey, error) {
	raw := strings.TrimSpace(os.Getenv(envKey))
	if raw == "" {
		return nil, fmt.Errorf("%s is not set", envKey)
	}

	// Allow users to store PEM in env as single line with "\n"
	raw = strings.ReplaceAll(raw, `\n`, "\n")

	pub, err := jwt.ParseRSAPublicKeyFromPEM([]byte(raw))
	if err != nil {
		return nil, fmt.Errorf("parse public key pem failed: %w", err)
	}

	return pub, nil
}

func ParseAndValidateRS256(tokenString string, pub *rsa.PublicKey) (*Claims, error) {
	if pub == nil {
		return nil, errors.New("public key is nil")
	}

	parser := jwt.NewParser(
		jwt.WithValidMethods([]string{jwt.SigningMethodRS256.Name}),
		jwt.WithLeeway(30*time.Second),
	)

	tok, err := parser.ParseWithClaims(tokenString, &Claims{}, func(t *jwt.Token) (any, error) {
		// ValidMethods already restricts method; still defend in depth.
		if t.Method.Alg() != jwt.SigningMethodRS256.Alg() {
			return nil, fmt.Errorf("unexpected alg: %s", t.Method.Alg())
		}
		return pub, nil
	})
	if err != nil {
		return nil, err
	}

	claims, ok := tok.Claims.(*Claims)
	if !ok || !tok.Valid {
		return nil, errors.New("invalid token")
	}

	if claims.TenantID == 0 {
		return nil, errors.New("tenant_id missing")
	}

	return claims, nil
}

// Helper for tests/debug to sign tokens with a private key.
func SignRS256ForTests(priv *rsa.PrivateKey, tenantID uint64, ttl time.Duration) (string, error) {
	now := time.Now().UTC()

	c := Claims{
		TenantID: tenantID,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "conductor",
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
	}

	tok := jwt.NewWithClaims(jwt.SigningMethodRS256, c)
	return tok.SignedString(priv)
}

func MustJSON(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}
