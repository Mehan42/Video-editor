package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var ErrEgressBlocked = errors.New("llm egress is blocked by policy")

type Config struct {
	BaseURL          string
	Timeout          time.Duration
	MaxResponseBytes int64
	AllowChat        bool
}

type Client struct {
	baseURL          string
	timeout          time.Duration
	maxResponseBytes int64
	allowChat        bool
	httpClient       *http.Client
}

type Model struct {
	ID      string `json:"id"`
	Object  string `json:"object,omitempty"`
	OwnedBy string `json:"owned_by,omitempty"`
}

type ModelsResponse struct {
	Data []Model `json:"data"`
}

type Readiness struct {
	Provider     string   `json:"provider"`
	Status       string   `json:"status"`
	BaseURL      string   `json:"base_url"`
	Models       []Model  `json:"models"`
	Capabilities []string `json:"capabilities"`
	Error        string   `json:"error,omitempty"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
}

type ChatResponse struct {
	ID      string `json:"id"`
	Model   string `json:"model"`
	Choices []struct {
		Message Message `json:"message"`
	} `json:"choices"`
}

func NewBionicClient(cfg Config, httpClient *http.Client) (*Client, error) {
	baseURL, err := validateLoopbackBaseURL(cfg.BaseURL)
	if err != nil {
		return nil, err
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 10 * time.Second
	}
	if cfg.MaxResponseBytes <= 0 {
		cfg.MaxResponseBytes = 2 * 1024 * 1024
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: cfg.Timeout}
	}
	return &Client{
		baseURL:          strings.TrimRight(baseURL, "/"),
		timeout:          cfg.Timeout,
		maxResponseBytes: cfg.MaxResponseBytes,
		allowChat:        cfg.AllowChat,
		httpClient:       httpClient,
	}, nil
}

func (c *Client) Check(ctx context.Context) Readiness {
	models, err := c.ListModels(ctx)
	if err != nil {
		return Readiness{
			Provider: "lm-studio-bionic",
			Status:   "BLOCKED_PROVIDER",
			BaseURL:  c.baseURL,
			Models:   []Model{},
			Capabilities: []string{
				"chat/completions",
				"models",
			},
			Error: err.Error(),
		}
	}
	return Readiness{
		Provider: "lm-studio-bionic",
		Status:   "READY",
		BaseURL:  c.baseURL,
		Models:   models,
		Capabilities: []string{
			"chat/completions",
			"models",
		},
	}
}

func (c *Client) ListModels(ctx context.Context) ([]Model, error) {
	response, err := c.do(ctx, http.MethodGet, "/models", nil)
	if err != nil {
		return nil, err
	}
	var decoded ModelsResponse
	if err := json.Unmarshal(response, &decoded); err != nil {
		return nil, fmt.Errorf("decode models response: %w", err)
	}
	if decoded.Data == nil {
		return nil, errors.New("models response has no data")
	}
	return decoded.Data, nil
}

func (c *Client) Chat(ctx context.Context, request ChatRequest) (ChatResponse, error) {
	if !c.allowChat {
		return ChatResponse{}, ErrEgressBlocked
	}
	if strings.TrimSpace(request.Model) == "" {
		return ChatResponse{}, errors.New("model is required")
	}
	if len(request.Messages) == 0 {
		return ChatResponse{}, errors.New("messages are required")
	}
	body, err := json.Marshal(request)
	if err != nil {
		return ChatResponse{}, fmt.Errorf("encode chat request: %w", err)
	}
	response, err := c.do(ctx, http.MethodPost, "/chat/completions", bytes.NewReader(body))
	if err != nil {
		return ChatResponse{}, err
	}
	var decoded ChatResponse
	if err := json.Unmarshal(response, &decoded); err != nil {
		return ChatResponse{}, fmt.Errorf("decode chat response: %w", err)
	}
	if len(decoded.Choices) == 0 {
		return ChatResponse{}, errors.New("chat response has no choices")
	}
	return decoded, nil
}

func (c *Client) do(ctx context.Context, method, path string, body io.Reader) ([]byte, error) {
	request, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, body)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	request.Header.Set("Accept", "application/json")
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	response, err := c.httpClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("request %s %s: %w", method, path, err)
	}
	defer response.Body.Close()
	limited := io.LimitReader(response.Body, c.maxResponseBytes+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}
	if int64(len(data)) > c.maxResponseBytes {
		return nil, fmt.Errorf("response exceeds limit of %d bytes", c.maxResponseBytes)
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("provider returned HTTP %d", response.StatusCode)
	}
	return data, nil
}

func validateLoopbackBaseURL(raw string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return "", fmt.Errorf("parse provider URL: %w", err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", errors.New("provider URL must use http or https")
	}
	if parsed.User != nil || parsed.Hostname() == "" {
		return "", errors.New("provider URL must not contain credentials and must have a host")
	}
	if !isLoopback(parsed.Hostname()) {
		return "", errors.New("Bionic provider URL must resolve to loopback")
	}
	return strings.TrimRight(parsed.String(), "/"), nil
}

func isLoopback(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
