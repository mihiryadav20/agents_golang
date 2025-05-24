package handlers

import (
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"path/filepath"

	"agents_go/config"

	"github.com/mrjones/oauth"
)

// Templates holds all parsed templates
var Templates map[string]*template.Template

// InitTemplates initializes all templates
func InitTemplates() {
	Templates = make(map[string]*template.Template)
	baseTemplate := filepath.Join("templates", "base.html")
	
	// Parse each template with the base template
	templateFiles := []string{"home.html", "dashboard.html"}
	for _, file := range templateFiles {
		templatePath := filepath.Join("templates", file)
		tmpl, err := template.ParseFiles(baseTemplate, templatePath)
		if err != nil {
			log.Fatalf("Error parsing template %s: %v", file, err)
		}
		Templates[file] = tmpl
	}
}

// HomeHandler displays the home page with login link
func HomeHandler(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"Title": "Trello OAuth",
	}
	Templates["home.html"].Execute(w, data)
}

// LoginHandler initiates the OAuth flow
func LoginHandler(w http.ResponseWriter, r *http.Request) {
	// Get a request token
	requestToken, url, err := config.Consumer.GetRequestTokenAndUrl(config.CallbackURL)
	if err != nil {
		log.Printf("Error getting request token: %v", err)
		http.Error(w, "Error connecting to Trello", http.StatusInternalServerError)
		return
	}

	// Store the request token in the session
	session, _ := config.Store.Get(r, "trello-oauth")
	session.Values["requestToken"] = requestToken.Token
	session.Values["requestSecret"] = requestToken.Secret
	session.Save(r, w)

	// Redirect the user to the authorization URL
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

// CallbackHandler handles the OAuth callback from Trello
func CallbackHandler(w http.ResponseWriter, r *http.Request) {
	// Get the request token from the session
	session, _ := config.Store.Get(r, "trello-oauth")
	requestToken, ok1 := session.Values["requestToken"].(string)
	requestSecret, ok2 := session.Values["requestSecret"].(string)

	if !ok1 || !ok2 {
		http.Error(w, "No request token found", http.StatusBadRequest)
		return
	}

	// Get the verification code from the URL
	verifier := r.URL.Query().Get("oauth_verifier")
	if verifier == "" {
		http.Error(w, "No verification code found", http.StatusBadRequest)
		return
	}

	// Exchange the request token for an access token
	accessToken, err := config.Consumer.AuthorizeToken(
		&oauth.RequestToken{Token: requestToken, Secret: requestSecret},
		verifier,
	)
	if err != nil {
		log.Printf("Error getting access token: %v", err)
		http.Error(w, "Error connecting to Trello", http.StatusInternalServerError)
		return
	}

	// Store the access token in the session
	session.Values["accessToken"] = accessToken.Token
	session.Values["accessSecret"] = accessToken.Secret
	session.Save(r, w)

	// Redirect to the dashboard
	http.Redirect(w, r, "/dashboard", http.StatusTemporaryRedirect)
}

// Board represents a Trello board structure
type Board struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"desc"`
	URL         string `json:"url"`
}

// DashboardHandler displays user information and boards after successful OAuth
func DashboardHandler(w http.ResponseWriter, r *http.Request) {
	// Check if the user is authenticated
	session, _ := config.Store.Get(r, "trello-oauth")
	accessToken, ok1 := session.Values["accessToken"].(string)
	accessSecret, ok2 := session.Values["accessSecret"].(string)

	if !ok1 || !ok2 {
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}

	// Create an access token object
	token := &oauth.AccessToken{
		Token:  accessToken,
		Secret: accessSecret,
	}

	// Make a request to get user information
	userResp, err := config.Consumer.Get(
		"https://api.trello.com/1/members/me",
		map[string]string{},
		token,
	)
	if err != nil {
		log.Printf("Error getting user info: %v", err)
		http.Error(w, "Error getting user information", http.StatusInternalServerError)
		return
	}
	defer userResp.Body.Close()

	// Make a request to get user's boards
	boardsResp, err := config.Consumer.Get(
		"https://api.trello.com/1/members/me/boards",
		map[string]string{"fields": "name,desc,url"},
		token,
	)
	if err != nil {
		log.Printf("Error getting boards: %v", err)
		http.Error(w, "Error getting boards", http.StatusInternalServerError)
		return
	}
	defer boardsResp.Body.Close()

	// Parse the boards response
	var boards []Board
	decoder := json.NewDecoder(boardsResp.Body)
	if err := decoder.Decode(&boards); err != nil {
		log.Printf("Error decoding boards response: %v", err)
		http.Error(w, "Error processing boards data", http.StatusInternalServerError)
		return
	}

	// Render the dashboard template with token information and boards
	data := map[string]interface{}{
		"Title":        "Trello Dashboard",
		"AccessToken":  accessToken,
		"AccessSecret": accessSecret,
		"Boards":       boards,
	}
	Templates["dashboard.html"].Execute(w, data)
}

// LogoutHandler clears the session and logs the user out
func LogoutHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := config.Store.Get(r, "trello-oauth")
	// Clear session
	session.Values = make(map[interface{}]interface{})
	session.Save(r, w)

	// Redirect to home page
	http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
}
