package aifoundry

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"agents_go/config"
)

// Client is an AI Foundry API client
type Client struct {
	APIKey     string
	BaseURL    string
	APIVersion string
}

// Message represents a message in the chat
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatRequest represents a request to the AI Foundry chat API
type ChatRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Temperature float64   `json:"temperature,omitempty"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
}

// ChatResponse represents a response from the AI Foundry chat API
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

// NewClient creates a new AI Foundry client
func NewClient() *Client {
	return &Client{
		APIKey:     config.AIFoundryAPIKey,
		BaseURL:    config.AIFoundryAPIURL,
		APIVersion: config.AIFoundryAPIVersion,
	}
}

// SendChatMessage sends a simple chat message to the AI Foundry API
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
		Model:       config.AIFoundryModel,
		Messages:    messages,
		Temperature: 0.8,
		MaxTokens:   2048,
	}

	// Convert request to JSON
	reqBody, err := json.Marshal(chatReq)
	if err != nil {
		return "", fmt.Errorf("error marshaling request: %v", err)
	}

	// Create HTTP request with the correct endpoint structure
	endpoint := fmt.Sprintf("%s/chat/completions?api-version=%s", c.BaseURL, c.APIVersion)
	req, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(reqBody))
	if err != nil {
		return "", fmt.Errorf("error creating request: %v", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("api-key", c.APIKey)

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

// GenerateReport generates a report using the AI Foundry API
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
		Model:       config.AIFoundryModel,
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
	req.Header.Set("api-key", c.APIKey)

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
	if err := json.NewDecoder(bytes.NewReader(body)).Decode(&chatResp); err != nil {
		return "", fmt.Errorf("error unmarshaling response: %v", err)
	}

	// Check if we got any choices
	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("no response from model")
	}

	// Return the generated report
	return chatResp.Choices[0].Message.Content, nil
}

// formatBoardData converts the board data to a readable format for the LLM
func formatBoardData(boardData map[string]interface{}) (string, error) {
	// Extract board information
	board, ok := boardData["board"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("invalid board data format")
	}

	boardName, _ := board["name"].(string)
	boardDesc, _ := board["desc"].(string)

	// Extract lists
	lists, ok := boardData["lists"].([]interface{})
	if !ok {
		return "", fmt.Errorf("invalid lists data format")
	}

	// Extract cards
	cards, ok := boardData["cards"].([]interface{})
	if !ok {
		return "", fmt.Errorf("invalid cards data format")
	}

	// Extract members
	members, ok := boardData["members"].([]interface{})
	if !ok {
		return "", fmt.Errorf("invalid members data format")
	}

	// Extract actions
	actions, ok := boardData["actions"].([]interface{})
	if !ok {
		return "", fmt.Errorf("invalid actions data format")
	}

	// Build summary
	var summary string

	// Board info
	summary += fmt.Sprintf("# Board: %s\n\n", boardName)
	if boardDesc != "" {
		summary += fmt.Sprintf("Description: %s\n\n", boardDesc)
	}

	// Members
	summary += fmt.Sprintf("## Members (%d)\n\n", len(members))
	for _, m := range members {
		member, ok := m.(map[string]interface{})
		if !ok {
			continue
		}
		fullName, _ := member["fullName"].(string)
		username, _ := member["username"].(string)
		summary += fmt.Sprintf("- %s (@%s)\n", fullName, username)
	}
	summary += "\n"

	// Lists and cards
	summary += fmt.Sprintf("## Lists and Cards\n\n")
	
	// Create a map of list IDs to names
	listMap := make(map[string]string)
	for _, l := range lists {
		list, ok := l.(map[string]interface{})
		if !ok {
			continue
		}
		listID, _ := list["id"].(string)
		listName, _ := list["name"].(string)
		listMap[listID] = listName
	}

	// Group cards by list
	cardsByList := make(map[string][]map[string]interface{})
	for _, c := range cards {
		card, ok := c.(map[string]interface{})
		if !ok {
			continue
		}
		listID, _ := card["idList"].(string)
		if _, exists := cardsByList[listID]; !exists {
			cardsByList[listID] = []map[string]interface{}{}
		}
		cardsByList[listID] = append(cardsByList[listID], card)
	}

	// Output lists and their cards
	for _, l := range lists {
		list, ok := l.(map[string]interface{})
		if !ok {
			continue
		}
		listID, _ := list["id"].(string)
		listName, _ := list["name"].(string)
		
		summary += fmt.Sprintf("### List: %s\n\n", listName)
		
		// Get cards for this list
		listCards, exists := cardsByList[listID]
		if !exists || len(listCards) == 0 {
			summary += "No cards in this list.\n\n"
			continue
		}
		
		for _, card := range listCards {
			cardName, _ := card["name"].(string)
			cardDesc, _ := card["desc"].(string)
			cardDue, _ := card["due"].(string)
			cardLabels, _ := card["labels"].([]interface{})
			
			summary += fmt.Sprintf("#### Card: %s\n\n", cardName)
			
			if cardDesc != "" {
				summary += fmt.Sprintf("Description: %s\n\n", cardDesc)
			}
			
			if cardDue != "" {
				summary += fmt.Sprintf("Due: %s\n\n", cardDue)
			}
			
			if len(cardLabels) > 0 {
				summary += "Labels: "
				for i, l := range cardLabels {
					label, ok := l.(map[string]interface{})
					if !ok {
						continue
					}
					labelName, _ := label["name"].(string)
					labelColor, _ := label["color"].(string)
					
					if i > 0 {
						summary += ", "
					}
					summary += fmt.Sprintf("%s (%s)", labelName, labelColor)
				}
				summary += "\n\n"
			}
		}
	}

	// Recent activity
	summary += fmt.Sprintf("## Recent Activity (%d actions)\n\n", len(actions))
	
	// Only include the 20 most recent actions to keep the summary concise
	maxActions := 20
	if len(actions) > maxActions {
		actions = actions[:maxActions]
	}
	
	for _, a := range actions {
		action, ok := a.(map[string]interface{})
		if !ok {
			continue
		}
		
		actionType, _ := action["type"].(string)
		date, _ := action["date"].(string)
		
		// Get member who performed the action
		memberCreator, ok := action["memberCreator"].(map[string]interface{})
		if !ok {
			continue
		}
		memberName, _ := memberCreator["fullName"].(string)
		
		// Get data about the action
		data, ok := action["data"].(map[string]interface{})
		if !ok {
			continue
		}
		
		// Format the action based on its type
		var actionDesc string
		
		switch actionType {
		case "createCard":
			card, _ := data["card"].(map[string]interface{})
			cardName, _ := card["name"].(string)
			list, _ := data["list"].(map[string]interface{})
			listName, _ := list["name"].(string)
			actionDesc = fmt.Sprintf("created card '%s' in list '%s'", cardName, listName)
		
		case "updateCard":
			card, _ := data["card"].(map[string]interface{})
			cardName, _ := card["name"].(string)
			
			// Check if card was moved between lists
			listAfter, listAfterOk := data["listAfter"].(map[string]interface{})
			listBefore, listBeforeOk := data["listBefore"].(map[string]interface{})
			
			if listAfterOk && listBeforeOk {
				listAfterName, _ := listAfter["name"].(string)
				listBeforeName, _ := listBefore["name"].(string)
				actionDesc = fmt.Sprintf("moved card '%s' from '%s' to '%s'", cardName, listBeforeName, listAfterName)
			} else {
				actionDesc = fmt.Sprintf("updated card '%s'", cardName)
			}
			
		case "commentCard":
			card, _ := data["card"].(map[string]interface{})
			cardName, _ := card["name"].(string)
			text, _ := action["text"].(string)
			actionDesc = fmt.Sprintf("commented on '%s': '%s'", cardName, text)
			
		case "addMemberToCard", "removeMemberFromCard":
			card, _ := data["card"].(map[string]interface{})
			cardName, _ := card["name"].(string)
			member, _ := data["member"].(map[string]interface{})
			memberName, _ := member["name"].(string)
			
			if actionType == "addMemberToCard" {
				actionDesc = fmt.Sprintf("added %s to card '%s'", memberName, cardName)
			} else {
				actionDesc = fmt.Sprintf("removed %s from card '%s'", memberName, cardName)
			}
			
		default:
			board, _ := data["board"].(map[string]interface{})
			boardName, _ := board["name"].(string)
			actionDesc = fmt.Sprintf("performed action '%s' on board '%s'", actionType, boardName)
		}
		
		// Format the date
		t, err := time.Parse(time.RFC3339, date)
		if err == nil {
			date = t.Format("2006-01-02 15:04:05")
		}
		
		summary += fmt.Sprintf("- %s: %s %s\n", date, memberName, actionDesc)
	}

	return summary, nil
}

// getReportSystemPrompt returns the system prompt for the specified report type
func getReportSystemPrompt(reportType string) string {
	switch reportType {
	case "weekly":
		return `You are an AI assistant that generates weekly Trello board reports.
