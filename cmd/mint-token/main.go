package main

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/ETAnderson/conductor/internal/api/auth"
	"github.com/golang-jwt/jwt/v5"
)

func main() {
	var (
		tenantID = flag.Uint64("tenant", 1, "tenant_id claim value")
		ttl      = flag.Duration("ttl", 30*time.Minute, "token TTL (e.g. 30m, 2h)")
		issuer   = flag.String("iss", "conductor", "issuer (iss)")
		subject  = flag.String("sub", "dev-client", "subject (sub)")
		envKey   = flag.String("env", "JWT_PRIVATE_KEY_PEM", "env var containing RSA private key PEM")
	)
	flag.Parse()

	priv, err := loadRSAPrivateKeyFromEnv(*envKey)
	if err != nil {
		fmt.Fprintf(os.Stderr, "load private key failed: %v\n", err)
		os.Exit(1)
	}

	now := time.Now().UTC()

	claims := auth.Claims{
		TenantID: *tenantID,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    *issuer,
			Subject:   *subject,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(*ttl)),
		},
	}

	tok := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	s, err := tok.SignedString(priv)
	if err != nil {
		fmt.Fprintf(os.Stderr, "sign token failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(s)
}

func loadRSAPrivateKeyFromEnv(envKey string) (*rsa.PrivateKey, error) {
	raw := strings.TrimSpace(os.Getenv(envKey))
	if raw == "" {
		return nil, fmt.Errorf("%s is not set", envKey)
	}

	// Support single-line env with \n escapes
	raw = strings.ReplaceAll(raw, `\n`, "\n")

	// Decode PEM ourselves so we can support PKCS#1 and PKCS#8 reliably on Windows.
	block, _ := pem.Decode([]byte(raw))
	if block == nil {
		return nil, errors.New("pem decode failed")
	}

	switch block.Type {
	case "RSA PRIVATE KEY":
		// PKCS#1
		priv, err := x509.ParsePKCS1PrivateKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("parse pkcs1 private key failed: %w", err)
		}
		return priv, nil

	case "PRIVATE KEY":
		// PKCS#8
		key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("parse pkcs8 private key failed: %w", err)
		}
		priv, ok := key.(*rsa.PrivateKey)
		if !ok {
			return nil, errors.New("pkcs8 key is not rsa")
		}
		return priv, nil

	default:
		return nil, fmt.Errorf("unsupported pem type: %s", block.Type)
	}
}
