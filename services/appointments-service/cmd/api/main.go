package main

import (
	"log"
	"net/http"
	"os"

	"clinic-platform/services/appointments-service/internal/appointments"
	internalhttp "clinic-platform/services/appointments-service/internal/http"
)

func main() {
	port := envOrDefault("PORT", "8082")
	appEnv := envOrDefault("APP_ENV", "local")
	dsn := postgresDSN()

	db, err := appointments.OpenDB(dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	handler := internalhttp.NewServer(internalhttp.Config{
		ServiceName: "appointments-service",
		Version:     "v0.1.0",
		Environment: appEnv,
	}, appointments.NewRepository(db))

	log.Printf("starting appointments-service on :%s", port)

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
	port := envOrDefault("DB_PORT", "5434")
	name := envOrDefault("DB_NAME", "appointments")
	user := envOrDefault("DB_USER", "appointments")
	password := envOrDefault("DB_PASSWORD", "appointments")
	sslMode := envOrDefault("DB_SSLMODE", "disable")

	return "postgres://" + user + ":" + password + "@" + host + ":" + port + "/" + name + "?sslmode=" + sslMode
}
