package eventsub

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/pkg/browser"
)

const TwitchAuthorizeUrl = "https://id.twitch.tv/oauth2/authorize"

// AuthorizationCode is the code returned to us from the Twitch API after the user
// grants access to our app in the browser
// See: https://dev.twitch.tv/docs/authentication/getting-tokens-oauth/#authorization-code-grant-flow
type AuthorizationCode struct {
	Value  string
	Scopes []string
}

// PromptForCodeGrant spins up a small HTTP server on http://localhost:<port>, then
// opens a browser window that will send the user to Twitch, request that they grant
// access to the app with the given client ID and the requested scopes, then redirects
// them back to that server so that we can capture and parse the access code. The
// Twitch App must be configured with 'http://localhost:<port>/auth' as a valid
// redirect URI.
func PromptForCodeGrant(ctx context.Context, twitchAppClientId string, scopes []string, port uint16) (*AuthorizationCode, error) {
	// We'll run a tiny in-memory HTTP server so that the Twitch OAuth flow has
	// something to redirect to
	callbackUrl := fmt.Sprintf("http://localhost:%d/auth", port)

	// Generate a random CSRF token to supply in the authorization request: we'll
	// verify that Twitch sends back the same 'state' value in the redirect callback
	csrfToken := generateCsrfToken()

	// Don't run the server for more than a few minutes; if the user doesn't respond to
	// the authorization request, we want to abort eventually
	ctx, _ = context.WithTimeout(ctx, 5*time.Minute)

	// Handle GET /auth so we can capture the auth code grant
	codeChannel := make(chan *AuthorizationCode, 1)
	errorChannel := make(chan error, 1)
	handleAuthCallback := func(res http.ResponseWriter, req *http.Request) {
		if req.URL.Path != "/auth" {
			http.Error(res, "path not supported", http.StatusNotFound)
			return
		}
		if req.Method != http.MethodGet {
			http.Error(res, "method not supported", http.StatusMethodNotAllowed)
			return
		}
		code, err := parseCodeGrant(req, csrfToken, scopes)
		if err != nil {
			servePage(res, http.StatusBadRequest, "Authentication Failed", err.Error())
			errorChannel <- err
			return
		}
		servePage(res, http.StatusOK, "Authentication OK", fmt.Sprintf("A Twitch user access token has been granted. (%v)", code))
		codeChannel <- code
	}

	// Prepare a URL to a Twitch OAuth page that will request that the user authorize
	// our app to access their account with the given scopes, then redirect them back
	// to the callback URL that we specify
	authorizeUrl, err := url.Parse(TwitchAuthorizeUrl)
	if err != nil {
		log.Fatalf("failed to parse URL from %q: %v", TwitchAuthorizeUrl, err)
	}
	q := authorizeUrl.Query()
	q.Add("response_type", "code")
	q.Add("client_id", twitchAppClientId)
	q.Add("redirect_uri", callbackUrl)
	q.Add("scope", strings.Join(scopes, "+"))
	q.Add("state", csrfToken)
	authorizeUrl.RawQuery = q.Encode()

	// Start the server in a separate goroutine: once we get any result (or time out),
	// we'll close the server
	server := &http.Server{
		Addr:    fmt.Sprintf("localhost:%d", port),
		Handler: http.HandlerFunc(handleAuthCallback),
	}
	go func() {
		err := server.ListenAndServe()
		if err != http.ErrServerClosed {
			log.Fatalf("error running server: %v", err)
		}
	}()

	// Open a web browser, sending the user to the Twitch OAuth page: once they grant
	// access to our app, Twitch will redirect them user back to the callback URL we
	// specify, which in this case is our local HTTP server
	fmt.Printf("Opening web browser: %s\n", authorizeUrl.String())
	browser.OpenURL(authorizeUrl.String())

	// Block until we get a valid authorization code from an auth callback, the server
	// encounters an error while handling an auth callback, or we time out
	var code *AuthorizationCode
	select {
	case code = <-codeChannel:
		break
	case err = <-errorChannel:
		break
	case <-ctx.Done():
		break
	}
	if code == nil && err == nil {
		err = fmt.Errorf("timed out waiting for user authorization")
	}
	return code, err
}

// generateCsrfToken returns a cryptographically random hex string
func generateCsrfToken() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		log.Fatalf("failed to generate CSRF token: %v", err)
	}
	return hex.EncodeToString(bytes)
}

// parseCodeGrant examines our request URL and verifies that we've been given a valid
// code grant, parsing and returning the relevant data that was encoded in the URL
func parseCodeGrant(req *http.Request, csrfToken string, desiredScopes []string) (*AuthorizationCode, error) {
	// Verify that Twitch echoed our CSRF token back to us
	state := req.URL.Query().Get("state")
	if state == "" {
		return nil, fmt.Errorf("'state' value not found in URL query params")
	}
	if state != csrfToken {
		return nil, fmt.Errorf("CSRF token verification failed")
	}

	// Verify that the code grant includes all requested scopes
	scopeValue := req.URL.Query().Get("scope")
	if scopeValue == "" {
		return nil, fmt.Errorf("'scope' value not found in URL query params")
	}
	scopes := strings.Split(scopeValue, "+")
	if len(scopes) == 0 {
		return nil, fmt.Errorf("'scope' must specify at least one user scope")
	}
	for _, desiredScope := range desiredScopes {
		wasGranted := false
		for _, scope := range scopes {
			if scope == desiredScope {
				wasGranted = true
				break
			}
		}
		if !wasGranted {
			return nil, fmt.Errorf("required scope '%s' was not granted", desiredScope)
		}
	}

	// Parse the authorization code itself
	code := req.URL.Query().Get("code")
	if code == "" {
		return nil, fmt.Errorf("'code' value not found in URL query params")
	}

	return &AuthorizationCode{
		Value:  code,
		Scopes: scopes,
	}, nil
}

// servePage renders a simple HTML page so the user has some feedback and doesn't just
// get redirected to the void after granting access via twitch.tv
func servePage(res http.ResponseWriter, statusCode int, title string, message string) {
	pageTemplate := successPageTemplate
	if statusCode >= 300 {
		pageTemplate = errorPageTemplate
	}
	page := fmt.Sprintf(pageTemplate, title, title, message)
	res.Header().Set("Content-Type", "text/html; charset=utf-8")
	res.WriteHeader(statusCode)
	res.Write([]byte(page))
}

const successPageTemplate = `<!DOCTYPE html>
<html>
  <head>
    <title>%s</title>
  </head>
  <body>
    <h1>%s</h1>
	<p>%s</p>
	<p>This page will close automatically.</p>
	<script>
	  setTimeout(window.close, 0)
  	</script>
  </body>
</html>
`

const errorPageTemplate = `<!DOCTYPE html>
<html>
  <head>
    <title>%s</title>
  </head>
  <body>
    <h1>%s</h1>
	<p>%s</p>
	<p>You may now close this window.</p>
  </body>
</html>
`
