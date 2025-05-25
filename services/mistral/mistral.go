package mistral

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"agents_go/config"
)

// Client is a Mistral API client
type Client struct {
	APIKey  string
	BaseURL string
}

// Message represents a message in the chat
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatRequest represents a request to the Mistral chat API
type ChatRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Temperature float64   `json:"temperature,omitempty"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
}

// ChatResponse represents a response from the Mistral chat API
type ChatResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index   int `json:"index"`
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

// NewClient creates a new Mistral client
func NewClient() *Client {
	return &Client{
		APIKey:  config.MistralAPIKey,
		BaseURL: config.MistralAPIURL,
	}
}

// SendChatMessage sends a simple chat message to the Mistral API
func (c *Client) SendChatMessage(message string) (string, error) {
	// Create the request
	messages := []Message{
		{
			Role:    "system",
			Content: "You are a helpful assistant for Trello users. You provide concise and accurate information.",
		},
		{
			Role:    "user",
			Content: message,
		},
	}

	chatReq := ChatRequest{
		Model:       config.MistralModel,
		Messages:    messages,
		Temperature: 0.7,
		MaxTokens:   1000,
	}

	// Convert request to JSON
	reqBody, err := json.Marshal(chatReq)
	if err != nil {
		return "", fmt.Errorf("error marshaling request: %v", err)
	}

	// Create HTTP request
	req, err := http.NewRequest("POST", c.BaseURL+"/chat/completions", bytes.NewBuffer(reqBody))
	if err != nil {
		return "", fmt.Errorf("error creating request: %v", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.APIKey)
	// OpenRouter specific headers
	req.Header.Set("HTTP-Referer", "http://trello-reporting-agent.local")
	req.Header.Set("X-Title", "Trello Reporting Agent")

	// Send request
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("error sending request: %v", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("error from API: %s (status code: %d)", string(body), resp.StatusCode)
	}

	// Parse response
	var chatResp ChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return "", fmt.Errorf("error unmarshaling response: %v", err)
	}

	// Check if we got any choices
	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("no response from model")
	}

	// Return the generated message
	return chatResp.Choices[0].Message.Content, nil
}

// GenerateReport generates a report using the Mistral API
func (c *Client) GenerateReport(boardData map[string]interface{}, reportType string) (string, error) {
	// Convert board data to a more readable format for the LLM
	boardSummary, err := formatBoardData(boardData)
	if err != nil {
		return "", fmt.Errorf("error formatting board data: %v", err)
	}

	// Create system prompt based on report type
	systemPrompt := getReportSystemPrompt(reportType)

	// Create the request
	messages := []Message{
		{
			Role:    "system",
			Content: systemPrompt,
		},
		{
			Role:    "user",
			Content: boardSummary,
		},
	}

	chatReq := ChatRequest{
		Model:       config.MistralModel, // Using model specified in config
		Messages:    messages,
		Temperature: 0.7,
		MaxTokens:   4000,
	}

	// Convert request to JSON
	reqBody, err := json.Marshal(chatReq)
	if err != nil {
		return "", fmt.Errorf("error marshaling request: %v", err)
	}

	// Create HTTP request
	req, err := http.NewRequest("POST", c.BaseURL+"/chat/completions", bytes.NewBuffer(reqBody))
	if err != nil {
		return "", fmt.Errorf("error creating request: %v", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.APIKey)
	// OpenRouter specific headers
	req.Header.Set("HTTP-Referer", "http://trello-reporting-agent.local")
	req.Header.Set("X-Title", "Trello Reporting Agent")

	// Send request
	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("error sending request: %v", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading response: %v", err)
	}

	// Check for error status
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API error: %s (status code: %d)", string(body), resp.StatusCode)
	}

	// Parse response
	var chatResp ChatResponse
	if err := json.Unmarshal(body, &chatResp); err != nil {
		return "", fmt.Errorf("error unmarshaling response: %v", err)
	}

	// Check if we got any choices
	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("no response from model")
	}

	// Get the generated message and return it directly
	return chatResp.Choices[0].Message.Content, nil
}

// formatBoardData converts the board data to a readable format for the LLM
func formatBoardData(boardData map[string]interface{}) (string, error) {
	// Extract data
	board, ok := boardData["board"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("invalid board data format")
	}

	// Check if lists is wrapped in an items field
	listsData, ok := boardData["lists"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("invalid lists data format")
	}

	lists, ok := listsData["items"].([]interface{})
	if !ok {
		return "", fmt.Errorf("invalid lists items format")
	}

	// Check if cards is wrapped in an items field
	cardsData, ok := boardData["cards"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("invalid cards data format")
	}

	cards, ok := cardsData["items"].([]interface{})
	if !ok {
		return "", fmt.Errorf("invalid cards items format")
	}

	// Check if members is wrapped in an items field
	membersData, ok := boardData["members"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("invalid members data format")
	}

	members, ok := membersData["items"].([]interface{})
	if !ok {
		return "", fmt.Errorf("invalid members items format")
	}

	activities, ok := boardData["activities"].([]interface{})
	if !ok {
		activities = []interface{}{} // Set to empty if not available
	}

	// Format the data as a string
	var buffer bytes.Buffer

	// Board information
	buffer.WriteString(fmt.Sprintf("# Project: %s\n\n", board["name"]))
	if desc, ok := board["desc"].(string); ok && desc != "" {
		buffer.WriteString(fmt.Sprintf("Description: %s\n\n", desc))
	}

	// Lists and cards
	buffer.WriteString("## Lists and Cards\n\n")
	for _, l := range lists {
		list, ok := l.(map[string]interface{})
		if !ok {
			continue
		}

		listName, ok := list["name"].(string)
		if !ok {
			continue
		}

		buffer.WriteString(fmt.Sprintf("### List: %s\n\n", listName))

		// Find cards in this list
		listCards := []map[string]interface{}{}
		for _, c := range cards {
			card, ok := c.(map[string]interface{})
			if !ok {
				continue
			}

			if card["idList"] == list["id"] {
				listCards = append(listCards, card)
			}
		}

		if len(listCards) == 0 {
			buffer.WriteString("No cards in this list.\n\n")
			continue
		}

		for _, card := range listCards {
			cardName, _ := card["name"].(string)
			buffer.WriteString(fmt.Sprintf("- Card: %s\n", cardName))

			if desc, ok := card["desc"].(string); ok && desc != "" {
				buffer.WriteString(fmt.Sprintf("  Description: %s\n", desc))
			}

			if due, ok := card["due"].(string); ok && due != "" {
				buffer.WriteString(fmt.Sprintf("  Due: %s\n", due))
			}

			if labels, ok := card["labels"].([]interface{}); ok && len(labels) > 0 {
				buffer.WriteString("  Labels: ")
				for i, l := range labels {
					label, ok := l.(map[string]interface{})
					if !ok {
						continue
					}

					if i > 0 {
						buffer.WriteString(", ")
					}

					labelName, _ := label["name"].(string)
					if labelName == "" {
						labelColor, _ := label["color"].(string)
						buffer.WriteString(labelColor)
					} else {
						buffer.WriteString(labelName)
					}
				}
				buffer.WriteString("\n")
			}

			buffer.WriteString("\n")
		}
	}

	// Members
	buffer.WriteString("## Team Members\n\n")
	for _, m := range members {
		member, ok := m.(map[string]interface{})
		if !ok {
			continue
		}

		fullName, _ := member["fullName"].(string)
		username, _ := member["username"].(string)

		if fullName != "" {
			buffer.WriteString(fmt.Sprintf("- %s (@%s)\n", fullName, username))
		} else {
			buffer.WriteString(fmt.Sprintf("- @%s\n", username))
		}
	}
	buffer.WriteString("\n")

	// Recent Activities (limited to 10)
	buffer.WriteString("## Recent Activities\n\n")
	activityCount := 0
	for _, a := range activities {
		if activityCount >= 10 {
			break
		}

		activity, ok := a.(map[string]interface{})
		if !ok {
			continue
		}

		// Extract activity details
		activityType, _ := activity["type"].(string)

		data, ok := activity["data"].(map[string]interface{})
		if !ok {
			continue
		}

		memberCreator, ok := activity["memberCreator"].(map[string]interface{})
		if !ok {
			continue
		}

		memberName, _ := memberCreator["fullName"].(string)
		if memberName == "" {
			memberName, _ = memberCreator["username"].(string)
		}

		// Format activity based on type
		var activityDesc string
		switch activityType {
		case "createCard":
			card, _ := data["card"].(map[string]interface{})
			cardName, _ := card["name"].(string)
			activityDesc = fmt.Sprintf("%s created card '%s'", memberName, cardName)
		case "updateCard":
			card, _ := data["card"].(map[string]interface{})
			cardName, _ := card["name"].(string)
			activityDesc = fmt.Sprintf("%s updated card '%s'", memberName, cardName)
		case "commentCard":
			card, _ := data["card"].(map[string]interface{})
			cardName, _ := card["name"].(string)
			text, _ := data["text"].(string)
			activityDesc = fmt.Sprintf("%s commented on '%s': %s", memberName, cardName, text)
		case "addMemberToCard":
			card, _ := data["card"].(map[string]interface{})
			cardName, _ := card["name"].(string)
			member, _ := data["member"].(map[string]interface{})
			memberName, _ := member["name"].(string)
			activityDesc = fmt.Sprintf("%s added %s to card '%s'", memberName, memberName, cardName)
		default:
			// Skip unknown activity types
			continue
		}

		buffer.WriteString(fmt.Sprintf("- %s\n", activityDesc))
		activityCount++
	}

	if activityCount == 0 {
		buffer.WriteString("No recent activities found.\n")
	}

	return buffer.String(), nil
}

// getReportSystemPrompt returns the system prompt for the specified report type


func getReportSystemPrompt(reportType string) string {
	// Common preamble to set the stage for data input
	dataContextPreamble := "You will be provided with a structured summary of Trello board data. This may include card names, descriptions, current lists (statuses), assignees, due dates, labels, comments, and recent activity logs. Your analysis should be strictly based on this provided data.\n\n"

	switch reportType {
	case "weekly":
		return dataContextPreamble + `You are an expert AI Project Management Assistant. Your task is to analyze the provided Trello board data and generate a concise, professional weekly project status report.

