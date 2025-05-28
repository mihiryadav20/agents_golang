package aifoundry

import (
	"context"
	"fmt"

	"agents_go/config"
	"github.com/Azure/azure-sdk-for-go/sdk/ai/azopenai"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
)

// AIFoundryClient is an AI Foundry API client
type AIFoundryClient struct {
	client       *azopenai.Client
	deploymentID string
}

// NewClient creates a new AI Foundry client
func NewClient() *AIFoundryClient {
	// Create the API key credential
	cred := azcore.NewKeyCredential(config.AIFoundryAPIKey)

	// Create the client with API key
	client, err := azopenai.NewClientWithKeyCredential(config.AIFoundryAPIURL, cred, nil)
	if err != nil {
		panic(fmt.Sprintf("Failed to create Azure OpenAI client: %v", err))
	}

	return &AIFoundryClient{
		client:       client,
		deploymentID: config.AIFoundryModel,
	}
}

// SendChatMessage sends a simple chat message to the AI Foundry API
func (c *AIFoundryClient) SendChatMessage(message string) (string, error) {
	// Create the messages
	systemMessage := azopenai.ChatRequestSystemMessage{
		Content: azopenai.NewChatRequestSystemMessageContent("You are a helpful assistant for Trello users. You provide concise and accurate information."),
	}

	userMessage := azopenai.ChatRequestUserMessage{
		Content: azopenai.NewChatRequestUserMessageContent(message),
	}

	// Create the request
	deploymentID := c.deploymentID // Create a copy to take address of
	request := azopenai.ChatCompletionsOptions{
		DeploymentName: &deploymentID,
		Messages: []azopenai.ChatRequestMessageClassification{
			&systemMessage,
			&userMessage,
		},
		Temperature: floatPtr(0.8),
		MaxTokens:   int32Ptr(2048),
	}

	// Send the request
	ctx := context.Background()
	resp, err := c.client.GetChatCompletions(ctx, request, nil)
	if err != nil {
		return "", fmt.Errorf("error sending request: %v", err)
	}

	// Extract the response content
	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no response choices returned")
	}

	// Get the content as string
	if resp.Choices[0].Message == nil || resp.Choices[0].Message.Content == nil {
		return "", fmt.Errorf("empty response content")
	}

	return *resp.Choices[0].Message.Content, nil
}

// GenerateReport generates a report using the AI Foundry API
func (c *AIFoundryClient) GenerateReport(boardData map[string]interface{}, reportType string) (string, error) {
	// Convert board data to a more readable format for the LLM
	boardSummary, err := formatBoardData(boardData)
	if err != nil {
		return "", fmt.Errorf("error formatting board data: %v", err)
	}

	// Create system prompt based on report type
	systemPrompt := getReportSystemPrompt(reportType)

	// Create the system message
	systemMessage := azopenai.ChatRequestSystemMessage{
		Content: azopenai.NewChatRequestSystemMessageContent(systemPrompt),
	}

	// Create the user message
	userMessage := azopenai.ChatRequestUserMessage{
		Content: azopenai.NewChatRequestUserMessageContent(boardSummary),
	}

	// Create the request
	deploymentID := c.deploymentID // Create a copy to take address of
	request := azopenai.ChatCompletionsOptions{
		DeploymentName: &deploymentID,
		Messages: []azopenai.ChatRequestMessageClassification{
			&systemMessage,
			&userMessage,
		},
		Temperature: floatPtr(0.7),
		MaxTokens:   int32Ptr(4000),
	}

	// Send the request
	ctx := context.Background()
	resp, err := c.client.GetChatCompletions(ctx, request, nil)
	if err != nil {
		return "", fmt.Errorf("error sending request: %v", err)
	}

	// Extract the response content
	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no response choices returned")
	}

	// Get the content as string
	if resp.Choices[0].Message == nil || resp.Choices[0].Message.Content == nil {
		return "", fmt.Errorf("empty response content")
	}

	return *resp.Choices[0].Message.Content, nil
}

// Helper functions for pointer types
func floatPtr(v float32) *float32 {
	return &v
}

