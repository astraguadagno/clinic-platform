package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"clinic-platform/services/directory-service/internal/directory"
	internalhttp "clinic-platform/services/directory-service/internal/http"
)

func main() {
	port := envOrDefault("PORT", "8081")
	appEnv := envOrDefault("APP_ENV", "local")
	dsn := postgresDSN()

	repository, err := directory.OpenDB(dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer repository.Close()

	repo := directory.NewRepository(repository)
	if appEnv == "local" || appEnv == "docker" {
		if err := repo.BootstrapAccess(context.Background(), directory.BootstrapAccessParams{
			AdminEmail:    envOrDefault("BOOTSTRAP_ADMIN_EMAIL", "admin@clinic.local"),
			AdminPassword: envOrDefault("BOOTSTRAP_ADMIN_PASSWORD", "admin123"),
		}); err != nil {
			log.Fatal(err)
		}
	}

	handler := internalhttp.NewServer(internalhttp.Config{
		ServiceName:  "directory-service",
		Version:      "v0.1.0",
		Environment:  appEnv,
		AuthTokenTTL: 24 * time.Hour,
	}, repo)

	log.Printf("starting directory-service on :%s", port)

	if err := http.ListenAndServe(":"+port, handler); err != nil {
		log.Fatal(err)
	}
}

func envOrDefault(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	return value
}

func postgresDSN() string {
	host := envOrDefault("DB_HOST", "localhost")
	port := envOrDefault("DB_PORT", "5433")
	name := envOrDefault("DB_NAME", "directory")
	user := envOrDefault("DB_USER", "directory")
	password := envOrDefault("DB_PASSWORD", "directory")
	sslMode := envOrDefault("DB_SSLMODE", "disable")

	return "postgres://" + user + ":" + password + "@" + host + ":" + port + "/" + name + "?sslmode=" + sslMode
}
