package handlers

import (
	"agents_go/services/aifoundry"
	"encoding/json"
	"log"
	"net/http"
)

// ChatRequest represents a request to the chat endpoint
type ChatRequest struct {
	Message string `json:"message"`
}

// ChatResponse represents a response from the chat endpoint
type ChatResponse struct {
	Response string `json:"response"`
	Error    string `json:"error,omitempty"`
}

// ChatHandler handles chat requests to test the Mistral model
func ChatHandler(w http.ResponseWriter, r *http.Request) {
	// Set content type
	w.Header().Set("Content-Type", "application/json")

	// Only accept POST requests
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(ChatResponse{
			Error: "Method not allowed",
		})
		return
	}

	// Parse request
	var chatReq ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&chatReq); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ChatResponse{
			Error: "Invalid request format",
		})
		return
	}

	// Create Mistral client
	client := aifoundry.NewClient()
	// Send message to Mistral
	response, err := client.SendChatMessage(chatReq.Message)
	if err != nil {
		log.Printf("Error sending chat message: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ChatResponse{
			Error: "Error generating response: " + err.Error(),
		})
		return
	}

	// Return response
	json.NewEncoder(w).Encode(ChatResponse{
		Response: response,
	})
}
