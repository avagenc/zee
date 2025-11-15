package tuya

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/avagenc/agentic-tuya-smart/internal/models"
)

type Client struct {
	AccessID     string
	AccessSecret string
	BaseURL      string
	httpClient   *http.Client
	Token        *models.TuyaToken
	tokenLock    sync.RWMutex
}

func NewClient(accessID, accessSecret, baseURL string) (*Client, error) {
	client := &Client{
		AccessID:     accessID,
		AccessSecret: accessSecret,
		BaseURL:      baseURL,
		httpClient:   &http.Client{Timeout: 10 * time.Second},
		Token:        &models.TuyaToken{},
		tokenLock:    sync.RWMutex{},
	}

	if err := client.refreshToken(); err != nil {
		return nil, fmt.Errorf("failed to refresh token during client initialization: %w", err)
	}

	return client, nil
}

func (c *Client) doRequest(tuyaReq models.TuyaRequest) (*models.TuyaResponse, error) {
	fullURL := c.BaseURL + tuyaReq.URLPath

	bodyBytes := []byte(tuyaReq.Body)

	signature, err := c.generateSignature(c.AccessID, c.AccessSecret, c.Token.AccessToken, tuyaReq)
	if err != nil {
		return nil, fmt.Errorf("failed to generate signature: %w", err)
	}

	bodyReader := bytes.NewReader(bodyBytes)
	httpReq, err := http.NewRequest(tuyaReq.Method, fullURL, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request to %s: %w", fullURL, err)
	}

	if len(bodyBytes) > 0 {
		httpReq.Header.Set("Content-Type", "application/json")
	}
	httpReq.Header.Set("client_id", c.AccessID)
	httpReq.Header.Set("sign", signature.Sign)
	httpReq.Header.Set("t", signature.Timestamp)
	httpReq.Header.Set("sign_method", signature.SignMethod)
	httpReq.Header.Set("access_token", signature.AccessToken)
	httpReq.Header.Set("nonce", signature.Nonce)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request to %s failed: %w", fullURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("request to %s returned non-200 status code: %d, body: %s", fullURL, resp.StatusCode, string(respBody))
	}

	respBodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response from %s: %w", fullURL, err)
	}

	var tuyaResponse models.TuyaResponse
	if err := json.Unmarshal(respBodyBytes, &tuyaResponse); err != nil {
		return nil, fmt.Errorf("failed to decode response from %s: %w", fullURL, err)
	}

	if !tuyaResponse.Success {
		return nil, fmt.Errorf("tuya api error %d: %s", tuyaResponse.Code, tuyaResponse.Msg)
	}

	return &tuyaResponse, nil
}

func (c *Client) refreshToken() error {
	c.tokenLock.Lock()
	defer c.tokenLock.Unlock()

	resp, err := c.getToken()
	if err != nil {
		return fmt.Errorf("failed to get token: %w", err)
	}

	var token models.TuyaToken
	if err := json.Unmarshal(resp.Result, &token); err != nil {
		return fmt.Errorf("failed to unmarshal token: %w", err)
	}
	c.Token = &token

	return nil
}
