package ondemand

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
)

func ExtractFirstURL(data []byte) string {
	urlPattern := regexp.MustCompile(`https?://[^\s<>"]+`)

	// Find first match
	match := urlPattern.Find(data)
	if match == nil {
		return ""
	}

	return strings.Split(string(match), ")")[0]
}

// Response structures for better type safety
type SessionResponse struct {
	Data struct {
		ID string `json:"id"`
	} `json:"data"`
}

type QueryResponse struct {
	Data struct {
		Response string `json:"response"`
	} `json:"data"`
}

func OnDemand(query string) (string, error) {
	apiKey := "8mY1Fmb0h27x5osVfChQsIqwjHiVblvS"
	externalUserID := "agent-1715119442"

	// Create Chat Session
	sessionID, err := createChatSession(apiKey, externalUserID)
	if err != nil {
		return "", fmt.Errorf("failed to create chat session: %w", err)
	}

	var result string

	// Submit Query
	if result, err = submitQuery(apiKey, sessionID, fmt.Sprintf(`"%s"? .Sanitize the text as it most likely pdf extracted plaintext and generate a link to an audio narration of this text.`, query)); err != nil {
		return "", fmt.Errorf("failed to submit query: %w", err)
	}

	return result, nil
}

func createChatSession(apiKey, externalUserID string) (string, error) {
	if apiKey == "" || externalUserID == "" {
		return "", errors.New("apiKey and externalUserID cannot be empty")
	}

	url := "https://api.on-demand.io/chat/v1/sessions"
	body := map[string]interface{}{
		"pluginIds":      []string{},
		"externalUserId": externalUserID,
	}

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request body: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("apikey", apiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 201 {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(body))
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	var sessionResp SessionResponse
	if err := json.Unmarshal(respBody, &sessionResp); err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if sessionResp.Data.ID == "" {
		return "", errors.New("session ID not found in response")
	}

	return sessionResp.Data.ID, nil
}

func submitQuery(apiKey, sessionID, query string) (string, error) {
	if apiKey == "" || sessionID == "" || query == "" {
		return "", errors.New("apiKey, sessionID, and query cannot be empty")
	}

	url := fmt.Sprintf("https://api.on-demand.io/chat/v1/sessions/%s/query", sessionID)
	body := map[string]interface{}{
		"endpointId":   "predefined-openai-gpt4o",
		"query":        query,
		"pluginIds":    []string{"plugin-1712327325", "plugin-1715119442"},
		"responseMode": "sync",
	}

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request body: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("apikey", apiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(body))
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	firsturl := ExtractFirstURL(respBody)
	return firsturl, nil
}
