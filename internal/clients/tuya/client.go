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

const (
	tokenExpiredTuyaErrorCode = 1010
	maxIoTRequestAttempts     = 2
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
	newClient := &Client{
		AccessID:     accessID,
		AccessSecret: accessSecret,
		BaseURL:      baseURL,
		httpClient:   &http.Client{Timeout: 10 * time.Second},
		Token:        &models.TuyaToken{},
		tokenLock:    sync.RWMutex{},
	}

	if err := newClient.ensureValidToken(); err != nil {
		return nil, fmt.Errorf("failed to set token during client initialization: %w", err)
	}

	return newClient, nil
}

func (c *Client) doIoTRequest(req models.TuyaRequest) (*models.TuyaResponse, error) {
	for attempt := 0; attempt < maxIoTRequestAttempts; attempt++ {
		fullURL := c.BaseURL + req.URLPath

		var accessToken string
		c.tokenLock.RLock()
		if c.Token != nil {
			accessToken = c.Token.AccessToken
		}
		c.tokenLock.RUnlock()

		signature, err := generateSignature(c.AccessID, c.AccessSecret, accessToken, req)
		if err != nil {
			return nil, fmt.Errorf("failed to generate signature: %w", err)
		}

		bodyBytes := []byte(req.Body)
		bodyReader := bytes.NewReader(bodyBytes)
		httpReq, err := http.NewRequest(req.Method, fullURL, bodyReader)
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
		httpReq.Header.Set("access_token", accessToken)
		httpReq.Header.Set("nonce", signature.Nonce)

		resp, err := c.httpClient.Do(httpReq)
		if err != nil {
			return nil, fmt.Errorf("request to %s failed: %w", fullURL, err)
		}
		defer resp.Body.Close()

		respBodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read response from %s: %w", fullURL, err)
		}

		if resp.StatusCode >= 400 {
			return nil, fmt.Errorf("request to %s returned non-200 status code: %d, body: %s", fullURL, resp.StatusCode, string(respBodyBytes))
		}

		var tuyaResponse models.TuyaResponse
		if err := json.Unmarshal(respBodyBytes, &tuyaResponse); err != nil {
			return nil, fmt.Errorf("failed to decode response from %s: %w", fullURL, err)
		}

		if tuyaResponse.Success {
			return &tuyaResponse, nil
		}

		if tuyaResponse.Code == tokenExpiredTuyaErrorCode && attempt == 0 {
			if err := c.ensureValidToken(); err != nil {
				return nil, fmt.Errorf("failed to refresh token after Tuya error %d: %w", tuyaResponse.Code, err)
			}
			continue
		}

		return nil, fmt.Errorf("tuya api error %d: %s", tuyaResponse.Code, tuyaResponse.Msg)
	}

	return nil, fmt.Errorf("failed to execute request to %s after retrying with a refreshed token", req.URLPath)
}

func (c *Client) doTokenRequest(req models.TuyaRequest) (*models.TuyaResponse, error) {
	fullURL := c.BaseURL + req.URLPath

	signature, err := generateSignature(c.AccessID, c.AccessSecret, "", req)
	if err != nil {
		return nil, fmt.Errorf("failed to generate token signature: %w", err)
	}

	httpReq, err := http.NewRequest(req.Method, fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create token request to %s: %w", fullURL, err)
	}

	httpReq.Header.Set("client_id", c.AccessID)
	httpReq.Header.Set("sign", signature.Sign)
	httpReq.Header.Set("t", signature.Timestamp)
	httpReq.Header.Set("sign_method", signature.SignMethod)
	httpReq.Header.Set("access_token", "")
	httpReq.Header.Set("nonce", signature.Nonce)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("token request to %s failed: %w", err)
	}
	defer resp.Body.Close()

	respBodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read token response from %s: %w", fullURL, err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("token request to %s returned non-200 status code: %d, body: %s", fullURL, resp.StatusCode, string(respBodyBytes))
	}

	var tuyaResponse models.TuyaResponse
	if err := json.Unmarshal(respBodyBytes, &tuyaResponse); err != nil {
		return nil, fmt.Errorf("failed to decode token response from %s: %w", fullURL, err)
	}

	return &tuyaResponse, nil
}

func (c *Client) updateToken() error {
	resp, err := c.getToken()
	if err != nil {
		return fmt.Errorf("failed to get token: %w", err)
	}

	if !resp.Success {
		return fmt.Errorf("Tuya token request failed with code %d: %s", resp.Code, resp.Msg)
	}

	var newToken models.TuyaToken
	if err := json.Unmarshal(resp.Result, &newToken); err != nil {
		return fmt.Errorf("failed to unmarshal token result: %w", err)
	}

	newToken.ExpireTime = time.Now().Unix() + newToken.ExpireTime

	c.Token = &newToken
	return nil
}

func (c *Client) ensureValidToken() error {
	c.tokenLock.Lock()
	defer c.tokenLock.Unlock()

	if c.Token != nil && c.Token.ExpireTime > time.Now().Unix() {
		return nil
	}

	return c.updateToken()
}
