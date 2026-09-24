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

	if err := godotenv.Load(); err != nil {

		log.Println("No .env file found; falling back to system environment variables")
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
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080" // local fallback
	}

	// FIX: Ensure the port is prepended with a colon
	log.Printf("Server is running on port %s", port)
	err := http.ListenAndServe(":"+port, handler)
	if err != nil {
		log.Fatal("Error starting server: ", err)
	}

}
