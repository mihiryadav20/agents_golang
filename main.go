package main

import (
	"agents_go/config"
	"agents_go/handlers"
	"agents_go/routes"
	"log"
	"net/http"
	"os"
	"time"
)

func main() {
	// Initialize the OAuth consumer
	config.Init()
	
	// Initialize templates
	handlers.InitTemplates()
	
	// Initialize the reporting agent
	handlers.InitAgent()
	
	// Set up routes
	r := routes.SetupRoutes()
	
	// Start the server
	port := os.Getenv("PORT")
	if port == "" {
		port = "5001"
	}

	srv := &http.Server{
		Handler:      r,
		Addr:         "0.0.0.0:" + port,
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	log.Printf("Server starting on port %s...", port)
	log.Fatal(srv.ListenAndServe())
}
