package config

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

const devYAML = "config_dev.yaml"

func withConfigDir(t *testing.T, dir string) {
	t.Helper()
	old := configDir
	configDir = dir
	t.Cleanup(func() { configDir = old })
}

// clearRuntimeEnv resets every runtime env var to empty so tests start from a
// clean environment regardless of the surrounding shell.
func clearRuntimeEnv(t *testing.T) {
	t.Helper()
	for _, v := range []string{"APP_ENV", "PORT", "GRPC_PORT", "MONGO_URI", "MONGO_DATABASE", "JWT_SECRET"} {
		t.Setenv(v, "")
	}
}

func TestLoad(t *testing.T) {
	withConfigDir(t, ".")
	tests := []struct {
		name    string
		env     string
		wantEnv string
	}{
		{name: "defaults to dev", env: "", wantEnv: "dev"},
		{name: "explicit dev", env: "dev", wantEnv: "dev"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearRuntimeEnv(t)
			t.Setenv("APP_ENV", tt.env)

			cfg, err := Load()
			if err != nil {
				t.Fatalf("Load() error = %v", err)
			}
			if cfg.Env != tt.wantEnv {
				t.Errorf("Load() Env = %q, want %q", cfg.Env, tt.wantEnv)
			}
		})
	}
}

func TestLoadDefaults(t *testing.T) {
	wantDev := Config{
		Env:           "dev",
		Port:          "8080",
		GRPCPort:      "50051",
		MongoURI:      "mongodb://localhost:27017",
		MongoDatabase: "userdb",
		JWTSecret:     "dev-only-jwt-secret",
	}

	tests := []struct {
		name string
		env  string
		want Config
	}{
		{name: "dev yaml and secrets", env: "dev", want: wantDev},
		{name: "empty env falls back to dev", env: "", want: wantDev},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearRuntimeEnv(t)

			cfg, err := LoadFrom(devYAML, tt.env)
			if err != nil {
				t.Fatalf("LoadFrom() error = %v", err)
			}
			if !reflect.DeepEqual(cfg, tt.want) {
				t.Errorf("LoadFrom() = %+v, want %+v", cfg, tt.want)
			}
		})
	}
}

func TestLoadEnvOverrides(t *testing.T) {
	devDefault := Config{
		Env:           "dev",
		Port:          "8080",
		GRPCPort:      "50051",
		MongoURI:      "mongodb://localhost:27017",
		MongoDatabase: "userdb",
		JWTSecret:     "dev-only-jwt-secret",
	}

	tests := []struct {
		name string
		env  map[string]string
		want Config
	}{
		{
			name: "port override",
			env:  map[string]string{"PORT": "9090"},
			want: Config{Env: devDefault.Env, Port: "9090", GRPCPort: devDefault.GRPCPort, MongoURI: devDefault.MongoURI, MongoDatabase: devDefault.MongoDatabase, JWTSecret: devDefault.JWTSecret},
		},
		{
			name: "grpc port override",
			env:  map[string]string{"GRPC_PORT": "50052"},
			want: Config{Env: devDefault.Env, Port: devDefault.Port, GRPCPort: "50052", MongoURI: devDefault.MongoURI, MongoDatabase: devDefault.MongoDatabase, JWTSecret: devDefault.JWTSecret},
		},
		{
			name: "database overrides",
			env:  map[string]string{"MONGO_URI": "mongodb://override:27017", "MONGO_DATABASE": "otherdb"},
			want: Config{Env: devDefault.Env, Port: devDefault.Port, GRPCPort: devDefault.GRPCPort, MongoURI: "mongodb://override:27017", MongoDatabase: "otherdb", JWTSecret: devDefault.JWTSecret},
		},
		{
			name: "jwt secret override",
			env:  map[string]string{"JWT_SECRET": "override-secret"},
			want: Config{Env: devDefault.Env, Port: devDefault.Port, GRPCPort: devDefault.GRPCPort, MongoURI: devDefault.MongoURI, MongoDatabase: devDefault.MongoDatabase, JWTSecret: "override-secret"},
		},
		{
			name: "all overrides win",
			env: map[string]string{
				"PORT":           "9090",
				"GRPC_PORT":      "50052",
				"MONGO_URI":      "mongodb://override:27017",
				"MONGO_DATABASE": "otherdb",
				"JWT_SECRET":     "override-secret",
			},
			want: Config{
				Env:           "dev",
				Port:          "9090",
				GRPCPort:      "50052",
				MongoURI:      "mongodb://override:27017",
				MongoDatabase: "otherdb",
				JWTSecret:     "override-secret",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearRuntimeEnv(t)
			for k, v := range tt.env {
				t.Setenv(k, v)
			}

			cfg, err := LoadFrom(devYAML, "dev")
			if err != nil {
				t.Fatalf("LoadFrom() error = %v", err)
			}
			if !reflect.DeepEqual(cfg, tt.want) {
				t.Errorf("LoadFrom() = %+v, want %+v", cfg, tt.want)
			}
		})
	}
}

func TestLoadErrors(t *testing.T) {
	tests := []struct {
		name string
		env  string
		prep func(t *testing.T) string
	}{
		{name: "missing file", env: "nope", prep: func(*testing.T) string { return "config_nope.yaml" }},
		{name: "invalid environment", env: "../evil", prep: func(*testing.T) string { return devYAML }},
		{
			name: "malformed yaml",
			env:  "dev",
			prep: func(t *testing.T) string {
				p := filepath.Join(t.TempDir(), "bad.yaml")
				if err := os.WriteFile(p, []byte(":\n  - not: [valid"), 0o600); err != nil {
					t.Fatal(err)
				}
				return p
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearRuntimeEnv(t)
			if _, err := LoadFrom(tt.prep(t), tt.env); err == nil {
				t.Fatal("LoadFrom() error = nil, want error")
			}
		})
	}
}

