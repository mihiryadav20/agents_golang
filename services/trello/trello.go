package trello

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"reflect"
	"time"

	"agents_go/config"

	"github.com/mrjones/oauth"
)

// Board represents a Trello board with its basic information
type Board struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"desc"`
	URL         string `json:"url"`
	ShortURL    string `json:"shortUrl"`
}

// List represents a Trello list within a board
type List struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Closed  bool   `json:"closed"`
	BoardID string `json:"idBoard"`
	Pos     int    `json:"pos"`
}

// Card represents a Trello card within a list
type Card struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"desc"`
	Closed      bool      `json:"closed"`
	BoardID     string    `json:"idBoard"`
	ListID      string    `json:"idList"`
	Due         *time.Time `json:"due"`
	Labels      []Label   `json:"labels"`
	Members     []string  `json:"idMembers"`
	Created     time.Time `json:"dateLastActivity"`
}

// Label represents a Trello label
type Label struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Color   string `json:"color"`
	BoardID string `json:"idBoard"`
}

// Member represents a Trello member
type Member struct {
	ID        string `json:"id"`
	FullName  string `json:"fullName"`
	Username  string `json:"username"`
	AvatarURL string `json:"avatarUrl"`
}

// Client is a Trello API client
type Client struct {
	AccessToken  string
	AccessSecret string
}

// NewClient creates a new Trello client
func NewClient(accessToken, accessSecret string) *Client {
	return &Client{
		AccessToken:  accessToken,
		AccessSecret: accessSecret,
	}
}

// GetBoards returns all boards for the authenticated user
func (c *Client) GetBoards() ([]Board, error) {
	token := &oauth.AccessToken{
		Token:  c.AccessToken,
		Secret: c.AccessSecret,
	}

	resp, err := config.Consumer.Get(
		"https://api.trello.com/1/members/me/boards",
		map[string]string{"fields": "name,desc,url,shortUrl"},
		token,
	)
	if err != nil {
		return nil, fmt.Errorf("error getting boards: %v", err)
	}
	defer resp.Body.Close()

	var boards []Board
	if err := json.NewDecoder(resp.Body).Decode(&boards); err != nil {
		return nil, fmt.Errorf("error parsing boards data: %v", err)
	}

	return boards, nil
}

// GetBoardDetails returns detailed information about a specific board
func (c *Client) GetBoardDetails(boardID string) (*Board, error) {
	token := &oauth.AccessToken{
		Token:  c.AccessToken,
		Secret: c.AccessSecret,
	}

	resp, err := config.Consumer.Get(
		fmt.Sprintf("https://api.trello.com/1/boards/%s", boardID),
		map[string]string{"fields": "name,desc,url,shortUrl"},
		token,
	)
	if err != nil {
		return nil, fmt.Errorf("error getting board details: %v", err)
	}
	defer resp.Body.Close()

	var board Board
	if err := json.NewDecoder(resp.Body).Decode(&board); err != nil {
		return nil, fmt.Errorf("error parsing board data: %v", err)
	}

	return &board, nil
}

// GetLists returns all lists for a specific board
func (c *Client) GetLists(boardID string) ([]List, error) {
	token := &oauth.AccessToken{
		Token:  c.AccessToken,
		Secret: c.AccessSecret,
	}

	resp, err := config.Consumer.Get(
		fmt.Sprintf("https://api.trello.com/1/boards/%s/lists", boardID),
		map[string]string{"fields": "name,closed,idBoard,pos"},
		token,
	)
	if err != nil {
		return nil, fmt.Errorf("error getting lists: %v", err)
	}
	defer resp.Body.Close()

	var lists []List
	if err := json.NewDecoder(resp.Body).Decode(&lists); err != nil {
		return nil, fmt.Errorf("error parsing lists data: %v", err)
	}

	return lists, nil
}

// GetCards returns all cards for a specific board
func (c *Client) GetCards(boardID string) ([]Card, error) {
	token := &oauth.AccessToken{
		Token:  c.AccessToken,
		Secret: c.AccessSecret,
	}

	resp, err := config.Consumer.Get(
		fmt.Sprintf("https://api.trello.com/1/boards/%s/cards", boardID),
		map[string]string{
			"fields": "name,desc,closed,idBoard,idList,due,labels,idMembers,dateLastActivity",
			"members": "true",
			"member_fields": "fullName,username,avatarUrl",
		},
		token,
	)
	if err != nil {
		return nil, fmt.Errorf("error getting cards: %v", err)
	}
	defer resp.Body.Close()

	var cards []Card
	if err := json.NewDecoder(resp.Body).Decode(&cards); err != nil {
		return nil, fmt.Errorf("error parsing cards data: %v", err)
	}

	return cards, nil
}

