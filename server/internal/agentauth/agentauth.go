// Package agentauth holds the credentials used on the agent channel:
// one-time join tokens, long-lived node keys, and the hub's TLS certificate
// whose fingerprint agents pin. See docs/PROTOCOL.md.
package agentauth

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"
)

const (
	// JoinTokenPrefix versions the join token format.
	JoinTokenPrefix = "vpsp1"
	// JoinTokenTTL is how long a join token stays valid.
	JoinTokenTTL = time.Hour
)

var ErrMalformedJoinToken = errors.New("malformed join token")

// randomString returns n random bytes, base64url-encoded without padding.
func randomString(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// Hash returns the hex SHA-256 of a secret. Only hashes are stored.
func Hash(secret string) string {
	sum := sha256.Sum256([]byte(secret))
	return hex.EncodeToString(sum[:])
}

// HashEqual compares a secret against a stored hash in constant time.
func HashEqual(secret, storedHash string) bool {
	return subtle.ConstantTimeCompare([]byte(Hash(secret)), []byte(storedHash)) == 1
}

// NewJoinToken returns a join token embedding the hub cert fingerprint,
// plus the hash of its secret part for storage.
// Format: vpsp1.<secret>.<sha256 fingerprint hex>
func NewJoinToken(fingerprint string) (token string, secretHash string, err error) {
	secret, err := randomString(32)
	if err != nil {
		return "", "", err
	}
	return fmt.Sprintf("%s.%s.%s", JoinTokenPrefix, secret, fingerprint), Hash(secret), nil
}

// ParseJoinToken splits a join token into its secret and fingerprint.
func ParseJoinToken(token string) (secret, fingerprint string, err error) {
	parts := strings.Split(strings.TrimSpace(token), ".")
	if len(parts) != 3 || parts[0] != JoinTokenPrefix || parts[1] == "" || len(parts[2]) != 64 {
		return "", "", ErrMalformedJoinToken
	}
	return parts[1], parts[2], nil
}

// NewNodeCredentials returns a fresh agent ID and node key.
func NewNodeCredentials() (agentID, nodeKey string, err error) {
	id := make([]byte, 16)
	if _, err := rand.Read(id); err != nil {
		return "", "", err
	}
	id[6] = (id[6] & 0x0f) | 0x40 // UUID v4
	id[8] = (id[8] & 0x3f) | 0x80
	agentID = fmt.Sprintf("%x-%x-%x-%x-%x", id[0:4], id[4:6], id[6:8], id[8:10], id[10:])

	nodeKey, err = randomString(32)
	if err != nil {
		return "", "", err
	}
	return agentID, nodeKey, nil
}

// SplitConnectToken splits the "connected" token "<agent_id>:<node_key>".
func SplitConnectToken(token string) (agentID, nodeKey string, ok bool) {
	agentID, nodeKey, ok = strings.Cut(token, ":")
	return agentID, nodeKey, ok && agentID != "" && nodeKey != ""
}
