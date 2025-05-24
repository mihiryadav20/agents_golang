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

	// Return the generated report
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
	switch reportType {
	case "weekly":
		return `You are a professional project manager assistant. Your task is to analyze the provided Trello board data and generate a comprehensive weekly project report.

Focus on:
1. Progress made this week (completed tasks, moved cards)
2. Current status of the project (what's in progress, what's blocked)
3. Upcoming deadlines and priorities for next week
4. Any risks or blockers identified
5. Team performance and contributions

Your report should be well-structured, professional, and concise. Use markdown formatting for better readability.
Avoid making assumptions beyond what's in the data. If information is missing, note it as a limitation rather than inventing details.

The report should be suitable for presentation to stakeholders and team members.`

	case "monthly":
		return `You are a professional project manager assistant. Your task is to analyze the provided Trello board data and generate a comprehensive monthly project report.

Focus on:
1. Overall project status and health
2. Major achievements and milestones reached this month
3. Key metrics (completion rate, velocity, etc.)
4. Trends and patterns observed over the month
5. Resource utilization and team performance
6. Risks, issues, and mitigation strategies
7. Recommendations for the upcoming month

Your report should be detailed yet concise, with an executive summary at the beginning.
Use markdown formatting for better readability, including sections, bullet points, and emphasis where appropriate.
Avoid making assumptions beyond what's in the data. If information is missing, note it as a limitation rather than inventing details.

The report should be suitable for presentation to senior management and stakeholders.`

	default:
		return `You are a professional project manager assistant. Your task is to analyze the provided Trello board data and generate a comprehensive project report.

Focus on:
1. Current project status
2. Progress on key tasks and milestones
3. Team contributions and performance
4. Risks and issues identified
5. Recommendations for improvement

Your report should be well-structured, professional, and concise. Use markdown formatting for better readability.
Avoid making assumptions beyond what's in the data. If information is missing, note it as a limitation rather than inventing details.`
	}
}
