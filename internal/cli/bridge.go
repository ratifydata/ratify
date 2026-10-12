package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"go.yaml.in/yaml/v3"
)

// Bridge sends CLI operations through the server's API-key middleware.
// It never connects to the database or assigns an organization locally.
type Bridge struct {
	baseURL string
	apiKey  string
	client  *http.Client
}

func NewBridge() *Bridge {
	return &Bridge{client: &http.Client{
		Timeout: 60 * time.Second,
		// Do not forward credentials or replay mutations to a redirected endpoint.
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}}
}

func (b *Bridge) Configure(cmd *cobra.Command) error {
	data, err := os.ReadFile(filepath.Join(".ratify", "config.yml"))
	if err != nil {
		return fmt.Errorf("read CLI config (run ratify auth set-key): %w", err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return fmt.Errorf("invalid CLI config: expected YAML with API_KEY and optional API_URL strings")
	}
	key := strings.TrimSpace(cfg.APIKey)
	if key == "" || strings.ContainsAny(key, "\r\n") {
		return fmt.Errorf("missing or invalid API_KEY; View docs on how to configure API key")
	}
	endpoint, err := cmd.Flags().GetString("api-url")
	if err != nil {
		return err
	}
	if endpoint == "" {
		endpoint = cfg.APIURL
	}
	if endpoint == "" {
		endpoint = "http://localhost:8080"
	}
	endpoint = strings.TrimRight(strings.TrimSpace(endpoint), "/")
	parsed, err := url.Parse(endpoint)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return fmt.Errorf("invalid API URL: expected an http or https server URL without credentials, query, or fragment")
	}
	b.baseURL, b.apiKey = endpoint, key
	return nil
}

func (b *Bridge) Do(ctx context.Context, method, path string, input, output any) error {
	var body io.Reader
	if input != nil {
		data, err := json.Marshal(input)
		if err != nil {
			return fmt.Errorf("encode API request: %w", err)
		}
		body = bytes.NewReader(data)
	}
	req, err := http.NewRequestWithContext(ctx, method, b.baseURL+"/api/cli/v1"+path, body)
	if err != nil {
		return fmt.Errorf("create API request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+b.apiKey)
	req.Header.Set("Accept", "application/json")
	if input != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := b.client.Do(req)
	if err != nil {
		return fmt.Errorf("request to Ratify API failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if resp.StatusCode == http.StatusUnauthorized {
			return fmt.Errorf("API authentication failed (401); check your saved API key")
		}
		data, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		var envelope struct {
			Message string `json:"message"`
		}
		message := strings.TrimSpace(string(data))
		if json.Unmarshal(data, &envelope) == nil && envelope.Message != "" {
			message = envelope.Message
		}
		message = strings.ReplaceAll(message, b.apiKey, "[redacted]")
		if message == "" {
			message = http.StatusText(resp.StatusCode)
		}
		return fmt.Errorf("Ratify API returned %d: %s", resp.StatusCode, message)
	}
	if output == nil {
		return nil
	}
	if err := json.NewDecoder(resp.Body).Decode(output); err != nil {
		return fmt.Errorf("decode API response: %w", err)
	}
	return nil
}
