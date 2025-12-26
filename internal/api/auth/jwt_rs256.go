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

// LoadRSAPublicKeyFromEnvPEM reads a PEM public key from an env var.
// Supports either a normal multi-line PEM, or a single-line PEM with \n escapes.
// Use this only if you store the PEM *contents* in env (not a file path).
func LoadRSAPublicKeyFromEnvPEM(envKey string) (*rsa.PublicKey, error) {
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

// LoadRSAPublicKeyFromPathEnv reads a PEM public key from a file path stored in an env var.
func LoadRSAPublicKeyFromPathEnv(envKey string) (*rsa.PublicKey, error) {
	path := strings.TrimSpace(os.Getenv(envKey))
	if path == "" {
		return nil, fmt.Errorf("%s is not set", envKey)
	}

	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read public key failed (%s): %w", path, err)
	}

	pub, err := jwt.ParseRSAPublicKeyFromPEM(b)
	if err != nil {
		return nil, fmt.Errorf("parse public key pem failed (%s): %w", path, err)
	}

	return pub, nil
}

// LoadRSAPrivateKeyFromPathEnv reads an RSA private key PEM (PKCS#1 or PKCS#8) from a file path stored in an env var.
func LoadRSAPrivateKeyFromPathEnv(envKey string) (*rsa.PrivateKey, error) {
	path := strings.TrimSpace(os.Getenv(envKey))
	if path == "" {
		return nil, fmt.Errorf("%s is not set", envKey)
	}

	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read private key failed (%s): %w", path, err)
	}

	priv, err := jwt.ParseRSAPrivateKeyFromPEM(b)
	if err != nil {
		return nil, fmt.Errorf("parse private key pem failed (%s): %w", path, err)
	}

	return priv, nil
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
