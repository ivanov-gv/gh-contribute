// Package auth implements GitHub App Device Authorization Flow (RFC 8628).
package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	gogithub "github.com/google/go-github/v69/github"
)

const (
	// clientID is the GitHub App client ID used in the Device Authorization Flow.
	clientID = "Iv23li5Jz2VqBUOWp6u6"

	// githubDeviceCodeURL is the endpoint for requesting a device + user code.
	githubDeviceCodeURL = "https://github.com/login/device/code"

	// githubTokenURL is the endpoint polled to exchange a device code for an access token.
	githubTokenURL = "https://github.com/login/oauth/access_token"
)

// RunDeviceFlow executes the GitHub App Device Authorization Flow (RFC 8628)
// and returns the resulting user access token.
//
// onDeviceCode is called with the user code and verification URI before polling begins,
// allowing the caller to display them and attempt to open a browser.
func RunDeviceFlow(onDeviceCode func(userCode, verificationURI string)) (string, error) {
	// request device + user codes from GitHub
	codeResp, err := requestDeviceCode()
	if err != nil {
		return "", fmt.Errorf("requestDeviceCode: %w", err)
	}

	// let caller display the code and URI before we start polling
	onDeviceCode(codeResp.UserCode, codeResp.VerificationURI)

	// poll until the user authorizes or the code expires
	token, err := pollForToken(codeResp)
	if err != nil {
		return "", fmt.Errorf("pollForToken: %w", err)
	}

	return token, nil
}

// GetUsername returns the GitHub login name for the given token.
// Uses the go-github SDK — returns ErrTokenInvalid on 401.
func GetUsername(ctx context.Context, token string) (string, error) {
	client := gogithub.NewClient(nil).WithAuthToken(token)

	user, _, err := client.Users.Get(ctx, "")
	if err != nil {
		var ghErr *gogithub.ErrorResponse
		if errors.As(err, &ghErr) && ghErr.Response.StatusCode == http.StatusUnauthorized {
			return "", fmt.Errorf("client.Users.Get: %w", ErrTokenInvalid)
		}
		return "", fmt.Errorf("client.Users.Get: %w", err)
	}

	return user.GetLogin(), nil
}

// deviceCodeResponse holds the fields from the device code endpoint.
type deviceCodeResponse struct {
	DeviceCode      string `json:"device_code"`
	UserCode        string `json:"user_code"`
	VerificationURI string `json:"verification_uri"`
	ExpiresIn       int    `json:"expires_in"`
	Interval        int    `json:"interval"`
}

// requestDeviceCode POSTs to the device code endpoint and returns the parsed response.
func requestDeviceCode() (deviceCodeResponse, error) {
	req, err := http.NewRequest(http.MethodPost, githubDeviceCodeURL,
		strings.NewReader("client_id="+clientID))
	if err != nil {
		return deviceCodeResponse{}, fmt.Errorf("http.NewRequest: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return deviceCodeResponse{}, fmt.Errorf("http.DefaultClient.Do: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return deviceCodeResponse{}, fmt.Errorf("device code request returned status %d", resp.StatusCode)
	}

	var result deviceCodeResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return deviceCodeResponse{}, fmt.Errorf("json.NewDecoder.Decode: %w", err)
	}
	if result.DeviceCode == "" {
		return deviceCodeResponse{}, fmt.Errorf("empty device_code in response")
	}

	return result, nil
}

// tokenPollResponse holds the fields from the token polling endpoint.
type tokenPollResponse struct {
	AccessToken      string `json:"access_token"`
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
}

// pollForToken polls the token endpoint until authorization succeeds or the code expires.
// It respects the interval from the device code response and handles slow_down signals per RFC 8628.
func pollForToken(codeResp deviceCodeResponse) (string, error) {
	interval := time.Duration(codeResp.Interval) * time.Second
	if interval == 0 {
		// default per RFC 8628 §3.5
		interval = 5 * time.Second
	}

	deadline := time.Now().Add(time.Duration(codeResp.ExpiresIn) * time.Second)

	// build static polling request body
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
			// server requests slower polling — add 5s per RFC 8628 §3.5
			interval += 5 * time.Second
		case "expired_token":
			return "", fmt.Errorf("device code expired — please run 'gh contribute auth login' again")
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
func requestToken(body string) (tokenPollResponse, error) {
	req, err := http.NewRequest(http.MethodPost, githubTokenURL,
		strings.NewReader(body))
	if err != nil {
		return tokenPollResponse{}, fmt.Errorf("http.NewRequest: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return tokenPollResponse{}, fmt.Errorf("http.DefaultClient.Do: %w", err)
	}
	defer resp.Body.Close()

	var result tokenPollResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return tokenPollResponse{}, fmt.Errorf("json.NewDecoder.Decode: %w", err)
	}

	return result, nil
}
