# Trello OAuth System in Go

This project demonstrates how to implement OAuth 1.0a authentication for Trello using Golang. It provides a complete web application that allows users to authenticate with their Trello accounts and obtain access tokens for making API calls.

## Features

- Complete OAuth 1.0a flow implementation for Trello
- Session management using secure cookies
- User-friendly UI with responsive design
- Access token display for API usage
- Logout functionality

## Prerequisites

- Go 1.16 or higher
- A Trello account

## Installation

1. Clone this repository
2. Install dependencies:

```
go mod tidy
```

## Configuration

The application uses the following Trello API credentials:

- API Key: `a2f217e66e60163384df3e891fd329a8`
- API Secret: `904e785848d1994523d17337b16a4473da7a9747690587d76f1b78e1dfa3779f`
- Callback URL: `http://localhost:5000/callback`

These are already configured in the application. If you want to use your own credentials, update the constants in `main.go`.

## Running the Application

```
go run main.go
```

The application will start on http://localhost:5000

## OAuth Flow

1. User visits the homepage and clicks "Connect to Trello"
2. Application obtains a request token from Trello
3. User is redirected to Trello to authorize the application
4. Trello redirects back to the callback URL with a verification code
5. Application exchanges the request token and verification code for an access token
6. Access token is stored in the session and displayed on the dashboard

## API Usage

Once you have obtained an access token, you can use it to make API calls to Trello on behalf of the user. Example:

```go
resp, err := consumer.Get(
    "https://api.trello.com/1/members/me",
    map[string]string{},
    &oauth.AccessToken{Token: accessToken, Secret: accessSecret},
)
```

## Dependencies

- [github.com/gorilla/mux](https://github.com/gorilla/mux) - HTTP router and dispatcher
- [github.com/gorilla/sessions](https://github.com/gorilla/sessions) - Cookie and session management
- [github.com/mrjones/oauth](https://github.com/mrjones/oauth) - OAuth 1.0a implementation

## License

MIT