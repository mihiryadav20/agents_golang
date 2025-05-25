package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os/exec"

	"agents_go/config"
	"agents_go/models"
	"agents_go/services/agent"
	"agents_go/services/pdf"

	"github.com/mrjones/oauth"
)

var reportAgent *agent.Agent

// InitAgent initializes the report agent
func InitAgent() {
	// Create data directory
	if err := createDataDirectory(); err != nil {
		log.Printf("Error creating data directory: %v", err)
	}
}

// createDataDirectory creates the data directory for reports
func createDataDirectory() error {
	// Create data directory
	cmd := exec.Command("mkdir", "-p", "./data/reports")
	err := cmd.Run()
	if err != nil {
		log.Printf("Error creating data directory: %v", err)
	}
	return err
}

// ReportsHandler displays the reports page
func ReportsHandler(w http.ResponseWriter, r *http.Request) {
	// Check if the user is authenticated
	session, _ := config.Store.Get(r, "trello-oauth")
	accessToken, ok1 := session.Values["accessToken"].(string)
	accessSecret, ok2 := session.Values["accessSecret"].(string)

	if !ok1 || !ok2 {
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}

	// Get the board ID from the query parameters
	boardID := r.URL.Query().Get("board_id")
	if boardID == "" {
		// If no board ID is provided, redirect to the dashboard
		http.Redirect(w, r, "/dashboard", http.StatusTemporaryRedirect)
		return
	}

	// Create agent if not already created
	var err error
	if reportAgent == nil {
		reportAgent, err = agent.NewAgent(accessToken, accessSecret, agent.ReportSchedule{
			Weekly:  true,
			Monthly: true,
		})
		if err != nil {
			log.Printf("Error creating agent: %v", err)
			http.Error(w, "Error creating agent", http.StatusInternalServerError)
			return
		}
	}

	// Get reports for the board
	reports, err := reportAgent.GetReportsByBoard(boardID)
	if err != nil {
		log.Printf("Error getting reports: %v", err)
		reports = []*models.Report{} // Set to empty if error
	}

	// Create a token for API calls
	token := &oauth.AccessToken{
		Token:  accessToken,
		Secret: accessSecret,
	}

	// Get board details
	resp, err := config.Consumer.Get(
		fmt.Sprintf("https://api.trello.com/1/boards/%s", boardID),
		map[string]string{"fields": "name,desc"},
		token,
	)
	if err != nil {
		log.Printf("Error getting board details: %v", err)
		http.Error(w, "Error getting board details", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	var board map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&board); err != nil {
		log.Printf("Error parsing board data: %v", err)
		http.Error(w, "Error parsing board data", http.StatusInternalServerError)
		return
	}

	// Render the reports template
	data := map[string]interface{}{
		"Title":   "Trello Reports",
		"Board":   board,
		"Reports": reports,
	}
	Templates["reports.html"].Execute(w, data)
}

// GenerateReportHandler generates a new report
func GenerateReportHandler(w http.ResponseWriter, r *http.Request) {
	// Check if the user is authenticated
	session, _ := config.Store.Get(r, "trello-oauth")
	accessToken, ok1 := session.Values["accessToken"].(string)
	accessSecret, ok2 := session.Values["accessSecret"].(string)

	if !ok1 || !ok2 {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Only allow POST requests
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse form data
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Error parsing form", http.StatusBadRequest)
		return
	}

	// Get parameters
	boardID := r.FormValue("board_id")
	reportType := r.FormValue("report_type")

	if boardID == "" || reportType == "" {
		http.Error(w, "Missing parameters", http.StatusBadRequest)
		return
	}

	// Validate report type
	var rType models.ReportType
	if reportType == "weekly" {
		rType = models.Weekly
	} else if reportType == "monthly" {
		rType = models.Monthly
	} else {
		http.Error(w, "Invalid report type", http.StatusBadRequest)
		return
	}

	// Create agent if not already created
	var err error
	if reportAgent == nil {
		reportAgent, err = agent.NewAgent(accessToken, accessSecret, agent.ReportSchedule{
			Weekly:  true,
			Monthly: true,
		})
		if err != nil {
			log.Printf("Error creating agent: %v", err)
			http.Error(w, "Error creating agent", http.StatusInternalServerError)
			return
		}
	}

	// Generate report
	report, err := reportAgent.GenerateReportOnDemand(boardID, rType)
	if err != nil {
		log.Printf("Error generating report: %v", err)
		http.Error(w, "Error generating report", http.StatusInternalServerError)
		return
	}

	// Redirect to the report view
	http.Redirect(w, r, fmt.Sprintf("/view-report?id=%s", report.ID), http.StatusSeeOther)
}

// ViewReportHandler displays a specific report
func ViewReportHandler(w http.ResponseWriter, r *http.Request) {
	// Check if the user is authenticated
	session, _ := config.Store.Get(r, "trello-oauth")
	accessToken, ok1 := session.Values["accessToken"].(string)
	accessSecret, ok2 := session.Values["accessSecret"].(string)

	if !ok1 || !ok2 {
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}

	// Get the report ID from the query parameters
	reportID := r.URL.Query().Get("id")
	if reportID == "" {
		http.Error(w, "Missing report ID", http.StatusBadRequest)
		return
	}

	// Create agent if not already created
	var err error
	if reportAgent == nil {
		reportAgent, err = agent.NewAgent(accessToken, accessSecret, agent.ReportSchedule{
			Weekly:  true,
			Monthly: true,
		})
		if err != nil {
			log.Printf("Error creating agent: %v", err)
			http.Error(w, "Error creating agent", http.StatusInternalServerError)
			return
		}
	}

	// Get the report
	report, err := reportAgent.GetReport(reportID)
	if err != nil {
		log.Printf("Error getting report: %v", err)
		http.Error(w, "Report not found", http.StatusNotFound)
		return
	}

	// Render the report template
	data := map[string]interface{}{
		"Title":  fmt.Sprintf("%s Report - %s", report.Type, report.BoardName),
		"Report": report,
	}
	Templates["view_report.html"].Execute(w, data)
}

// DownloadReportPDFHandler generates and serves a PDF version of a report
func DownloadReportPDFHandler(w http.ResponseWriter, r *http.Request) {
	// Check if the user is authenticated
	session, _ := config.Store.Get(r, "trello-oauth")
	accessToken, ok1 := session.Values["accessToken"].(string)
	accessSecret, ok2 := session.Values["accessSecret"].(string)

	if !ok1 || !ok2 {
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}

	// Get the report ID from the query parameters
	reportID := r.URL.Query().Get("id")
	if reportID == "" {
		http.Error(w, "Missing report ID", http.StatusBadRequest)
		return
	}

	// Create agent if not already created
	var err error
	if reportAgent == nil {
		reportAgent, err = agent.NewAgent(accessToken, accessSecret, agent.ReportSchedule{
			Weekly:  true,
			Monthly: true,
		})
		if err != nil {
			log.Printf("Error creating agent: %v", err)
			http.Error(w, "Error creating agent", http.StatusInternalServerError)
			return
		}
	}

	// Get the report
	report, err := reportAgent.GetReport(reportID)
	if err != nil {
		log.Printf("Error getting report: %v", err)
		http.Error(w, "Report not found", http.StatusNotFound)
		return
	}

	// Create PDF generator
	pdfGenerator := pdf.NewGenerator()

	// Generate PDF from report content
	pdfBuffer, err := pdfGenerator.GenerateReport(
		report.Content,
		report.BoardName,
		string(report.Type),
		report.StartDate,
		report.EndDate,
	)
	if err != nil {
		log.Printf("Error generating PDF: %v", err)
		http.Error(w, "Error generating PDF", http.StatusInternalServerError)
		return
	}

	// Set response headers for PDF download
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s_%s_report_%s.pdf", 
		report.BoardName,
		report.Type,
		report.GeneratedAt.Format("2006-01-02")))

	// Write PDF buffer to response
	if _, err := w.Write(pdfBuffer.Bytes()); err != nil {
		log.Printf("Error writing PDF to response: %v", err)
		http.Error(w, "Error serving PDF", http.StatusInternalServerError)
		return
	}
}