The primary goal of this report is to inform stakeholders and team members about weekly progress, current standing, and immediate next steps.

**Report Structure and Focus:**
Please use markdown formatting and structure your report with the following sections:

1.  **` + "`## Executive Summary`" + `** (2-3 key sentences highlighting overall progress and any critical alerts for the week.)
2.  **` + "`## Progress This Week`" + `**
    * List tasks completed (e.g., moved to 'Done' or a similar final status).
    * Highlight tasks that made significant forward movement (e.g., 'In Progress' to 'Review').
    * Note any newly critical tasks that emerged.
3.  **` + "`## Current Project Status`" + `**
    * **In Progress:** Briefly list key tasks actively being worked on.
    * **Blocked/Impeded:** Clearly identify any tasks marked as blocked or that show no progress despite upcoming deadlines. Specify the blocker if mentioned in the data.
    * **Upcoming (Next 7 Days):** List tasks scheduled or expected to start next week.
4.  **` + "`## Priorities & Deadlines for Next Week`" + `**
    * Identify key tasks and milestones due in the upcoming week.
    * Suggest priorities based on due dates, dependencies (if apparent in data), or stated priorities in task labels/descriptions.
5.  **` + "`## Risks, Blockers & Issues`" + `**
    * Reiterate any critical blockers identified above.
    * Summarize any new risks or issues that surfaced this week (e.g., derived from comments, new 'Blocked' items).
