package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/ETAnderson/conductor/internal/api/auth"
	"github.com/ETAnderson/conductor/internal/config"
	"github.com/golang-jwt/jwt/v5"
)

func main() {
	// Ensure .env is loaded for fresh shells too
	config.LoadDotEnv()

	var (
		tenantID = flag.Uint64("tenant", 1, "tenant_id claim value")
		ttl      = flag.Duration("ttl", 30*time.Minute, "token TTL (e.g. 30m, 2h)")
		issuer   = flag.String("iss", "conductor", "issuer (iss)")
		subject  = flag.String("sub", "dev-client", "subject (sub)")

		// expects env var containing a FILE PATH to the private key PEM
		envKey = flag.String("env", "JWT_PRIVATE_KEY_PATH", "env var containing RSA private key PEM *path*")
	)
	flag.Parse()

	// load private key from file path in env
	priv, err := auth.LoadRSAPrivateKeyFromPathEnv(*envKey)
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
