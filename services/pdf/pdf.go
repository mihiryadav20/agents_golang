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
	// Create a new PDF document with margins
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(15, 15, 15) // Left, Top, Right margins for a clean layout

	// Set document properties
	pdf.SetTitle(fmt.Sprintf("%s %s Report", boardName, reportType), true)
	pdf.SetAuthor("Trello Reporting Agent", true)
	pdf.SetCreationDate(time.Now())

	// Add a page
	pdf.AddPage()

	// Add title
	pdf.SetFont("Arial", "B", 18)
	pdf.CellFormat(180, 12, fmt.Sprintf("%s %s Report", boardName, strings.Title(reportType)), "", 0, "C", false, 0, "")
	pdf.Ln(15)

	// Add report period
	pdf.SetFont("Arial", "I", 10)
	pdf.CellFormat(180, 6, fmt.Sprintf("Period: %s to %s", startDate.Format("Jan 2, 2006"), endDate.Format("Jan 2, 2006")), "", 0, "L", false, 0, "")
	pdf.Ln(6)
	pdf.CellFormat(180, 6, fmt.Sprintf("Generated: %s", time.Now().Format("January 2, 2006 at 3:04 PM")), "", 0, "L", false, 0, "")
	pdf.Ln(12)

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

// processContentForPDF processes the Markdown content to make it suitable for PDF
func (g *Generator) processContentForPDF(content string) string {
	// Remove HTML tags if present
	content = regexp.MustCompile("<[^>]*>").ReplaceAllString(content, "")

	// Replace HTML entities
	content = strings.ReplaceAll(content, "“", "\"")
	content = strings.ReplaceAll(content, "”", "\"")
	content = strings.ReplaceAll(content, "&", "&")
	content = strings.ReplaceAll(content, "<", "<")
	content = strings.ReplaceAll(content, ">", ">")
	content = strings.ReplaceAll(content, " ", " ")

	// Remove Markdown heading markers
	content = regexp.MustCompile(`(?m)^##\s+`).ReplaceAllString(content, "")
	content = regexp.MustCompile(`(?m)^###\s+`).ReplaceAllString(content, "**")

	// Clean up bold markers for subsections (e.g., **Tasks Completed:**)
	content = regexp.MustCompile(`\*\*(.*?):\*\*`).ReplaceAllString(content, "**$1**")

	// Normalize whitespace and newlines
	content = regexp.MustCompile(`\s+`).ReplaceAllString(content, " ")
	content = regexp.MustCompile(`\n\s*\n+`).ReplaceAllString(content, "\n\n")

	return strings.TrimSpace(content)
}

// addFormattedContent adds formatted content to the PDF
func (g *Generator) addFormattedContent(pdf *gofpdf.Fpdf, content string) {
	// Parse content into sections
	sections := g.parseContentSections(content)

	// Set initial font for body text
	pdf.SetFont("Arial", "", 10)

	// Add each section to the PDF
	for _, section := range sections {
		// Add section heading
		pdf.Ln(10)
		pdf.SetFont("Arial", "B", 16)
		pdf.CellFormat(180, 8, section.Title, "", 0, "L", false, 0, "")
		pdf.Ln(8)

		// Add paragraphs and subsections
		for _, para := range section.Paragraphs {
			// Check if this is a subsection heading
			if strings.HasPrefix(para, "**") && strings.HasSuffix(para, "**") {
				// Extract subsection title (remove **)
				subTitle := strings.TrimPrefix(strings.TrimSuffix(para, "**"), "**")
				pdf.Ln(4)
				pdf.SetFont("Arial", "B", 12)
				pdf.CellFormat(180, 6, subTitle, "", 0, "L", false, 0, "")
				pdf.Ln(5)
			} else {
				// Regular paragraph
				pdf.SetFont("Arial", "", 10)
				pdf.MultiCell(180, 5, para, "", "", false)
				pdf.Ln(4)
			}
		}

		// Add bullet points if any
		if len(section.BulletPoints) > 0 {
			pdf.Ln(4)
			for _, bullet := range section.BulletPoints {
				// Handle nested bullet points
				indent := 10
				if strings.HasPrefix(bullet, "  ") {
					indent = 15 // Extra indent for nested bullets
					bullet = strings.TrimPrefix(bullet, "  ")
				}
				pdf.SetX(float64(indent))
				pdf.SetFont("Arial", "", 10)
				pdf.Cell(5, 5, "•")
				pdf.SetX(float64(indent + 5))
				pdf.MultiCell(170, 5, strings.TrimSpace(bullet), "", "", false)
				pdf.Ln(2)
			}
		}
	}
}

// ContentSection represents a section of the report
type ContentSection struct {
	Title        string
	Paragraphs   []string
	BulletPoints []string
}

// parseContentSections parses the content into structured sections
func (g *Generator) parseContentSections(content string) []ContentSection {
	// Define main section titles
	sectionTitles := []string{
		"Executive Summary",
		"Progress This Week",
		"Current Project Status",
		"Priorities & Deadlines for Next Week",
		"Risks, Blockers & Issues",
		"Team Focus & Contributions",
		"Data Limitations",
	}

	// Create regex pattern for section titles
	pattern := "(?m)^(" + strings.Join(sectionTitles, "|") + ")"
	regex := regexp.MustCompile(pattern)
	sectionTexts := regex.Split(content, -1)
	sectionMatches := regex.FindAllStringSubmatch(content, -1)

	// Create sections
	sections := []ContentSection{}

	// Process each section
	for i, match := range sectionMatches {
		if i < len(sectionTexts)-1 { // Skip the first split (before any heading)
			sectionText := strings.TrimSpace(sectionTexts[i+1])
			section := ContentSection{
				Title: match[1], // Use the matched title
			}

			// Split section text into paragraphs and bullet points
			lines := strings.Split(sectionText, "\n")
			currentParagraph := ""
			inBulletList := false

			for _, line := range lines {
				line = strings.TrimSpace(line)

				if line == "" {
					// End of paragraph
					if currentParagraph != "" {
						section.Paragraphs = append(section.Paragraphs, strings.TrimSpace(currentParagraph))
						currentParagraph = ""
					}
					inBulletList = false
					continue
				}

				if strings.HasPrefix(line, "- ") || strings.HasPrefix(line, "* ") {
					// Bullet point
					bulletText := strings.TrimPrefix(strings.TrimPrefix(line, "- "), "* ")
					section.BulletPoints = append(section.BulletPoints, bulletText)
					inBulletList = true
					continue
				}

				if strings.HasPrefix(line, "**") && strings.HasSuffix(line, "**") {
					// Subsection heading
					if currentParagraph != "" {
						section.Paragraphs = append(section.Paragraphs, strings.TrimSpace(currentParagraph))
						currentParagraph = ""
					}
					section.Paragraphs = append(section.Paragraphs, line)
					inBulletList = false
					continue
				}

				// Part of a paragraph
				if currentParagraph != "" && !inBulletList {
					currentParagraph += " "
				}
				currentParagraph += line
			}

			// Add the last paragraph if any
			if currentParagraph != "" {
				section.Paragraphs = append(section.Paragraphs, strings.TrimSpace(currentParagraph))
			}

			sections = append(sections, section)
		}
	}

	return sections
}
