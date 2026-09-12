package config

import (
	"os"
	"path/filepath"
	"testing"
)

const devYAML = "config_dev.yaml"

func withConfigDir(t *testing.T, dir string) {
	t.Helper()
	old := configDir
	configDir = dir
	t.Cleanup(func() { configDir = old })
}

func TestLoad(t *testing.T) {
	withConfigDir(t, ".")
	t.Setenv("APP_ENV", "dev")
	t.Setenv("PORT", "")
	t.Setenv("MONGO_URI", "")
	t.Setenv("MONGO_DATABASE", "")
	t.Setenv("JWT_SECRET", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Env != "dev" || cfg.Port != "8080" || cfg.JWTSecret != "dev-only-jwt-secret" {
		t.Errorf("Load() = %+v", cfg)
	}
}

func TestLoadDefaultEnv(t *testing.T) {
	withConfigDir(t, ".")
	t.Setenv("APP_ENV", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Env != "dev" {
		t.Errorf("Env = %q, want dev default", cfg.Env)
	}
}

func TestLoadDevDefaults(t *testing.T) {
	t.Setenv("APP_ENV", "")
	t.Setenv("PORT", "")
	t.Setenv("MONGO_URI", "")
	t.Setenv("MONGO_DATABASE", "")
	t.Setenv("JWT_SECRET", "")

	cfg, err := LoadFrom(devYAML, "dev")
	if err != nil {
		t.Fatalf("LoadFrom() error = %v", err)
	}
	if cfg.Env != "dev" {
		t.Errorf("Env = %q, want dev", cfg.Env)
	}
	if cfg.Port != "8080" {
		t.Errorf("Port = %q, want 8080", cfg.Port)
	}
	if cfg.MongoDatabase != "userdb" {
		t.Errorf("MongoDatabase = %q, want userdb", cfg.MongoDatabase)
	}
	if cfg.MongoURI != "mongodb://localhost:27017" {
		t.Errorf("MongoURI = %q, want dev secret", cfg.MongoURI)
	}
	if cfg.JWTSecret != "dev-only-jwt-secret" {
		t.Errorf("JWTSecret = %q, want dev secret", cfg.JWTSecret)
	}
}

func TestLoadDefaultsEnvEmpty(t *testing.T) {
	t.Setenv("APP_ENV", "")

	cfg, err := LoadFrom(devYAML, "")
	if err != nil {
		t.Fatalf("LoadFrom() error = %v", err)
	}
	if cfg.Env != "dev" {
		t.Errorf("Env = %q, want dev fallback", cfg.Env)
	}
}

func TestLoadEnvironmentOverrides(t *testing.T) {
	t.Setenv("PORT", "9090")
	t.Setenv("MONGO_URI", "mongodb://override:27017")
	t.Setenv("MONGO_DATABASE", "otherdb")
	t.Setenv("JWT_SECRET", "override-secret")

	cfg, err := LoadFrom(devYAML, "dev")
	if err != nil {
		t.Fatalf("LoadFrom() error = %v", err)
	}
	if cfg.Port != "9090" {
		t.Errorf("Port = %q, want 9090", cfg.Port)
	}
	if cfg.MongoURI != "mongodb://override:27017" {
		t.Errorf("MongoURI = %q, want override", cfg.MongoURI)
	}
	if cfg.MongoDatabase != "otherdb" {
		t.Errorf("MongoDatabase = %q, want otherdb", cfg.MongoDatabase)
	}
	if cfg.JWTSecret != "override-secret" {
		t.Errorf("JWTSecret = %q, want override", cfg.JWTSecret)
	}
}

func TestLoadMissingFile(t *testing.T) {
	if _, err := LoadFrom("config_nope.yaml", "nope"); err == nil {
		t.Fatal("LoadFrom() with missing file: want error, got nil")
	}
}

func TestLoadInvalidEnv(t *testing.T) {
	if _, err := LoadFrom(devYAML, "../evil"); err == nil {
		t.Fatal("LoadFrom() with invalid env: want error, got nil")
	}
}

func TestLoadMalformedYAML(t *testing.T) {
	dir := t.TempDir()
	bad := filepath.Join(dir, "bad.yaml")
	if err := os.WriteFile(bad, []byte(":\n  - not: [valid"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadFrom(bad, "dev"); err == nil {
		t.Fatal("LoadFrom() with malformed yaml: want error, got nil")
	}
}

func TestYAMLOnlyConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "custom.yaml")
	content := "server:\n  port: \"7000\"\ndatabase:\n  name: customdb\n  mongo_uri: mongodb://yaml:27017\njwt:\n  secret: yaml-secret\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	t.Setenv("PORT", "")
	t.Setenv("MONGO_URI", "")
	t.Setenv("MONGO_DATABASE", "")
	t.Setenv("JWT_SECRET", "")

	cfg, err := LoadFrom(path, "staging")
	if err != nil {
		t.Fatalf("LoadFrom() error = %v", err)
	}
	if cfg.Port != "7000" {
		t.Errorf("Port = %q, want 7000", cfg.Port)
	}
	if cfg.MongoDatabase != "customdb" {
		t.Errorf("MongoDatabase = %q, want customdb", cfg.MongoDatabase)
	}
	// No secret registered for "staging", so YAML values must surface.
	if cfg.MongoURI != "mongodb://yaml:27017" {
		t.Errorf("MongoURI = %q, want YAML value", cfg.MongoURI)
	}
	if cfg.JWTSecret != "yaml-secret" {
		t.Errorf("JWTSecret = %q, want YAML value", cfg.JWTSecret)
	}
}

func TestSecretsRegistry(t *testing.T) {
	registerSecrets("staging", Secret{MongoURI: "mongo://staging", JWTSecret: "staging-secret"})

	sec, ok := secretsFor("staging")
	if !ok {
		t.Fatal("secretsFor(staging) = not found")
	}
	if sec.MongoURI != "mongo://staging" || sec.JWTSecret != "staging-secret" {
		t.Errorf("secret = %+v", sec)
	}

	if _, ok := secretsFor("nonexistent-env"); ok {
		t.Error("secretsFor(nonexistent-env) = found, want not found")
	}
}

func TestFirstNonEmpty(t *testing.T) {
	if got := firstNonEmpty("", "  ", "x"); got != "x" {
		t.Errorf("firstNonEmpty = %q, want x", got)
	}
	if got := firstNonEmpty(""); got != "" {
		t.Errorf("firstNonEmpty() = %q, want empty", got)
	}
}

func TestYAMLName(t *testing.T) {
	if got := yamlName("prod"); got != "config_prod.yaml" {
		t.Errorf("yamlName(prod) = %q", got)
	}
}
