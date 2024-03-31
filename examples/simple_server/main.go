package main

import (
	"go-rtc-lib/internal/connection" // Adjust the import path according to your project structure.
	"log"
	"net/http"
)

func main() {
	http.HandleFunc("/ws", connection.Handler) // Set WebSocket endpoint.
	log.Println("WebSocket server starting on :8080...")
	err := http.ListenAndServe(":8080", nil) // Start HTTP server.
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
