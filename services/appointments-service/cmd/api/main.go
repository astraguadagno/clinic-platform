package main

import (
	"log"
	"net/http"
	"os"

	internalhttp "clinic-platform/services/appointments-service/internal/http"
)

func main() {
	port := envOrDefault("PORT", "8082")
	appEnv := envOrDefault("APP_ENV", "local")

	handler := internalhttp.NewServer(internalhttp.Config{
		ServiceName: "appointments-service",
		Version:     "v0.1.0",
		Environment: appEnv,
	})

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
