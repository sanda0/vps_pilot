package agentauth

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestJoinTokenRoundTrip(t *testing.T) {
	fp := Fingerprint([]byte("cert"))
	token, hash, err := NewJoinToken(fp)
	if err != nil {
		t.Fatal(err)
	}
	secret, gotFP, err := ParseJoinToken(token)
	if err != nil {
		t.Fatal(err)
	}
	if gotFP != fp {
		t.Errorf("fingerprint %q, want %q", gotFP, fp)
	}
	if !HashEqual(secret, hash) || HashEqual(secret+"x", hash) {
		t.Error("hash comparison wrong")
	}
	if strings.Contains(hash, secret) {
		t.Error("hash contains secret")
	}
}

func TestParseJoinTokenRejectsMalformed(t *testing.T) {
	for _, tok := range []string{"", "vpsp1", "vpsp2.abc." + strings.Repeat("a", 64), "vpsp1..x", "vpsp1.abc.short"} {
		if _, _, err := ParseJoinToken(tok); err == nil {
			t.Errorf("%q: expected error", tok)
		}
	}
}

func TestLoadOrCreateCertIsStable(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "tls")
	first, err := LoadOrCreateCert(dir, "", "")
	if err != nil {
		t.Fatal(err)
	}
	second, err := LoadOrCreateCert(dir, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if first.Fingerprint != second.Fingerprint || len(first.Fingerprint) != 64 {
		t.Errorf("fingerprint changed across restarts: %s vs %s", first.Fingerprint, second.Fingerprint)
	}
}
