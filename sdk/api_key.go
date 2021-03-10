package sendgrid

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// APIKey is a Sendgrid API key.
type APIKey struct {
	ID     string   `json:"api_key_id,omitempty"`
	APIKey string   `json:"api_key,omitempty"`
	Name   string   `json:"name,omitempty"`
	Scopes []string `json:"scopes,omitempty"`
}

func parseAPIKey(respBody string) (*APIKey, RequestError) {
	var body APIKey
	if err := json.Unmarshal([]byte(respBody), &body); err != nil {
		return nil, RequestError{
			StatusCode: http.StatusInternalServerError,
			Err:        fmt.Errorf("failed parsing API key: %w", err),
		}
	}

	return &body, RequestError{StatusCode: http.StatusOK, Err: nil}
}

// CreateAPIKey creates an APIKey and returns it.
func (c *Client) CreateAPIKey(name string, scopes []string) (*APIKey, RequestError) {
	if name == "" {
		return nil, RequestError{
			StatusCode: http.StatusInternalServerError,
			Err:        ErrNameRequired,
		}
	}

	respBody, statusCode, err := c.Post("POST", "/api_keys", APIKey{
		Name:   name,
		Scopes: scopes,
	})
	if err != nil {
		return nil, RequestError{
			StatusCode: http.StatusInternalServerError,
			Err:        fmt.Errorf("failed creating API key: %w", err),
		}
	}

	if statusCode >= 300 {
		return nil, RequestError{
			StatusCode: statusCode,
			Err:        fmt.Errorf("failed creating apiKey, status: %d, response: %s", statusCode, respBody),
		}
	}

	return parseAPIKey(respBody)
}

// ReadAPIKey retreives an APIKey and returns it.
func (c *Client) ReadAPIKey(id string) (*APIKey, RequestError) {
	if id == "" {
		return nil, RequestError{
			StatusCode: http.StatusInternalServerError,
			Err:        ErrAPIKeyIDRequired,
		}
	}

	respBody, _, err := c.Get("GET", "/api_keys/"+id)
	if err != nil {
		return nil, RequestError{
			StatusCode: http.StatusInternalServerError,
			Err:        fmt.Errorf("failed reading API key: %w", err),
		}
	}

	return parseAPIKey(respBody)
}

// UpdateAPIKey edits an APIKey and returns it.
func (c *Client) UpdateAPIKey(id, name string, scopes []string) (*APIKey, RequestError) {
	if id == "" {
		return nil, RequestError{
			StatusCode: http.StatusInternalServerError,
			Err:        ErrAPIKeyIDRequired,
		}
	}

	t := APIKey{}
	if name != "" {
		t.Name = name
	}

	if len(scopes) > 0 {
		t.Scopes = scopes
	}

	respBody, _, err := c.Post("PUT", "/api_keys/"+id, t)
	if err != nil {
		return nil, RequestError{
			StatusCode: http.StatusInternalServerError,
			Err:        fmt.Errorf("failed updating API key: %w", err),
		}
	}

	return parseAPIKey(respBody)
}

// DeleteAPIKey deletes an APIKey.
func (c *Client) DeleteAPIKey(id string) (bool, error) {
	if id == "" {
		return false, ErrAPIKeyIDRequired
	}

	if _, statusCode, err := c.Get("DELETE", "/api_keys/"+id); statusCode > 299 || err != nil {
		return false, fmt.Errorf("failed deleting API key: %w", err)
	}

	return true, nil
}