func TestLoadYAMLOnly(t *testing.T) {
	path := filepath.Join(t.TempDir(), "custom.yaml")
	content := "server:\n  port: \"7000\"\ndatabase:\n  name: customdb\n  mongo_uri: mongodb://yaml:27017\njwt:\n  secret: yaml-secret\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name string
		env  string
	}{
		{name: "no secret registered", env: "staging"},
		{name: "unregistered environment", env: "unknown"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearRuntimeEnv(t)
			want := Config{
				Env:           tt.env,
				Port:          "7000",
				GRPCPort:      "50051",
				MongoURI:      "mongodb://yaml:27017",
				MongoDatabase: "customdb",
				JWTSecret:     "yaml-secret",
			}

			cfg, err := LoadFrom(path, tt.env)
			if err != nil {
				t.Fatalf("LoadFrom() error = %v", err)
			}
			if !reflect.DeepEqual(cfg, want) {
				t.Errorf("LoadFrom() = %+v, want %+v", cfg, want)
			}
		})
	}
}

// TestLoadProductionYAML guards the production YAML contract: the file must
// parse, the trust_proxy/trusted_proxies fields must surface, and runtime
// secrets must win over the (intentionally empty) prod file. Secrets are
// provided via env vars so the test stays deterministic even where the
// git-ignored secret_prod.go is absent (e.g. CI).
func TestLoadProductionYAML(t *testing.T) {
	clearRuntimeEnv(t)
	const (
		envURI = "mongodb://user:pass@prod-host:27017"
		envJWT = "prod-jwt-secret"
	)
	t.Setenv("MONGO_URI", envURI)
	t.Setenv("JWT_SECRET", envJWT)

	cfg, err := LoadFrom("config_prod.yaml", "prod")
	if err != nil {
		t.Fatalf("LoadFrom(config_prod.yaml, prod) error = %v", err)
	}

	if cfg.Env != "prod" {
		t.Errorf("Env = %q, want prod", cfg.Env)
	}
	if cfg.Port != "8080" {
		t.Errorf("Port = %q, want 8080", cfg.Port)
	}
	if cfg.GRPCPort != "50051" {
		t.Errorf("GRPCPort = %q, want 50051", cfg.GRPCPort)
	}
	if cfg.MongoDatabase != "userdb" {
		t.Errorf("MongoDatabase = %q, want userdb", cfg.MongoDatabase)
	}
	if cfg.TrustProxy {
		t.Errorf("TrustProxy = true, want false (default)")
	}
	if cfg.TrustedProxies != nil {
		t.Errorf("TrustedProxies = %v, want nil (empty slice normalized)", cfg.TrustedProxies)
	}
	// Environment variables must override both the empty file and any
	// registered secret file.
	if cfg.MongoURI != envURI {
		t.Errorf("MongoURI = %q, want %q (env override)", cfg.MongoURI, envURI)
	}
	if cfg.JWTSecret != envJWT {
		t.Errorf("JWTSecret = %q, want %q (env override)", cfg.JWTSecret, envJWT)
	}
}

func TestLoadMissingRequiredSecrets(t *testing.T) {
	clearRuntimeEnv(t)

	path := filepath.Join(t.TempDir(), "minimal.yaml")
	content := "server:\n  port: \"7000\"\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := LoadFrom(path, "no-secret-env"); err == nil {
		t.Fatal("LoadFrom() error = nil, want error for missing MONGO_URI/JWT_SECRET")
	}
}

func TestSecretsRegistry(t *testing.T) {
	registerSecrets("ci", Secret{MongoURI: "mongo://ci", JWTSecret: "ci-secret"})

	tests := []struct {
		name string
		env  string
		want Secret
		ok   bool
	}{
		{name: "registered env", env: "ci", want: Secret{MongoURI: "mongo://ci", JWTSecret: "ci-secret"}, ok: true},
		{name: "unregistered env", env: "not-registered", ok: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sec, ok := secretsFor(tt.env)
			if ok != tt.ok {
				t.Fatalf("secretsFor(%q) ok = %v, want %v", tt.env, ok, tt.ok)
			}
			if tt.ok && sec != tt.want {
				t.Errorf("secretsFor(%q) = %+v, want %+v", tt.env, sec, tt.want)
			}
		})
	}
}

func TestFirstNonEmpty(t *testing.T) {
	tests := []struct {
		name string
		vals []string
		want string
	}{
		{name: "skips empty and blank", vals: []string{"", "  ", "x"}, want: "x"},
		{name: "all empty", vals: []string{"", ""}, want: ""},
		{name: "no values", vals: nil, want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := firstNonEmpty(tt.vals...); got != tt.want {
				t.Errorf("firstNonEmpty() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestYAMLName(t *testing.T) {
	tests := []struct {
		env  string
		want string
	}{
		{env: "dev", want: "config_dev.yaml"},
		{env: "prod", want: "config_prod.yaml"},
		{env: "staging", want: "config_staging.yaml"},
	}
	for _, tt := range tests {
		if got := yamlName(tt.env); got != tt.want {
			t.Errorf("yamlName(%q) = %q, want %q", tt.env, got, tt.want)
		}
	}
}
