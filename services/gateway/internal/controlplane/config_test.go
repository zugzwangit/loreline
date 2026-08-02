package controlplane

import (
	"strings"
	"testing"
)

func baseConfigEnv(t *testing.T) {
	t.Helper()
	t.Setenv("DATABASE_URL", "postgres://localhost/loreline")
	t.Setenv("LORELINE_ENVIRONMENT", "test")
	t.Setenv("LORELINE_AI_URL", "http://127.0.0.1:8090")
}

func TestLoadConfigRejectsMalformedBoolean(t *testing.T) {
	baseConfigEnv(t)
	t.Setenv("LORELINE_AUTO_MIGRATE", "sometimes")
	_, err := LoadConfig()
	if err == nil || !strings.Contains(err.Error(), "true or false") {
		t.Fatalf("expected a boolean validation error, got %v", err)
	}
}

func TestLoadConfigRejectsMalformedNumber(t *testing.T) {
	baseConfigEnv(t)
	t.Setenv("LORELINE_RATE_PER_MINUTE", "many")
	_, err := LoadConfig()
	if err == nil || !strings.Contains(err.Error(), "must be an integer") {
		t.Fatalf("expected an integer validation error, got %v", err)
	}
}

func TestLoadConfigRejectsDevelopmentAuthenticationInProduction(t *testing.T) {
	baseConfigEnv(t)
	t.Setenv("LORELINE_ENVIRONMENT", "production")
	t.Setenv("LORELINE_ALLOW_DEV_AUTH", "true")
	_, err := LoadConfig()
	if err == nil || !strings.Contains(err.Error(), "forbidden") {
		t.Fatalf("expected production authentication guard, got %v", err)
	}
}

func TestLoadConfigRejectsUnsafeAIURL(t *testing.T) {
	baseConfigEnv(t)
	t.Setenv("LORELINE_AI_URL", "https://user:secret@example.com")
	_, err := LoadConfig()
	if err == nil || !strings.Contains(err.Error(), "without embedded credentials") {
		t.Fatalf("expected URL validation error, got %v", err)
	}
}

func TestLoadConfigAcceptsTestConfiguration(t *testing.T) {
	baseConfigEnv(t)
	t.Setenv("LORELINE_ALLOW_DEV_AUTH", "true")
	cfg, err := LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.AllowDevAuth || cfg.Environment != "test" {
		t.Fatalf("unexpected config: %+v", cfg)
	}
}