func int32Ptr(v int32) *int32 {
	return &v
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

	// Helper function to safely extract slice of maps from either direct slice or wrapped in map with "items" key
	extractMaps := func(data interface{}) ([]map[string]interface{}, error) {
		// First, check if data is a map with an "items" key
		if dataMap, ok := data.(map[string]interface{}); ok {
			if items, ok := dataMap["items"]; ok {
				data = items
			}
		}

		switch v := data.(type) {
		case []map[string]interface{}:
			return v, nil
		case []interface{}:
			result := make([]map[string]interface{}, 0, len(v))
			for i, item := range v {
				if m, ok := item.(map[string]interface{}); ok {
					result = append(result, m)
				} else {
					return nil, fmt.Errorf("invalid item format at index %d", i)
				}
			}
			return result, nil
		default:
			return nil, fmt.Errorf("unexpected data type: %T", data)
		}
	}

	// Extract lists
	listsData, ok := boardData["lists"]
	if !ok {
		return "", fmt.Errorf("missing lists data")
	}
	lists, err := extractMaps(listsData)
	if err != nil {
		return "", fmt.Errorf("invalid lists data: %v", err)
	}

	// Extract cards
	cardsData, ok := boardData["cards"]
	if !ok {
		return "", fmt.Errorf("missing cards data")
	}
	cards, err := extractMaps(cardsData)
	if err != nil {
		return "", fmt.Errorf("invalid cards data: %v", err)
	}

	// Extract members
	membersData, ok := boardData["members"]
	if !ok {
		return "", fmt.Errorf("missing members data")
	}
	members, err := extractMaps(membersData)
	if err != nil {
		return "", fmt.Errorf("invalid members data: %v", err)
	}

	// Extract actions (activities)
	actions := []map[string]interface{}{}
	if activitiesData, ok := boardData["activities"]; ok {
		if acts, err := extractMaps(activitiesData); err == nil {
			actions = acts
		}
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
	for _, member := range members {
		memberName, _ := member["fullName"].(string)
		memberUsername, _ := member["username"].(string)
		summary += fmt.Sprintf("- %s (@%s)\n", memberName, memberUsername)
	}
	summary += "\n"

	// Lists and cards
	summary += fmt.Sprintf("## Lists and Cards\n\n")
	
	// Create a map of list IDs to names
	listMap := make(map[string]string)
	for _, list := range lists {
		listID, _ := list["id"].(string)
		listName, _ := list["name"].(string)
		listMap[listID] = listName
	}

	// Group cards by list
	cardsByList := make(map[string][]map[string]interface{})
	for _, card := range cards {
		listID, _ := card["idList"].(string)
		if _, exists := cardsByList[listID]; !exists {
			cardsByList[listID] = []map[string]interface{}{}
		}
		cardsByList[listID] = append(cardsByList[listID], card)
	}

	// Output lists and their cards
	for _, list := range lists {
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
				for i, labelObj := range cardLabels {
					if label, ok := labelObj.(map[string]interface{}); ok {
						labelName, _ := label["name"].(string)
						labelColor, _ := label["color"].(string)
						
						if i > 0 {
							summary += ", "
						}
						summary += fmt.Sprintf("%s (%s)", labelName, labelColor)
					}
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
	
	for _, action := range actions {
		actionType, _ := action["type"].(string)
		date, _ := action["date"].(string)
		
		// Get member who performed the action
		memberCreator, _ := action["memberCreator"].(map[string]interface{})
		memberName, _ := memberCreator["fullName"].(string)
		
		// Get data about the action
		data, _ := action["data"].(map[string]interface{})
		
		// Format the action based on its type
		var actionDesc string
		
		switch actionType {
		case "createCard":
			card, _ := data["card"].(map[string]interface{})
			cardName, _ := card["name"].(string)
			list, _ := data["list"].(map[string]interface{})
			listName, _ := list["name"].(string)
			actionDesc = fmt.Sprintf("Created card '%s' in list '%s'", cardName, listName)
		case "updateCard":
			card, _ := data["card"].(map[string]interface{})
			cardName, _ := card["name"].(string)
			if listAfter, ok := data["listAfter"].(map[string]interface{}); ok {
				listAfterName, _ := listAfter["name"].(string)
				listBefore, _ := data["listBefore"].(map[string]interface{})
				listBeforeName, _ := listBefore["name"].(string)
				actionDesc = fmt.Sprintf("Moved card '%s' from list '%s' to '%s'", cardName, listBeforeName, listAfterName)
			} else {
				actionDesc = fmt.Sprintf("Updated card '%s'", cardName)
			}
		case "commentCard":
			card, _ := data["card"].(map[string]interface{})
			cardName, _ := card["name"].(string)
			text, _ := data["text"].(string)
			actionDesc = fmt.Sprintf("Commented on card '%s': %s", cardName, text)
		default:
			actionDesc = fmt.Sprintf("Action of type '%s'", actionType)
		}
		
		// Add the action to the summary
		summary += fmt.Sprintf("- %s: %s (%s)\n", memberName, actionDesc, date)
	}
	
	return summary, nil
}

// getReportSystemPrompt returns the system prompt for the specified report type
func getReportSystemPrompt(reportType string) string {
	switch reportType {
	case "weekly":
		return `You are an AI assistant that generates weekly reports for Trello boards. 
Your task is to analyze the board data provided and create a comprehensive weekly report.

The report should include:
1. A summary of the board's current state
2. Progress made during the week (completed tasks, moved cards)
3. Pending tasks and their status
4. Any blockers or issues identified
5. Recommendations for the upcoming week

Be concise but thorough. Use markdown formatting to make the report readable.
Start with an executive summary, then break down the details by list/category.
Highlight important metrics and trends.

Your report should be professional and actionable, providing clear insights into the project's progress.`

	case "monthly":
		return `You are an AI assistant that generates monthly reports for Trello boards. 
Your task is to analyze the board data provided and create a comprehensive monthly report.

The report should include:
1. An executive summary of the month's progress
2. Key achievements and milestones reached
3. Detailed analysis of completed work
4. Current status of ongoing tasks
5. Blockers and challenges encountered
6. Trends and patterns observed
7. Strategic recommendations for the next month

Use markdown formatting to structure the report clearly.
Include metrics where possible, such as completion rates, task distribution, etc.
Compare the current state with previous periods if the data allows.

Your report should be thorough, insightful, and provide strategic value to the project stakeholders.`

	default:
		return `You are an AI assistant that generates reports for Trello boards.
Analyze the board data provided and create a comprehensive report.
Use markdown formatting to make the report readable and well-structured.
Focus on providing actionable insights and clear status updates.`
	}
}
