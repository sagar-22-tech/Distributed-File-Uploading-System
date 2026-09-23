package main

import (
	"file-system/handlers"
	"fmt"
	"log"
	"net/http"

	"github.com/rs/cors"
)

func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("/upload", handlers.Upload)
	mux.HandleFunc("/upload/init", handlers.UploadInit)
	mux.HandleFunc("/upload/{fileId}", handlers.UploadChunk)
	mux.HandleFunc("/upload/{fileId}/complete", handlers.UploadComplete)
	mux.HandleFunc("/health", handlers.HealthCheck)
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"http://localhost:5173"},
		AllowedMethods:   []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodOptions},
		AllowedHeaders:   []string{"Authorization", "Content-Type"},
		AllowCredentials: true,
		Debug:            false, // Turn on for helpful logging during development
	})

	// Wrap your multiplexer with the CORS middleware
	handler := c.Handler(mux)

	fmt.Println("Server is running on http://localhost:8080")
	PORT := ":8080"
	err := http.ListenAndServe(PORT, handler)
	if err != nil {
		log.Fatal("Error:", err)
	}
}
