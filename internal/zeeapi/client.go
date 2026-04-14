package zeeapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"go.ibnfadl.com/api"
)

type Client interface {
	GetAccount(ctx context.Context, userID string) (any, error)
	ListDevices(ctx context.Context, userID string) (any, error)
	SendCommands(ctx context.Context, userID string, deviceID string, commands any) (any, error)
}

type client struct {
	baseURL    string
	httpClient *http.Client
}

func NewClient(baseURL string, httpClient *http.Client) Client {
	return &client{
		baseURL:    baseURL,
		httpClient: httpClient,
	}
}

func (c *client) doRequest(ctx context.Context, method, path, userID string, body any, target any) error {
	var bodyReader io.Reader
	if body != nil {
		reqBody, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(reqBody)
	}

	url := c.baseURL + path
	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	if userID != "" {
		req.Header.Set("x-user-id", userID)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("execute request %s %s: %w", method, path, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("zee api error (status %d): %s", resp.StatusCode, string(respBody))
	}

	if target != nil {
		if err := json.Unmarshal(respBody, target); err != nil {
			return fmt.Errorf("decode response body: %w", err)
		}
	}

	return nil
}

func (c *client) GetAccount(ctx context.Context, userID string) (any, error) {
	var res api.Response[any]
	if err := c.doRequest(ctx, http.MethodGet, "/account", userID, nil, &res); err != nil {
		return nil, err
	}
	if !res.Success {
		return nil, fmt.Errorf("api error: [%s] %s", res.Code, res.Message)
	}
	return res.Data, nil
}

func (c *client) ListDevices(ctx context.Context, userID string) (any, error) {
	var res api.Response[any]
	if err := c.doRequest(ctx, http.MethodGet, "/devices", userID, nil, &res); err != nil {
		return nil, err
	}
	if !res.Success {
		return nil, fmt.Errorf("api error: [%s] %s", res.Code, res.Message)
	}
	return res.Data, nil
}

func (c *client) SendCommands(ctx context.Context, userID string, deviceID string, commands any) (any, error) {
	reqBody := map[string]any{"commands": commands}
	var res api.Response[any]
	path := fmt.Sprintf("/devices/%s/commands", deviceID)
	if err := c.doRequest(ctx, http.MethodPost, path, userID, reqBody, &res); err != nil {
		return nil, err
	}
	if !res.Success {
		return nil, fmt.Errorf("api error: [%s] %s", res.Code, res.Message)
	}
	return res.Data, nil
}
