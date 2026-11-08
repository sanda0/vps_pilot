package app

import (
	"bytes"
	"log"
	"strings"
	"testing"

	"github.com/sanda0/vps_pilot/internal/utils"
)

// WarnTokenConfig runs once at hub startup and reports each fallback once.
func TestWarnTokenConfigLogsFallbacks(t *testing.T) {
	t.Setenv("TOKEN_LIFESPAN", "soon")
	t.Setenv("TOKEN_SECRET", "")
	var buf bytes.Buffer
	prev := log.Writer()
	log.SetOutput(&buf)
	t.Cleanup(func() { log.SetOutput(prev) })

	utils.WarnTokenConfig()
	out := buf.String()
	for _, want := range []string{
		"Invalid TOKEN_LIFESPAN value 'soon', using default 60 minutes",
		"TOKEN_SECRET not set, using default",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("startup log %q missing %q", out, want)
		}
	}
}

func TestWarnTokenConfigSilentWhenSet(t *testing.T) {
	t.Setenv("TOKEN_LIFESPAN", "120")
	t.Setenv("TOKEN_SECRET", "s3cret")
	var buf bytes.Buffer
	prev := log.Writer()
	log.SetOutput(&buf)
	t.Cleanup(func() { log.SetOutput(prev) })

	utils.WarnTokenConfig()
	if buf.Len() > 0 {
		t.Errorf("WarnTokenConfig logged with valid config: %q", buf.String())
	}
}
