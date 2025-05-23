package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"github.com/mrjones/oauth"
)

const (
	trelloKey       = "a2f217e66e60163384df3e891fd329a8"
	trelloSecret    = "904e785848d1994523d17337b16a4473da7a9747690587d76f1b78e1dfa3779f"
	callbackURL     = "http://localhost:5000/callback"
	requestTokenURL = "https://trello.com/1/OAuthGetRequestToken"
	authorizeURL    = "https://trello.com/1/OAuthAuthorizeToken"
	accessTokenURL  = "https://trello.com/1/OAuthGetAccessToken"
)

// Store will hold all session data
var store = sessions.NewCookieStore([]byte("trello-oauth-secret-key"))

// Global OAuth consumer
var consumer *oauth.Consumer

// Initialize OAuth consumer
func init() {
	consumer = oauth.NewConsumer(
		trelloKey,
		trelloSecret,
		oauth.ServiceProvider{
			RequestTokenUrl:   requestTokenURL,
			AuthorizeTokenUrl: authorizeURL,
			AccessTokenUrl:    accessTokenURL,
		},
	)

	// Set the callback URL
	consumer.AdditionalAuthorizationUrlParams["callback_url"] = callbackURL
	// Request read and write permissions
	consumer.AdditionalAuthorizationUrlParams["scope"] = "read,write"
	// Request token to never expire
	consumer.AdditionalAuthorizationUrlParams["expiration"] = "never"
	// Set the app name
	consumer.AdditionalAuthorizationUrlParams["name"] = "Trello OAuth Go App"
}

// HomeHandler displays the home page with login link
func HomeHandler(w http.ResponseWriter, r *http.Request) {
	html := `
	<!DOCTYPE html>
	<html>
	<head>
		<title>Trello OAuth</title>
		<style>
			body {
				font-family: Arial, sans-serif;
				margin: 40px;
				line-height: 1.6;
			}
			.container {
				max-width: 800px;
				margin: 0 auto;
				padding: 20px;
				border: 1px solid #ddd;
				border-radius: 5px;
			}
			h1 {
				color: #0079BF;
			}
			a {
				display: inline-block;
				background-color: #0079BF;
				color: white;
				padding: 10px 20px;
				text-decoration: none;
				border-radius: 5px;
				margin-top: 20px;
			}
			a:hover {
				background-color: #005A8C;
			}
		</style>
	</head>
	<body>
		<div class="container">
			<h1>Trello OAuth Example</h1>
			<p>Click the button below to authorize this application with your Trello account.</p>
			<a href="/login">Connect to Trello</a>
		</div>
	</body>
	</html>
	`
	fmt.Fprint(w, html)
}

// LoginHandler initiates the OAuth flow
func LoginHandler(w http.ResponseWriter, r *http.Request) {
	// Get a request token
	requestToken, url, err := consumer.GetRequestTokenAndUrl(callbackURL)
	if err != nil {
		log.Printf("Error getting request token: %v", err)
		http.Error(w, "Error connecting to Trello", http.StatusInternalServerError)
		return
	}

	// Store the request token in the session
	session, _ := store.Get(r, "trello-oauth")
	session.Values["requestToken"] = requestToken.Token
	session.Values["requestSecret"] = requestToken.Secret
	session.Save(r, w)

	// Redirect the user to the authorization URL
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

// CallbackHandler handles the OAuth callback from Trello
func CallbackHandler(w http.ResponseWriter, r *http.Request) {
	// Get the request token from the session
	session, _ := store.Get(r, "trello-oauth")
	requestToken := session.Values["requestToken"].(string)
	requestSecret := session.Values["requestSecret"].(string)

	// Get the verification code from the URL
	verifier := r.URL.Query().Get("oauth_verifier")
	if verifier == "" {
		http.Error(w, "No verification code found", http.StatusBadRequest)
		return
	}

	// Exchange the request token for an access token
	accessToken, err := consumer.AuthorizeToken(
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

// DashboardHandler displays user information after successful OAuth
func DashboardHandler(w http.ResponseWriter, r *http.Request) {
	// Check if the user is authenticated
	session, _ := store.Get(r, "trello-oauth")
	accessToken, ok := session.Values["accessToken"].(string)
	accessSecret, ok2 := session.Values["accessSecret"].(string)

	if !ok || !ok2 {
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}

	// Create an access token object
	token := &oauth.AccessToken{
		Token:  accessToken,
		Secret: accessSecret,
	}

	// Make a request to get user information
	resp, err := consumer.Get(
		"https://api.trello.com/1/members/me",
		map[string]string{},
		token,
	)
	if err != nil {
		log.Printf("Error getting user info: %v", err)
		http.Error(w, "Error getting user information", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	// Display the dashboard with user info
	html := fmt.Sprintf(`
	<!DOCTYPE html>
	<html>
	<head>
		<title>Trello Dashboard</title>
		<style>
			body {
				font-family: Arial, sans-serif;
				margin: 40px;
				line-height: 1.6;
			}
			.container {
				max-width: 800px;
				margin: 0 auto;
				padding: 20px;
				border: 1px solid #ddd;
				border-radius: 5px;
			}
			h1 {
				color: #0079BF;
			}
			.token-info {
				background-color: #f5f5f5;
				padding: 15px;
				border-radius: 5px;
				margin-top: 20px;
				font-family: monospace;
				word-break: break-all;
			}
			a {
				display: inline-block;
				background-color: #0079BF;
				color: white;
				padding: 10px 20px;
				text-decoration: none;
				border-radius: 5px;
				margin-top: 20px;
			}
			a:hover {
				background-color: #005A8C;
			}
		</style>
	</head>
	<body>
		<div class="container">
			<h1>Trello OAuth Success!</h1>
			<p>You have successfully connected to Trello. Your access token information:</p>
			<div class="token-info">
				<p><strong>Access Token:</strong> %s</p>
				<p><strong>Access Secret:</strong> %s</p>
			</div>
			<p>You can now use these tokens to make API calls to Trello on behalf of the user.</p>
			<a href="/logout">Logout</a>
		</div>
	</body>
	</html>
	`, accessToken, accessSecret)

	fmt.Fprint(w, html)
}

// LogoutHandler clears the session and logs the user out
func LogoutHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "trello-oauth")
	// Clear session
	session.Values = make(map[interface{}]interface{})
	session.Save(r, w)

	// Redirect to home page
	http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
}

func main() {
	// Create a new router
	r := mux.NewRouter()

	// Register route handlers
	r.HandleFunc("/", HomeHandler)
	r.HandleFunc("/login", LoginHandler)
	r.HandleFunc("/callback", CallbackHandler)
	r.HandleFunc("/dashboard", DashboardHandler)
	r.HandleFunc("/logout", LogoutHandler)

	// Start the server
	port := os.Getenv("PORT")
	if port == "" {
		port = "5000"
	}

	server := &http.Server{
		Addr:         "localhost:" + port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	fmt.Printf("Server starting on http://localhost:%s\n", port)
	log.Fatal(server.ListenAndServe())
}
