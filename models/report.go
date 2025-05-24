package models

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"
)

// ReportType defines the type of report
type ReportType string

const (
	// Weekly report type
	Weekly ReportType = "weekly"
	// Monthly report type
	Monthly ReportType = "monthly"
)

// Report represents a project report
type Report struct {
	ID          string     `json:"id"`
	BoardID     string     `json:"board_id"`
	BoardName   string     `json:"board_name"`
	Type        ReportType `json:"type"`
	Content     string     `json:"content"`
	GeneratedAt time.Time  `json:"generated_at"`
	StartDate   time.Time  `json:"start_date"`
	EndDate     time.Time  `json:"end_date"`
}

// ReportStore handles storage and retrieval of reports
type ReportStore struct {
	StoragePath string
}

// NewReportStore creates a new report store
func NewReportStore(storagePath string) (*ReportStore, error) {
	// Create storage directory if it doesn't exist
	if err := os.MkdirAll(storagePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create storage directory: %v", err)
	}

	return &ReportStore{
		StoragePath: storagePath,
	}, nil
}

// SaveReport saves a report to storage
func (s *ReportStore) SaveReport(report *Report) error {
	// Create filename based on report properties
	filename := fmt.Sprintf("%s_%s_%s.json", 
		report.BoardID, 
		report.Type, 
		report.GeneratedAt.Format("2006-01-02"))
	
	filepath := filepath.Join(s.StoragePath, filename)

	// Convert report to JSON
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshaling report: %v", err)
	}

	// Write to file
	if err := ioutil.WriteFile(filepath, data, 0644); err != nil {
		return fmt.Errorf("error writing report file: %v", err)
	}

	return nil
}

// GetReportsByBoard retrieves all reports for a specific board
func (s *ReportStore) GetReportsByBoard(boardID string) ([]*Report, error) {
	pattern := fmt.Sprintf("%s_*.json", boardID)
	matches, err := filepath.Glob(filepath.Join(s.StoragePath, pattern))
	if err != nil {
		return nil, fmt.Errorf("error finding reports: %v", err)
	}

	reports := make([]*Report, 0, len(matches))
	for _, match := range matches {
		data, err := ioutil.ReadFile(match)
		if err != nil {
			return nil, fmt.Errorf("error reading report file: %v", err)
		}

		var report Report
		if err := json.Unmarshal(data, &report); err != nil {
			return nil, fmt.Errorf("error unmarshaling report: %v", err)
		}

		reports = append(reports, &report)
	}

	return reports, nil
}

// GetReportsByType retrieves all reports of a specific type
func (s *ReportStore) GetReportsByType(reportType ReportType) ([]*Report, error) {
	pattern := fmt.Sprintf("*_%s_*.json", reportType)
	matches, err := filepath.Glob(filepath.Join(s.StoragePath, pattern))
	if err != nil {
		return nil, fmt.Errorf("error finding reports: %v", err)
	}

	reports := make([]*Report, 0, len(matches))
	for _, match := range matches {
		data, err := ioutil.ReadFile(match)
		if err != nil {
			return nil, fmt.Errorf("error reading report file: %v", err)
		}

		var report Report
		if err := json.Unmarshal(data, &report); err != nil {
			return nil, fmt.Errorf("error unmarshaling report: %v", err)
		}

		reports = append(reports, &report)
	}

	return reports, nil
}

// GetReport retrieves a specific report by ID
func (s *ReportStore) GetReport(id string) (*Report, error) {
	// List all files in the directory
	files, err := ioutil.ReadDir(s.StoragePath)
	if err != nil {
		return nil, fmt.Errorf("error reading storage directory: %v", err)
	}

	// Look for a file that contains the report ID
	for _, file := range files {
		if !file.IsDir() {
			filePath := filepath.Join(s.StoragePath, file.Name())
			data, err := ioutil.ReadFile(filePath)
			if err != nil {
				continue // Skip files we can't read
			}

			var report Report
			if err := json.Unmarshal(data, &report); err != nil {
				continue // Skip files that aren't valid reports
			}

			// Check if this is the report we're looking for
			if report.ID == id {
				return &report, nil
			}
		}
	}

	return nil, fmt.Errorf("report not found")
}

// DeleteReport deletes a report by ID
func (s *ReportStore) DeleteReport(id string) error {
	pattern := fmt.Sprintf("*_%s_*.json", id)
	matches, err := filepath.Glob(filepath.Join(s.StoragePath, pattern))
	if err != nil {
		return fmt.Errorf("error finding report: %v", err)
	}

	if len(matches) == 0 {
		return fmt.Errorf("report not found")
	}

	if err := os.Remove(matches[0]); err != nil {
		return fmt.Errorf("error deleting report file: %v", err)
	}

	return nil
}
