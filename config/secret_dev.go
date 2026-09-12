package config

// Development secrets. Safe to commit — development-only values that require no
// infrastructure and never reach production.
func init() {
	registerSecrets("dev", Secret{
		MongoURI:  "mongodb://localhost:27017",
		JWTSecret: "dev-only-jwt-secret",
	})
}
