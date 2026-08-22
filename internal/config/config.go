package config

import "os"

type Config struct {
	DatabaseURL string
	Port        string
	SessionTTL  string
}

func Load() Config {
	db := os.Getenv("Dianchi_DATABASE_URL")
	if db == "" {
		db = "file:dianchi.db"
	}
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	ttl := os.Getenv("SESSION_TTL")
	if ttl == "" {
		ttl = "8h"
	}
	return Config{DatabaseURL: db, Port: port, SessionTTL: ttl}
}
