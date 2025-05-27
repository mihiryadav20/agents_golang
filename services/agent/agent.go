package agent

import (
	"fmt"
	"log"
	"sync"
	"time"

	"agents_go/models"
	"agents_go/services/aifoundry"
	"agents_go/services/trello"
)

// ReportSchedule defines when reports should be generated
type ReportSchedule struct {
	Weekly  bool
	Monthly bool
}

// Agent handles the scheduled generation of reports
type Agent struct {
	trelloClient   *trello.Client
	aifoundryClient *aifoundry.Client
	reportStore    *models.ReportStore
	schedule       ReportSchedule
	stop           chan struct{}
	wg             sync.WaitGroup
	running        bool
	mutex          sync.Mutex
}

// NewAgent creates a new agent
func NewAgent(accessToken, accessSecret string, schedule ReportSchedule) (*Agent, error) {
	trelloClient := trello.NewClient(accessToken, accessSecret)
	aifoundryClient := aifoundry.NewClient()

	reportStore, err := models.NewReportStore("./data/reports")
	if err != nil {
		return nil, fmt.Errorf("error creating report store: %v", err)
	}

	return &Agent{
		trelloClient:   trelloClient,
		aifoundryClient: aifoundryClient,
		reportStore:    reportStore,
		schedule:       schedule,
		stop:           make(chan struct{}),
	}, nil
}

// Start starts the agent
func (a *Agent) Start() error {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	if a.running {
		return fmt.Errorf("agent is already running")
	}

	a.running = true
	a.stop = make(chan struct{})

	a.wg.Add(1)
	go a.run()

	log.Println("Agent started")
	return nil
}

// Stop stops the agent
func (a *Agent) Stop() error {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	if !a.running {
		return fmt.Errorf("agent is not running")
	}

	close(a.stop)
	a.wg.Wait()
	a.running = false

	log.Println("Agent stopped")
	return nil
}

// run is the main loop of the agent
func (a *Agent) run() {
	defer a.wg.Done()

	// Check for reports immediately on startup
	a.checkAndGenerateReports()

	// Set up ticker for daily checks
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			a.checkAndGenerateReports()
		case <-a.stop:
			return
		}
	}
}

// checkAndGenerateReports checks if reports need to be generated based on the schedule
func (a *Agent) checkAndGenerateReports() {
	now := time.Now()

	// Check if weekly report is due (every Monday)
	if a.schedule.Weekly && now.Weekday() == time.Monday {
		a.generateWeeklyReports(now)
	}

	// Check if monthly report is due (1st day of the month)
	if a.schedule.Monthly && now.Day() == 1 {
		a.generateMonthlyReports(now)
	}
}

// generateWeeklyReports generates weekly reports for all boards
func (a *Agent) generateWeeklyReports(now time.Time) {
	// Get end date (current date)
	endDate := now.Truncate(24 * time.Hour)
	
	// Get start date (7 days ago)
	startDate := endDate.AddDate(0, 0, -7)

	// Get all boards
	boards, err := a.trelloClient.GetBoards()
	if err != nil {
		log.Printf("Error getting boards for weekly reports: %v", err)
		return
	}

	// Generate report for each board
	for _, board := range boards {
		a.generateReport(board.ID, board.Name, models.Weekly, startDate, endDate)
	}
}

// generateMonthlyReports generates monthly reports for all boards
func (a *Agent) generateMonthlyReports(now time.Time) {
	// Get end date (current date)
	endDate := now.Truncate(24 * time.Hour)
	
	// Get start date (last month)
	startDate := endDate.AddDate(0, -1, 0)

	// Get all boards
	boards, err := a.trelloClient.GetBoards()
	if err != nil {
		log.Printf("Error getting boards for monthly reports: %v", err)
		return
	}

	// Generate report for each board
	for _, board := range boards {
		a.generateReport(board.ID, board.Name, models.Monthly, startDate, endDate)
	}
}

// generateReport generates a report for a specific board
func (a *Agent) generateReport(boardID, boardName string, reportType models.ReportType, startDate, endDate time.Time) {
	log.Printf("Generating %s report for board %s (%s)", reportType, boardName, boardID)

	// Get board data
	boardData, err := a.trelloClient.GetBoardData(boardID, startDate)
	if err != nil {
		log.Printf("Error getting board data: %v", err)
		return
	}

	// Generate report using AI Foundry
	reportContent, err := a.aifoundryClient.GenerateReport(boardData, string(reportType))
	if err != nil {
		log.Printf("Error generating report: %v", err)
		return
	}

	// Create report
	report := &models.Report{
		ID:          fmt.Sprintf("%s_%s_%s", boardID, reportType, endDate.Format("2006-01-02")),
		BoardID:     boardID,
		BoardName:   boardName,
		Type:        reportType,
		Content:     reportContent,
		GeneratedAt: time.Now(),
		StartDate:   startDate,
		EndDate:     endDate,
	}

	// Save report
	if err := a.reportStore.SaveReport(report); err != nil {
		log.Printf("Error saving report: %v", err)
		return
	}

	log.Printf("Successfully generated %s report for board %s", reportType, boardName)
}

// GenerateReportOnDemand generates a report on demand
func (a *Agent) GenerateReportOnDemand(boardID string, reportType models.ReportType) (*models.Report, error) {
	// Get board details
	board, err := a.trelloClient.GetBoardDetails(boardID)
	if err != nil {
		return nil, fmt.Errorf("error getting board details: %v", err)
	}

	now := time.Now()
	var startDate time.Time

	// Set date range based on report type
	if reportType == models.Weekly {
		startDate = now.AddDate(0, 0, -7)
	} else {
		startDate = now.AddDate(0, -1, 0)
	}

	// Get board data
	boardData, err := a.trelloClient.GetBoardData(boardID, startDate)
	if err != nil {
		return nil, fmt.Errorf("error getting board data: %v", err)
	}

	// Generate report using AI Foundry
	reportContent, err := a.aifoundryClient.GenerateReport(boardData, string(reportType))
	if err != nil {
		return nil, fmt.Errorf("error generating report: %v", err)
	}

	// Create report
	report := &models.Report{
		ID:          fmt.Sprintf("%s_%s_%s", boardID, reportType, now.Format("2006-01-02")),
		BoardID:     boardID,
		BoardName:   board.Name,
		Type:        reportType,
		Content:     reportContent,
		GeneratedAt: now,
		StartDate:   startDate,
		EndDate:     now,
	}

	// Save report
	if err := a.reportStore.SaveReport(report); err != nil {
		return nil, fmt.Errorf("error saving report: %v", err)
	}

	return report, nil
}

// GetReportsByBoard gets all reports for a specific board
func (a *Agent) GetReportsByBoard(boardID string) ([]*models.Report, error) {
	return a.reportStore.GetReportsByBoard(boardID)
}

// GetReportsByType gets all reports of a specific type
func (a *Agent) GetReportsByType(reportType models.ReportType) ([]*models.Report, error) {
	return a.reportStore.GetReportsByType(reportType)
}

// GetReport gets a specific report by ID
func (a *Agent) GetReport(id string) (*models.Report, error) {
	return a.reportStore.GetReport(id)
}
