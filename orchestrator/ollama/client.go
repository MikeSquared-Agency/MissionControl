package ollama

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

const DefaultBaseURL = "http://localhost:11434"

// Client provides methods to interact with the Ollama API
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// Model represents an Ollama model
type Model struct {
	Name       string    `json:"name"`
	ModifiedAt time.Time `json:"modified_at"`
	Size       int64     `json:"size"`
}

// TagsResponse is the response from /api/tags endpoint
type TagsResponse struct {
	Models []Model `json:"models"`
}

// NewClient creates a new Ollama client
func NewClient(baseURL string) *Client {
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	return &Client{
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: 5 * time.Second},
	}
}

// IsRunning checks if Ollama is accessible
func (c *Client) IsRunning() bool {
	resp, err := c.httpClient.Get(c.baseURL)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// ListModels returns all available models from Ollama
func (c *Client) ListModels() ([]Model, error) {
	resp, err := c.httpClient.Get(c.baseURL + "/api/tags")
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Ollama: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Ollama returned status %d", resp.StatusCode)
	}

	var tags TagsResponse
	if err := json.NewDecoder(resp.Body).Decode(&tags); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return tags.Models, nil
}

// HasModel checks if a specific model is available
func (c *Client) HasModel(name string) (bool, error) {
	models, err := c.ListModels()
	if err != nil {
		return false, err
	}
	for _, m := range models {
		if m.Name == name {
			return true, nil
		}
	}
	return false, nil
}

// GetModelNames returns just the names of available models
func (c *Client) GetModelNames() ([]string, error) {
	models, err := c.ListModels()
	if err != nil {
		return nil, err
	}
	names := make([]string, len(models))
	for i, m := range models {
		names[i] = m.Name
	}
	return names, nil
}
