package routes

import (
	"agents_go/handlers"

	"github.com/gorilla/mux"
)

// SetupRoutes configures all application routes
func SetupRoutes() *mux.Router {
	r := mux.NewRouter()

	// Register routes
	r.HandleFunc("/", handlers.HomeHandler).Methods("GET")
	r.HandleFunc("/login", handlers.LoginHandler).Methods("GET")
	r.HandleFunc("/callback", handlers.CallbackHandler).Methods("GET")
	r.HandleFunc("/dashboard", handlers.DashboardHandler).Methods("GET")
	r.HandleFunc("/logout", handlers.LogoutHandler).Methods("GET")
	
	// Report routes
	r.HandleFunc("/reports", handlers.ReportsHandler).Methods("GET")
	r.HandleFunc("/generate-report", handlers.GenerateReportHandler).Methods("POST")
	r.HandleFunc("/view-report", handlers.ViewReportHandler).Methods("GET")
	r.HandleFunc("/download-report-pdf", handlers.DownloadReportPDFHandler).Methods("GET")
	
	// Chat endpoint for testing the model
	r.HandleFunc("/api/chat", handlers.ChatHandler).Methods("POST")

	// Serve static files if needed
	// r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))

	return r
}
