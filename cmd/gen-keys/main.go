package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	outDir := "./secrets"
	if len(os.Args) > 1 {
		outDir = os.Args[1]
	}

	if err := os.MkdirAll(outDir, 0o700); err != nil {
		fmt.Fprintf(os.Stderr, "mkdir failed: %v\n", err)
		os.Exit(1)
	}

	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		fmt.Fprintf(os.Stderr, "keygen failed: %v\n", err)
		os.Exit(1)
	}

	// Private key (PKCS#1): -----BEGIN RSA PRIVATE KEY-----
	privDER := x509.MarshalPKCS1PrivateKey(priv)
	privPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privDER,
	})

	// Public key (SPKI): -----BEGIN PUBLIC KEY-----
	pubDER, err := x509.MarshalPKIXPublicKey(&priv.PublicKey)
	if err != nil {
		fmt.Fprintf(os.Stderr, "marshal public key failed: %v\n", err)
		os.Exit(1)
	}
	pubPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubDER,
	})

	privPath := filepath.Join(outDir, "jwt_private.pem")
	pubPath := filepath.Join(outDir, "jwt_public.pem")

	if err := os.WriteFile(privPath, privPEM, 0o600); err != nil {
		fmt.Fprintf(os.Stderr, "write private key failed: %v\n", err)
		os.Exit(1)
	}
	if err := os.WriteFile(pubPath, pubPEM, 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "write public key failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Wrote %s\nWrote %s\n", privPath, pubPath)
}