6.  **` + "`## Team Focus & Contributions`" + `** (Optional - if data supports this without making performance judgments)
    * Summarize key areas of team activity or major task completions by the team. Focus on task movement and deliverables. *Avoid subjective performance evaluations.*
7.  **` + "`## Data Limitations`" + `** (If applicable)
    * If critical information seems missing from the provided data (e.g., unclear priorities for many tasks, lack of updates on key items), briefly note this as a limitation.

**Key Instructions:**
* **Be Data-Driven:** Base all observations strictly on the provided Trello data. Avoid making assumptions or inventing details.
* **Professional Tone:** Maintain a formal and objective tone suitable for stakeholders.
* **Conciseness:** Be thorough but avoid unnecessary jargon or overly lengthy descriptions.
* **Markdown Usage:** Use headings, sub-headings, bullet points, and bold emphasis for clarity and readability.
`
	case "monthly":
		return dataContextPreamble + `You are a strategic AI Project Management Analyst. Your task is to analyze the provided Trello board data spanning the last month and generate a comprehensive monthly project report.

This report is intended for senior management and key stakeholders to provide insights into project health, achievements, trends, and strategic recommendations.

**Report Structure and Focus:**
Please use markdown formatting and structure your report with the following sections:

1.  **` + "`## Executive Summary`" + `** (A concise overview of the month's performance, key achievements, overall project health, and critical concerns.)
2.  **` + "`## Overall Project Health & Status`" + `**
    * Provide a qualitative assessment (e.g., On Track, Minor Deviations, At Risk) based on goal completion, deadline adherence, and issue volume.
    * Current status of major ongoing initiatives or project phases.
