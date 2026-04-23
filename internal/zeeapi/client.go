package zeeapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"go.naturallyfunny.dev/api"
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

func (c *client) doRequest(ctx context.Context, method, path, userID string, body any) (io.ReadCloser, error) {
	var bodyReader io.Reader
	if body != nil {
		reqBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(reqBody)
	}

	url := c.baseURL + path
	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	if userID != "" {
		req.Header.Set("x-user-id", userID)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute request %s %s: %w", method, path, err)
	}

	return resp.Body, nil
}

func (c *client) GetAccount(ctx context.Context, userID string) (any, error) {
	body, err := c.doRequest(ctx, http.MethodGet, "/account", userID, nil)
	if err != nil {
		return nil, err
	}
	defer body.Close()
	return api.Decode[any](body)
}

func (c *client) ListDevices(ctx context.Context, userID string) (any, error) {
	body, err := c.doRequest(ctx, http.MethodGet, "/devices", userID, nil)
	if err != nil {
		return nil, err
	}
	defer body.Close()
	return api.Decode[any](body)
}

func (c *client) SendCommands(ctx context.Context, userID string, deviceID string, commands any) (any, error) {
	body, err := c.doRequest(ctx, http.MethodPost, fmt.Sprintf("/devices/%s/commands", deviceID), userID, map[string]any{"commands": commands})
	if err != nil {
		return nil, err
	}
	defer body.Close()
	return api.Decode[any](body)
}