Your task is to analyze the provided Trello board data and generate a comprehensive weekly report.
The report should include:

1. Executive Summary: A brief overview of the board's status and progress this week.
2. Key Metrics: Number of cards in each list, cards completed this week, new cards added.
3. Progress Analysis: Analyze the movement of cards between lists, focusing on completion rates.
4. Team Activity: Highlight active contributors and their main contributions.
5. Blockers and Issues: Identify cards that haven't moved in a while or have warning labels.
6. Next Week's Outlook: Suggest priorities based on due dates and card positions.
7. Recommendations: Provide 2-3 actionable suggestions to improve workflow or address issues.

Format the report in Markdown with clear headings, bullet points, and sections.
Be concise but thorough, focusing on insights rather than just listing facts.
The total report should be approximately 500-800 words.`

	case "monthly":
		return `You are an AI assistant that generates monthly Trello board reports.
Your task is to analyze the provided Trello board data and generate a comprehensive monthly report.
The report should include:

1. Executive Summary: A high-level overview of the month's achievements, challenges, and board status.
2. Monthly Metrics: 
   - Cards completed vs. created
   - Average completion time
   - Distribution of cards across lists
   - Member contribution statistics
3. Progress Analysis: 
   - Major milestones achieved
   - Comparison with previous month (if patterns can be detected)
   - Workflow efficiency analysis
4. Team Performance: 
   - Highlight key contributors
   - Collaboration patterns
   - Workload distribution
5. Issue Analysis:
   - Recurring blockers or bottlenecks
   - Cards with long cycle times
   - Potential process improvements
6. Strategic Recommendations:
   - 3-5 actionable insights to improve productivity
   - Suggested process adjustments
   - Resource allocation suggestions
7. Next Month Outlook:
   - Upcoming deadlines and priorities
   - Potential risks to monitor
   - Opportunities for improvement

Format the report in Markdown with clear headings, bullet points, and sections.
Include both quantitative metrics and qualitative insights.
The total report should be approximately 1000-1500 words.`

	default:
		return `You are an AI assistant that generates Trello board reports.
Your task is to analyze the provided Trello board data and generate a comprehensive report.
Focus on providing actionable insights about the board's status, progress, and team activity.
Format the report in Markdown with clear headings and sections.`
	}
}