3.  **` + "`## Major Achievements & Milestones Reached`" + `**
    * Highlight significant accomplishments, milestones completed (e.g., major features delivered, project phases concluded, key deliverables approved).
4.  **` + "`## Key Performance Indicators (KPIs) & Metrics Overview`" + `**
    * Based on the provided data (e.g., number of tasks planned vs. completed, cycle times if inferable from activity logs, story points if available), summarize relevant metrics.
    * Examples: Task completion rate, number of tasks processed, progress against specific goals.
    * If direct metrics are not calculable from the data, describe qualitative progress and velocity (e.g., "Steady progress on X feature set, with Y tasks completed"). Note if specific metric data isn't available.
5.  **` + "`## Trends and Patterns Observed This Month`" + `**
    * Analyze trends in task completion rates, common blockers or delays, types of tasks taking longer, or shifts in workload distribution if apparent from the data.
6.  **` + "`## Resource Overview & Team Contributions`" + `**
    * Summarize overall team activity and workload distribution based on task assignments and completions.
    * Highlight collective achievements or contributions towards major milestones. *Avoid individual performance evaluations unless explicitly supported by objective, quantifiable data and it's appropriate for the audience.*
7.  **` + "`## Significant Risks, Issues & Mitigation`" + `**
    * Detail any significant risks or issues that arose or persisted during the month.
    * If mitigation strategies were mentioned or implemented (e.g., in comments, task updates), summarize them.
    * Identify any unresolved critical issues.
8.  **` + "`## Recommendations for Upcoming Month`" + `**
    * Provide actionable and strategic recommendations for the next month (e.g., areas of focus, process improvements, risk mitigation steps).
9.  **` + "`## Data Limitations`" + `** (If applicable)
    * If a comprehensive analysis is hindered by missing data (e.g., lack of historical data for trend analysis, vague task descriptions), note these limitations.

**Key Instructions:**
* **Strategic Insight:** Focus on providing insights, not just a data dump.
* **Data-Driven Analysis:** All conclusions must be supported by the provided Trello data.
* **Professional & Formal Tone:** Suitable for senior management.
* **Clarity and Readability:** Use markdown effectively with headings, bullet points, bolding, and tables if appropriate for data presentation (e.g., for a small list of key metrics).
`
	default: // General or ad-hoc report
		return dataContextPreamble + `You are a helpful AI Project Management Assistant. Your task is to analyze the provided Trello board data and generate a clear and informative project report based on the aspects highlighted by the user.

**Report Structure and Focus (General Guidance - adapt as needed):**
Please use markdown formatting. Structure your report logically based on the query, but generally consider including:

1.  **` + "`## Overall Summary`" + `** (A brief overview of the current project state based on the data.)
2.  **` + "`## Key Findings on [Specific Aspect Queried]`" + `** (If the user asked about specific tasks, lists, or labels, detail findings here.)
3.  **` + "`## Progress on Key Tasks & Milestones`" + `**
    * Detail progress made on critical tasks or towards achieving significant milestones visible in the data.
4.  **` + "`## Current Status Snapshot`" + `**
    * Clearly delineate what is 'To Do', 'In Progress', 'Blocked', or 'Completed' based on current list statuses.
5.  **` + "`## Team Activity Summary`" + `**
    * Summarize task assignments and movements. Focus on work distribution and flow. *Avoid subjective performance comments.*
6.  **` + "`## Identified Risks & Issues`" + `**
    * Highlight any tasks marked as blocked, overdue (if dates are available), or comments indicating problems.
7.  **` + "`## Actionable Insights & Recommendations`" + `** (If appropriate for the query)
    * Based on the analysis, suggest any clear next steps or areas needing attention.
8.  **` + "`## Data Limitations`" + `** (If applicable)
    * Note if the provided data is insufficient to fully address the user's query.

**Key Instructions:**
* **Directly Address the Query:** Ensure the report focuses on the specifics asked for, if this is an ad-hoc request.
* **Objectivity:** Base your report strictly on the provided Trello data. Do not invent information.
* **Clarity:** Use clear language and well-structured markdown.
`
	}
}
