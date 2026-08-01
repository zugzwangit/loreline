package controlplane

import (
	"os"
	"testing"
	"time"
)

func TestRateLimiter(t *testing.T) {
	l := newRateLimiter(2)
	if !l.Allow("a") || !l.Allow("a") {
		t.Fatal("initial burst should pass")
	}
	if l.Allow("a") {
		t.Fatal("third request should be limited")
	}
}
func TestRoleHierarchy(t *testing.T) {
	if !validateRole("owner", "admin") || !validateRole("reviewer", "viewer") || validateRole("viewer", "reviewer") {
		t.Fatal("role hierarchy is invalid")
	}
}
func TestConfigRequiresDatabase(t *testing.T) {
	old := os.Getenv("DATABASE_URL")
	os.Unsetenv("DATABASE_URL")
	defer os.Setenv("DATABASE_URL", old)
	if _, err := LoadConfig(); err == nil {
		t.Fatal("missing database must fail closed")
	}
}
func TestDurationFallback(t *testing.T) {
	if duration("UNSET_LORELINE_TEST", 3*time.Second) != 3*time.Second {
		t.Fatal("duration fallback failed")
	}
}
