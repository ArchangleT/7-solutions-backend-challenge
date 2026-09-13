// Package config loads runtime configuration for the server. Non-secret
// settings come from a per-environment YAML file (config_{env}.yaml), while
// secrets are provided by build-time Go files named secret_{env}.go and picked
// at runtime by the active environment. Environment variables always take
// precedence so the same binary works in any deployment.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	defaultEnv           = "dev"
	defaultPort          = "8080"
	defaultGRPCPort      = "50051"
	defaultMongoDatabase = "userdb"
)

var envPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]*$`)

// Secret holds values that must not be committed to source control. Files
// named secret_{env}.go register one Secret per environment at init time.
type Secret struct {
	MongoURI  string
	JWTSecret string
}

// Config holds all runtime configuration for the server.
type Config struct {
	Env            string
	Port           string
	GRPCPort       string
	MongoURI       string
	MongoDatabase  string
	JWTSecret      string
	TrustProxy     bool
	TrustedProxies []string
}

// yamlConfig mirrors the shape of a config_{env}.yaml file.
type yamlConfig struct {
	Server struct {
		Port           string   `yaml:"port"`
		GRPCPort       string   `yaml:"grpc_port"`
		TrustProxy     bool     `yaml:"trust_proxy"`
		TrustedProxies []string `yaml:"trusted_proxies"`
	} `yaml:"server"`
	Database struct {
		URI  string `yaml:"mongo_uri"`
		Name string `yaml:"name"`
	} `yaml:"database"`
	JWT struct {
		Secret string `yaml:"secret"`
	} `yaml:"jwt"`
}

// secretByEnv is filled by the init() functions of the secret_{env}.go files.
var secretByEnv = map[string]Secret{}

// registerSecrets associates a Secret with an environment so Load can find it
// without the file having been imported explicitly.
func registerSecrets(env string, s Secret) {
	secretByEnv[env] = s
}

// secretsFor returns the registered Secret for env, if any.
func secretsFor(env string) (Secret, bool) {
	s, ok := secretByEnv[env]
	return s, ok
}

// configDir is the directory holding the per-env YAML files, resolved relative
// to the working directory (the repo root when running `go run ./cmd/api`).
var configDir = "config"

// Load reads the YAML config for the APP_ENV environment (defaulting to
// "dev"), overlays the registered secrets for that environment, then lets
// environment variables override everything.
func Load() (Config, error) {
	env := strings.TrimSpace(os.Getenv("APP_ENV"))
	if env == "" {
		env = defaultEnv
	}
	return load(filepath.Join(configDir, yamlName(env)), env)
}

// LoadFrom reads configuration from an explicit YAML path. It exists for tests
// and custom deployments that keep config files outside the default layout.
func LoadFrom(path, env string) (Config, error) {
	env = strings.TrimSpace(env)
	if env == "" {
		env = defaultEnv
	}
	return load(path, env)
}

func load(path, env string) (Config, error) {
	if !envPattern.MatchString(env) {
		return Config{}, fmt.Errorf("invalid environment %q", env)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read config file %s: %w", path, err)
	}
	var y yamlConfig
	if err := yaml.Unmarshal(raw, &y); err != nil {
		return Config{}, fmt.Errorf("parse config file %s: %w", path, err)
	}

	cfg := Config{Env: env}

	sec, ok := secretsFor(env)
	if ok {
		cfg.MongoURI, cfg.JWTSecret = sec.MongoURI, sec.JWTSecret
	}

	// YAML supplies the non-secret defaults.
	cfg.Port = y.Server.Port
	cfg.GRPCPort = y.Server.GRPCPort
	cfg.MongoDatabase = y.Database.Name
	cfg.TrustProxy = y.Server.TrustProxy
	cfg.TrustedProxies = y.Server.TrustedProxies
	if len(cfg.TrustedProxies) == 0 {
		cfg.TrustedProxies = nil
	}

	// Environment variables take precedence over both files.
	cfg.Port = getenv("PORT", firstNonEmpty(cfg.Port, defaultPort))
	cfg.GRPCPort = getenv("GRPC_PORT", firstNonEmpty(cfg.GRPCPort, defaultGRPCPort))
	cfg.MongoDatabase = getenv("MONGO_DATABASE", firstNonEmpty(cfg.MongoDatabase, defaultMongoDatabase))
	cfg.MongoURI = getenv("MONGO_URI", firstNonEmpty(cfg.MongoURI, y.Database.URI))
	cfg.JWTSecret = getenv("JWT_SECRET", firstNonEmpty(cfg.JWTSecret, y.JWT.Secret))

	// Secrets are mandatory: an empty URI or JWT secret would fail later in a
	// confusing way (or worse, sign tokens with an empty key). Fail fast.
	if strings.TrimSpace(cfg.MongoURI) == "" {
		return Config{}, fmt.Errorf("missing required config: MONGO_URI is not set (via YAML, secret_{env}.go or env var)")
	}
	if strings.TrimSpace(cfg.JWTSecret) == "" {
		return Config{}, fmt.Errorf("missing required config: JWT_SECRET is not set (via YAML, secret_{env}.go or env var)")
	}

	return cfg, nil
}

func yamlName(env string) string {
	return "config_" + env + ".yaml"
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
