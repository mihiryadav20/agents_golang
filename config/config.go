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

	// AI Foundry API configuration
	AIFoundryAPIKey     = "5A0S6uhOLsoYHEwSnTzsB9UhNo6WSjE6OCwePK1ze4mhc5soCiKCJQQJ99BEACHYHv6XJ3w3AAAAACOGtT4T"
	AIFoundryAPIURL     = "https://mihiryadav00-4718-resource.services.ai.azure.com/models"
	AIFoundryModel      = "Llama-4-Maverick-17B-128E-Instruct-FP8"
	AIFoundryAPIVersion = "2024-05-01-preview"
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
	Consumer.AdditionalAuthorizationUrlParams["name"] = "Trello AI Foundry Integration"
}
