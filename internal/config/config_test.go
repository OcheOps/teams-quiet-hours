package config

import (
	"testing"
	"time"
)

func TestValidate(t *testing.T) {
	cfg := Default()
	if err := Validate(cfg); err != nil {
		t.Fatalf("default config should validate: %v", err)
	}

	cfg.Start = "9:00"
	if err := Validate(cfg); err == nil {
		t.Fatal("expected non-zero-padded time to fail")
	}

	cfg = Default()
	cfg.Browser = "firefox"
	if err := Validate(cfg); err == nil {
		t.Fatal("expected unsupported browser to fail")
	}

	cfg = Default()
	cfg.Mode = "chaos"
	if err := Validate(cfg); err == nil {
		t.Fatal("expected unsupported mode to fail")
	}
}

func TestIsAllowedNow(t *testing.T) {
	cfg := Default()
	allowedTime := time.Date(2026, 5, 22, 10, 0, 0, 0, time.UTC)
	allowed, err := IsAllowedNow(cfg, allowedTime)
	if err != nil {
		t.Fatal(err)
	}
	if !allowed {
		t.Fatal("expected Friday 10:00 to be allowed")
	}

	blockedTime := time.Date(2026, 5, 23, 10, 0, 0, 0, time.UTC)
	allowed, err = IsAllowedNow(cfg, blockedTime)
	if err != nil {
		t.Fatal(err)
	}
	if allowed {
		t.Fatal("expected Saturday to be blocked")
	}
}
