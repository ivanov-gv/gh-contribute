package github

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const graphqlEndpoint = "https://api.github.com/graphql"

// GraphQLClient makes raw GraphQL requests to the GitHub API
type GraphQLClient struct {
	token      string
	httpClient *http.Client
}

// NewGraphQLClient creates a new GraphQL client
func NewGraphQLClient(token string) *GraphQLClient {
	return &GraphQLClient{
		token:      token,
		httpClient: http.DefaultClient,
	}
}

// graphqlRequest is the JSON body sent to the GraphQL endpoint
type graphqlRequest struct {
	Query     string                 `json:"query"`
	Variables map[string]interface{} `json:"variables,omitempty"`
}

// graphqlResponse is the raw JSON response from the GraphQL endpoint
type graphqlResponse struct {
	Data   json.RawMessage `json:"data"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

// Query executes a GraphQL query and unmarshals the "data" field into result
func (c *GraphQLClient) Query(query string, variables map[string]interface{}, result interface{}) error {
	// marshal request body
	body, err := json.Marshal(graphqlRequest{Query: query, Variables: variables})
	if err != nil {
		return fmt.Errorf("json.Marshal request: %w", err)
	}

	// build HTTP request
	req, err := http.NewRequest("POST", graphqlEndpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("http.NewRequest: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")

	// execute
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("httpClient.Do: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("io.ReadAll: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("GraphQL HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	// parse response
	var gqlResp graphqlResponse
	if err := json.Unmarshal(respBody, &gqlResp); err != nil {
		return fmt.Errorf("json.Unmarshal response: %w", err)
	}

	if len(gqlResp.Errors) > 0 {
		return fmt.Errorf("GraphQL error: %s", gqlResp.Errors[0].Message)
	}

	// unmarshal data into caller's result struct
	if err := json.Unmarshal(gqlResp.Data, result); err != nil {
		return fmt.Errorf("json.Unmarshal data: %w", err)
	}

	return nil
}
