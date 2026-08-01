package controlplane

import (
	"errors"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	HTTPAddr       string
	DatabaseURL    string
	AIURL          string
	AllowedOrigins []string
	AllowDevAuth   bool
	TrustedProxy   bool
	MaxBodyBytes   int64
	RatePerMinute  int
	ShutdownGrace  time.Duration
	AutoMigrate    bool
}

func LoadConfig() (Config, error) {
	c := Config{
		HTTPAddr: env("LORELINE_GATEWAY_ADDR", ":8080"), DatabaseURL: os.Getenv("DATABASE_URL"),
		AIURL: env("LORELINE_AI_URL", "http://intelligence:8090"), AllowedOrigins: split(env("LORELINE_ALLOWED_ORIGINS", "http://localhost:3000")),
		AllowDevAuth: boolean("LORELINE_ALLOW_DEV_AUTH", false), TrustedProxy: boolean("LORELINE_TRUST_PROXY", false),
		MaxBodyBytes: int64(integer("LORELINE_MAX_BODY_BYTES", 2<<20)), RatePerMinute: integer("LORELINE_RATE_PER_MINUTE", 120), ShutdownGrace: duration("LORELINE_SHUTDOWN_GRACE", 15*time.Second),
		AutoMigrate: boolean("LORELINE_AUTO_MIGRATE", false),
	}
	if c.DatabaseURL == "" {
		return Config{}, errors.New("DATABASE_URL is required; use LORELINE_DEMO_MODE=true only with the demo command")
	}
	if c.RatePerMinute < 1 || c.MaxBodyBytes < 1024 {
		return Config{}, errors.New("invalid rate or body size configuration")
	}
	return c, nil
}
func env(k, f string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return f
}
func split(v string) []string {
	out := []string{}
	for _, x := range strings.Split(v, ",") {
		if x = strings.TrimSpace(x); x != "" {
			out = append(out, x)
		}
	}
	return out
}
func boolean(k string, f bool) bool {
	v := os.Getenv(k)
	if v == "" {
		return f
	}
	b, e := strconv.ParseBool(v)
	return e == nil && b
}
func integer(k string, f int) int {
	v := os.Getenv(k)
	if v == "" {
		return f
	}
	n, e := strconv.Atoi(v)
	if e != nil {
		return f
	}
	return n
}
func duration(k string, f time.Duration) time.Duration {
	v := os.Getenv(k)
	if v == "" {
		return f
	}
	d, e := time.ParseDuration(v)
	if e != nil {
		return f
	}
	return d
}
