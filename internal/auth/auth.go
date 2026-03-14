// Package auth implements GitHub App Device Authorization Flow (RFC 8628)
// and manages the resulting user access token.
package auth

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const (
	// clientID is the GitHub App client ID used in the Device Authorization Flow.
	clientID = "Iv23li5Jz2VqBUOWp6u6"

	// tokenEnv is the environment variable for CI / non-interactive environments.
	tokenEnv = "MYEXT_TOKEN"

	// tokenRelPath is the token file path relative to the user home directory.
	tokenRelPath = ".config/gh-myext/token"
)

// LoadToken returns the GitHub App user access token.
// Priority: MYEXT_TOKEN env var → ~/.config/gh-myext/token file.
func LoadToken() (string, error) {
	// check env var first — suitable for CI / non-interactive use
	if t := os.Getenv(tokenEnv); t != "" {
		return t, nil
	}

	// load from config file
	path, err := tokenFilePath()
	if err != nil {
		return "", fmt.Errorf("tokenFilePath: %w", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", ErrNotAuthenticated
		}
		return "", fmt.Errorf("os.ReadFile [path='%s']: %w", path, err)
	}

	token := strings.TrimSpace(string(data))
	if token == "" {
		return "", ErrNotAuthenticated
	}

	return token, nil
}

// SaveToken persists the token to ~/.config/gh-myext/token with 0600 permissions.
func SaveToken(token string) error {
	path, err := tokenFilePath()
	if err != nil {
		return fmt.Errorf("tokenFilePath: %w", err)
	}

	// create parent directories with restricted permissions
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return fmt.Errorf("os.MkdirAll [dir='%s']: %w", filepath.Dir(path), err)
	}

	// write with owner-only read/write permissions
	if err := os.WriteFile(path, []byte(token), 0600); err != nil {
		return fmt.Errorf("os.WriteFile [path='%s']: %w", path, err)
	}

	return nil
}

// RunDeviceFlow executes the GitHub App Device Authorization Flow (RFC 8628)
// and returns the resulting user access token.
func RunDeviceFlow() (string, error) {
	// request device code and user code from GitHub
	codeResp, err := requestDeviceCode()
	if err != nil {
		return "", fmt.Errorf("requestDeviceCode: %w", err)
	}

	// prompt user — always to stderr so stdout stays clean
	fmt.Fprintf(os.Stderr, "Open: %s\nEnter code: %s\nWaiting for authorization...\n",
		codeResp.VerificationURI, codeResp.UserCode)

	// best-effort: open the browser (silently ignored if unavailable)
	openBrowser(codeResp.VerificationURI)

	// poll until the user authorizes or the code expires
	token, err := pollForToken(codeResp)
	if err != nil {
		return "", fmt.Errorf("pollForToken: %w", err)
	}

	return token, nil
}

// GetUsername calls GET /user with the given token and returns the GitHub login name.
func GetUsername(token string) (string, error) {
	req, err := http.NewRequest(http.MethodGet, "https://api.github.com/user", nil)
	if err != nil {
		return "", fmt.Errorf("http.NewRequest: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("http.DefaultClient.Do: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return "", ErrTokenInvalid
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GET /user returned status %d", resp.StatusCode)
	}

	var user struct {
		Login string `json:"login"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return "", fmt.Errorf("json.NewDecoder.Decode: %w", err)
	}

	return user.Login, nil
}

// deviceCodeResponse is the parsed response from the device code endpoint.
type deviceCodeResponse struct {
	DeviceCode      string `json:"device_code"`
	UserCode        string `json:"user_code"`
	VerificationURI string `json:"verification_uri"`
	ExpiresIn       int    `json:"expires_in"`
	Interval        int    `json:"interval"`
}

// requestDeviceCode POSTs to the device code endpoint and returns the parsed response.
func requestDeviceCode() (*deviceCodeResponse, error) {
	req, err := http.NewRequest(http.MethodPost,
		"https://github.com/login/device/code",
		strings.NewReader("client_id="+clientID))
	if err != nil {
		return nil, fmt.Errorf("http.NewRequest: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http.DefaultClient.Do: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("device code request returned status %d", resp.StatusCode)
	}

	var result deviceCodeResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("json.NewDecoder.Decode: %w", err)
	}
	if result.DeviceCode == "" {
		return nil, fmt.Errorf("empty device_code in response")
	}

	return &result, nil
}

// tokenPollResponse is the parsed response from the token polling endpoint.
type tokenPollResponse struct {
	AccessToken     string `json:"access_token"`
	Error           string `json:"error"`
	ErrorDescription string `json:"error_description"`
}

// pollForToken polls the token endpoint until authorization succeeds or the code expires.
// It respects the interval from the device code response and handles slow_down signals.
func pollForToken(codeResp *deviceCodeResponse) (string, error) {
	interval := time.Duration(codeResp.Interval) * time.Second
	if interval == 0 {
		// default per RFC 8628 §3.5
		interval = 5 * time.Second
	}

	deadline := time.Now().Add(time.Duration(codeResp.ExpiresIn) * time.Second)

	// build the static part of the token request body
	body := fmt.Sprintf(
		"client_id=%s&device_code=%s&grant_type=urn:ietf:params:oauth:grant-type:device_code",
		clientID, codeResp.DeviceCode,
	)

	for time.Now().Before(deadline) {
		time.Sleep(interval)

		pollResp, err := requestToken(body)
		if err != nil {
			return "", fmt.Errorf("requestToken: %w", err)
		}

		switch pollResp.Error {
		case "":
			// authorized — return the token
			return pollResp.AccessToken, nil
		case "authorization_pending":
			// user hasn't approved yet — keep polling at current interval
		case "slow_down":
			// server signals we're polling too fast — add 5s per RFC 8628
			interval += 5 * time.Second
		case "expired_token":
			return "", fmt.Errorf("device code expired — please run 'gh myext auth login' again")
		case "access_denied":
			return "", fmt.Errorf("authorization denied by user")
		default:
			return "", fmt.Errorf("token endpoint error: %s — %s",
				pollResp.Error, pollResp.ErrorDescription)
		}
	}

	return "", fmt.Errorf("timed out waiting for authorization")
}

// requestToken sends a single polling request to the OAuth token endpoint.
func requestToken(body string) (*tokenPollResponse, error) {
	req, err := http.NewRequest(http.MethodPost,
		"https://github.com/login/oauth/access_token",
		strings.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("http.NewRequest: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http.DefaultClient.Do: %w", err)
	}
	defer resp.Body.Close()

	var result tokenPollResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("json.NewDecoder.Decode: %w", err)
	}

	return &result, nil
}

// openBrowser attempts to open uri in the user's default browser.
// Failures are silently ignored — the user can open it manually using the printed URL.
func openBrowser(uri string) {
	var cmd string
	switch runtime.GOOS {
	case "linux":
		cmd = "xdg-open"
	case "darwin":
		cmd = "open"
	default:
		return
	}
	_ = exec.Command(cmd, uri).Start()
}

// tokenFilePath returns the absolute path to the token config file.
func tokenFilePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("os.UserHomeDir: %w", err)
	}
	return filepath.Join(home, tokenRelPath), nil
}