// GetBoardMembers returns all members of a specific board
func (c *Client) GetBoardMembers(boardID string) ([]Member, error) {
	token := &oauth.AccessToken{
		Token:  c.AccessToken,
		Secret: c.AccessSecret,
	}

	resp, err := config.Consumer.Get(
		fmt.Sprintf("https://api.trello.com/1/boards/%s/members", boardID),
		map[string]string{"fields": "fullName,username,avatarUrl"},
		token,
	)
	if err != nil {
		return nil, fmt.Errorf("error getting board members: %v", err)
	}
	defer resp.Body.Close()

	var members []Member
	if err := json.NewDecoder(resp.Body).Decode(&members); err != nil {
		return nil, fmt.Errorf("error parsing members data: %v", err)
	}

	return members, nil
}

// GetBoardActivity returns recent activity for a specific board
func (c *Client) GetBoardActivity(boardID string, since time.Time) ([]map[string]interface{}, error) {
	token := &oauth.AccessToken{
		Token:  c.AccessToken,
		Secret: c.AccessSecret,
	}

	params := map[string]string{
		"limit": "50",
	}
	
	if !since.IsZero() {
		params["since"] = since.Format(time.RFC3339)
	}

	resp, err := config.Consumer.Get(
		fmt.Sprintf("https://api.trello.com/1/boards/%s/actions", boardID),
		params,
		token,
	)
	if err != nil {
		return nil, fmt.Errorf("error getting board activity: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %v", err)
	}

	var activities []map[string]interface{}
	if err := json.Unmarshal(body, &activities); err != nil {
		return nil, fmt.Errorf("error parsing activity data: %v", err)
	}

	return activities, nil
}

// GetBoardData fetches all relevant data for a board report
func (c *Client) GetBoardData(boardID string, since time.Time) (map[string]interface{}, error) {
	board, err := c.GetBoardDetails(boardID)
	if err != nil {
		return nil, err
	}

	lists, err := c.GetLists(boardID)
	if err != nil {
		return nil, err
	}

	cards, err := c.GetCards(boardID)
	if err != nil {
		return nil, err
	}

	members, err := c.GetBoardMembers(boardID)
	if err != nil {
		return nil, err
	}

	activities, err := c.GetBoardActivity(boardID, since)
	if err != nil {
		log.Printf("Warning: Could not fetch board activities: %v", err)
		activities = []map[string]interface{}{}
	}

	// Convert board to map
	boardData, err := convertToMap(board)
	if err != nil {
		return nil, fmt.Errorf("error converting board data: %v", err)
	}

	// Convert lists to map
	listsData, err := convertToMap(lists)
	if err != nil {
		return nil, fmt.Errorf("error converting lists data: %v", err)
	}

	// Convert cards to map
	cardsData, err := convertToMap(cards)
	if err != nil {
		return nil, fmt.Errorf("error converting cards data: %v", err)
	}

	// Convert members to map
	membersData, err := convertToMap(members)
	if err != nil {
		return nil, fmt.Errorf("error converting members data: %v", err)
	}

	return map[string]interface{}{
		"board":      boardData,
		"lists":      listsData,
		"cards":      cardsData,
		"members":    membersData,
		"activities": activities,
	}, nil
}

// convertToMap converts a struct to a map using JSON marshaling/unmarshaling
func convertToMap(data interface{}) (map[string]interface{}, error) {
	if data == nil {
		return make(map[string]interface{}), nil
	}
	
	// Marshal the data to JSON
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("error marshaling data: %v", err)
	}
	
	// For slices, we need to handle them differently
	rt := reflect.TypeOf(data)
	if rt.Kind() == reflect.Slice || rt.Kind() == reflect.Array {
		// Unmarshal the slice
		var items []interface{}
		if err := json.Unmarshal(jsonData, &items); err != nil {
			return nil, fmt.Errorf("error unmarshaling slice data: %v", err)
		}
		return map[string]interface{}{"items": items}, nil
	}
	
	// Not a slice, unmarshal as a map
	var result map[string]interface{}
	if err := json.Unmarshal(jsonData, &result); err != nil {
		return nil, fmt.Errorf("error unmarshaling data: %v", err)
	}
	
	return result, nil
}
