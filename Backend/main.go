package main

import (
	"file-system/handlers"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	"github.com/rs/cors"
)

func main() {
	mux := http.NewServeMux()
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	mux.HandleFunc("/upload", handlers.Upload)
	mux.HandleFunc("/upload/init", handlers.UploadInit)
	mux.HandleFunc("/upload/{fileId}", handlers.UploadChunk)
	mux.HandleFunc("/upload/{fileId}/complete", handlers.UploadComplete)
	mux.HandleFunc("/health", handlers.HealthCheck)
	url := os.Getenv("FRONTEND_URL")
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{url},
		AllowedMethods:   []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodOptions},
		AllowedHeaders:   []string{"Authorization", "Content-Type"},
		AllowCredentials: true,
		Debug:            false,
	})

	handler := c.Handler(mux)

	fmt.Println("Server is running ")
	PORT := os.Getenv("PORT")
	if PORT == "" {
		PORT = ":8080"
	}
	err = http.ListenAndServe(PORT, handler)
	if err != nil {
		log.Fatal("Error:", err)
	}
}
