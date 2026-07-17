// Package config loads application configuration from environment variables.
package config

import (
	"fmt"
	"log"
	"net/url"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Port   string
	DBPath string

	// DBDriver selects the storage backend: "sqlite" or "mongo".
	DBDriver string

	// Mongo connection settings, used when DBDriver is "mongo".
	MongoUsername string
	MongoPassword string
	MongoHost     string
	MongoAppName  string
	MongoDatabase string

	// Auth gates the whole app behind a single username/password.
	AuthUsername  string
	AuthPassword  string
	SessionSecret string

	// Google reCAPTCHA v2 checkbox on the login page.
	RecaptchaSiteKey   string
	RecaptchaSecretKey string
}

func Load() Config {
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file found, using environment variables")
	}

	return Config{
		Port:     getEnv("PORT", "8080"),
		DBPath:   getEnv("DB_PATH", "data/bookmarks.db"),
		DBDriver: getEnv("DB_DRIVER", "sqlite"),

		MongoUsername: getEnv("MONGODB_USERNAME", ""),
		MongoPassword: getEnv("MONGODB_PASSWORD", ""),
		MongoHost:     getEnv("MONGODB_HOST", ""),
		MongoAppName:  getEnv("MONGODB_APP_NAME", "Cluster0"),
		MongoDatabase: getEnv("MONGODB_DATABASE", "bookmarks"),

		AuthUsername:  getEnv("AUTH_USERNAME", ""),
		AuthPassword:  getEnv("AUTH_PASSWORD", ""),
		SessionSecret: getEnv("SESSION_SECRET", ""),

		RecaptchaSiteKey:   getEnv("RECAPTCHA_SITE_KEY", ""),
		RecaptchaSecretKey: getEnv("RECAPTCHA_SECRET_KEY", ""),
	}
}

// MongoURI builds an Atlas mongodb+srv connection string from the
// individual configured parameters.
func (c Config) MongoURI() string {
	return fmt.Sprintf("mongodb+srv://%s:%s@%s/?appName=%s",
		url.QueryEscape(c.MongoUsername),
		url.QueryEscape(c.MongoPassword),
		c.MongoHost,
		url.QueryEscape(c.MongoAppName),
	)
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
