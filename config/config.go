package config

import (
	"github.com/gorilla/sessions"
	"github.com/mrjones/oauth"
)

const (
	TrelloKey       = "a2f217e66e60163384df3e891fd329a8"
	TrelloSecret    = "904e785848d1994523d17337b16a4473da7a9747690587d76f1b78e1dfa3779f"
	CallbackURL     = "http://127.0.0.1:5001/callback"
	RequestTokenURL = "https://trello.com/1/OAuthGetRequestToken"
	AuthorizeURL    = "https://trello.com/1/OAuthAuthorizeToken"
	AccessTokenURL  = "https://trello.com/1/OAuthGetAccessToken"

	// Mistral API configuration via OpenRouter
	MistralAPIKey   = "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"
	MistralAPIURL   = "https://openrouter.ai/api/v1"
	MistralModel    = "mistralai/mistral-medium-3"
)

// Store will hold all session data
var Store = sessions.NewCookieStore([]byte("trello-oauth-secret-key"))

// Consumer is the global OAuth consumer
var Consumer *oauth.Consumer

// Init initializes the OAuth consumer
func Init() {
	Consumer = oauth.NewConsumer(
		TrelloKey,
		TrelloSecret,
		oauth.ServiceProvider{
			RequestTokenUrl:   RequestTokenURL,
			AuthorizeTokenUrl: AuthorizeURL,
			AccessTokenUrl:    AccessTokenURL,
		},
	)

	// Set the callback URL
	Consumer.AdditionalAuthorizationUrlParams["callback_url"] = CallbackURL
	// Request read and write permissions
	Consumer.AdditionalAuthorizationUrlParams["scope"] = "read,write"
	// Request token to never expire
	Consumer.AdditionalAuthorizationUrlParams["expiration"] = "never"
	// Set the app name
	Consumer.AdditionalAuthorizationUrlParams["name"] = "Trello OAuth Go App"
}
