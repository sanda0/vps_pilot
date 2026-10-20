package agentauth

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"time"
)

// Cert is the hub's TLS certificate for the agent channel.
type Cert struct {
	TLS tls.Certificate
	// Fingerprint is the lowercase hex SHA-256 of the leaf certificate DER.
	// Agents pin it; it is embedded in every join token.
	Fingerprint string
}

// LoadOrCreateCert loads certFile/keyFile when both are set. Otherwise it uses
// agent-hub.crt / agent-hub.key in dir, generating a self-signed pair on first
// run. Keep these files: regenerating them changes the fingerprint and every
// enrolled agent will refuse to connect.
func LoadOrCreateCert(dir, certFile, keyFile string) (*Cert, error) {
	if certFile == "" && keyFile == "" {
		certFile = filepath.Join(dir, "agent-hub.crt")
		keyFile = filepath.Join(dir, "agent-hub.key")
		if _, err := os.Stat(certFile); errors.Is(err, os.ErrNotExist) {
			if err := generateSelfSigned(dir, certFile, keyFile); err != nil {
				return nil, fmt.Errorf("generate agent TLS cert: %w", err)
			}
		}
	} else if certFile == "" || keyFile == "" {
		return nil, errors.New("both agent TLS cert and key must be set")
	}

	pair, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, fmt.Errorf("load agent TLS cert: %w", err)
	}
	return &Cert{TLS: pair, Fingerprint: Fingerprint(pair.Certificate[0])}, nil
}

// Fingerprint returns the lowercase hex SHA-256 of a DER certificate.
func Fingerprint(der []byte) string {
	sum := sha256.Sum256(der)
	return hex.EncodeToString(sum[:])
}

func generateSelfSigned(dir, certFile, keyFile string) error {
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return err
	}
	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 127))
	if err != nil {
		return err
	}
	tmpl := &x509.Certificate{
		SerialNumber: serial,
		Subject:      pkix.Name{CommonName: "vps-pilot-hub"},
		NotBefore:    time.Now().Add(-time.Hour),
		// Agents pin the fingerprint rather than validating a chain, so a long
		// validity avoids forcing re-enrollment of every agent.
		NotAfter:    time.Now().AddDate(20, 0, 0),
		KeyUsage:    x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:    []string{"vps-pilot-hub"},
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		return err
	}
	keyDER, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return err
	}
	if err := os.WriteFile(keyFile, pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyDER}), 0600); err != nil {
		return err
	}
	return os.WriteFile(certFile, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), 0644)
}
