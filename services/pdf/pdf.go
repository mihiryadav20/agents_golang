package pdf

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/jung-kurt/gofpdf"
)

// Generator is a PDF report generator
type Generator struct {
	// PDF configuration options can be added here
}

// NewGenerator creates a new PDF generator
func NewGenerator() *Generator {
	return &Generator{}
}

// GenerateReport generates a PDF report from the given content
func (g *Generator) GenerateReport(content, boardName, reportType string, startDate, endDate time.Time) (*bytes.Buffer, error) {
	// Create a new PDF document
	pdf := gofpdf.New("P", "mm", "A4", "")
	
	// Set document properties
	pdf.SetTitle(fmt.Sprintf("%s %s Report", boardName, reportType), true)
	pdf.SetAuthor("Trello Reporting Agent", true)
	pdf.SetCreationDate(time.Now())
	
	// Add a page
	pdf.AddPage()
	
	// Set font
	pdf.SetFont("Arial", "B", 16)
	
	// Add title
	pdf.Cell(190, 10, fmt.Sprintf("%s %s Report", boardName, strings.Title(reportType)))
	pdf.Ln(15)
	
	// Add report period
	pdf.SetFont("Arial", "I", 10)
	pdf.Cell(190, 6, fmt.Sprintf("Period: %s to %s", startDate.Format("Jan 2, 2006"), endDate.Format("Jan 2, 2006")))
	pdf.Ln(6)
	pdf.Cell(190, 6, fmt.Sprintf("Generated: %s", time.Now().Format("January 2, 2006 at 3:04 PM")))
	pdf.Ln(15)
	
	// Process the content for PDF
	processedContent := g.processContentForPDF(content)
	
	// Add the content to the PDF
	g.addFormattedContent(pdf, processedContent)
	
	// Save to buffer
	var buf bytes.Buffer
	err := pdf.Output(&buf)
	if err != nil {
		return nil, fmt.Errorf("error generating PDF: %v", err)
	}
	
	return &buf, nil
}

// processContentForPDF processes the HTML content to make it suitable for PDF
func (g *Generator) processContentForPDF(content string) string {
	// If the content is HTML (from markdown conversion), extract the text
	// This is a simple approach - for a more robust solution, consider using an HTML parser
	
	// Remove HTML tags
	content = regexp.MustCompile("<[^>]*>").ReplaceAllString(content, "")
	
	// Replace HTML entities
	content = strings.ReplaceAll(content, "&ldquo;", "\"")
	content = strings.ReplaceAll(content, "&rdquo;", "\"")
	content = strings.ReplaceAll(content, "&amp;", "&")
	content = strings.ReplaceAll(content, "&lt;", "<")
	content = strings.ReplaceAll(content, "&gt;", ">")
	content = strings.ReplaceAll(content, "&nbsp;", " ")
	
	// Clean up extra whitespace
	content = regexp.MustCompile(`\s+`).ReplaceAllString(content, " ")
	content = regexp.MustCompile(`\n\s*\n`).ReplaceAllString(content, "\n\n")
	
	return content
}

// addFormattedContent adds formatted content to the PDF
func (g *Generator) addFormattedContent(pdf *gofpdf.Fpdf, content string) {
	// Split content into lines
	lines := strings.Split(content, "\n")
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			pdf.Ln(5)
			continue
		}
		
		// Check if this is a heading
		if strings.HasPrefix(line, "Executive Summary") || 
		   strings.HasPrefix(line, "Progress This Week") || 
		   strings.HasPrefix(line, "Current Project Status") || 
		   strings.HasPrefix(line, "Priorities & Deadlines") || 
		   strings.HasPrefix(line, "Risks, Blockers") || 
		   strings.HasPrefix(line, "Team Focus") || 
		   strings.HasPrefix(line, "Data Limitations") {
			// Add some space before headings
			pdf.Ln(5)
			pdf.SetFont("Arial", "B", 12)
			pdf.Cell(190, 6, line)
			pdf.Ln(8)
			pdf.SetFont("Arial", "", 10)
		} else {
			// Regular text
			pdf.MultiCell(190, 5, line, "", "", false)
			pdf.Ln(2)
		}
	}
}
