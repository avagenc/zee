package tuya

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

const (
	tokenExpiredTuyaErrorCode = 1010
	maxIoTRequestAttempts     = 2
)

type response struct {
	Success bool   `json:"success"`
	T       int64  `json:"t"`
	Tid     string `json:"tid"`

	Result json.RawMessage `json:"result,omitempty"`

	Code int    `json:"code,omitempty"`
	Msg  string `json:"msg,omitempty"`
}

type Client struct {
	accessID     string
	accessSecret string
	baseURL      string
	httpClient   *http.Client
	token        *Token
	tokenLock    sync.RWMutex
}

func NewClient(accessID, accessSecret, baseURL string) (*Client, error) {
	client := &Client{
		accessID:     accessID,
		accessSecret: accessSecret,
		baseURL:      baseURL,
		httpClient:   &http.Client{Timeout: 10 * time.Second},
		token:        &Token{},
		tokenLock:    sync.RWMutex{},
	}

	if err := client.ensureValidToken(); err != nil {
		return nil, fmt.Errorf("failed to set token during client initialization: %w", err)
	}

	return client, nil
}

func (c *Client) Do(method, path string, body []byte) (json.RawMessage, error) {
	for attempt := 0; attempt < maxIoTRequestAttempts; attempt++ {
		fullURL := c.baseURL + path

		var accessToken string
		c.tokenLock.RLock()
		if c.token != nil {
			accessToken = c.token.AccessToken
		}
		c.tokenLock.RUnlock()

		signature, err := generateSignature(c.accessID, c.accessSecret, accessToken, method, path, body)
		if err != nil {
			return nil, fmt.Errorf("failed to generate signature: %w", err)
		}

		bodyReader := bytes.NewReader(body)
		httpReq, err := http.NewRequest(method, fullURL, bodyReader)
		if err != nil {
			return nil, fmt.Errorf("failed to create request to %s: %w", fullURL, err)
		}

		if len(body) > 0 {
			httpReq.Header.Set("Content-Type", "application/json")
		}
		httpReq.Header.Set("client_id", c.accessID)
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

		var tuyaResp response
		if err := json.Unmarshal(respBodyBytes, &tuyaResp); err != nil {
			return nil, fmt.Errorf("failed to decode response from %s: %w", fullURL, err)
		}

		if tuyaResp.Success {
			return tuyaResp.Result, nil
		}

		if tuyaResp.Code == tokenExpiredTuyaErrorCode && attempt == 0 {
			if err := c.ensureValidToken(); err != nil {
				return nil, fmt.Errorf("failed to refresh token after Tuya error %d: %w", tuyaResp.Code, err)
			}
			continue
		}

		return nil, fmt.Errorf("tuya api error %d: %s", tuyaResp.Code, tuyaResp.Msg)
	}

	return nil, fmt.Errorf("failed to execute request to %s after retrying with a refreshed token", path)
}

func (c *Client) doTokenRequest(method, path string) (*response, error) {
	fullURL := c.baseURL + path

	signature, err := generateSignature(c.accessID, c.accessSecret, "", method, path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to generate token signature: %w", err)
	}

	httpReq, err := http.NewRequest(method, fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create token request to %s: %w", fullURL, err)
	}

	httpReq.Header.Set("client_id", c.accessID)
	httpReq.Header.Set("sign", signature.Sign)
	httpReq.Header.Set("t", signature.Timestamp)
	httpReq.Header.Set("sign_method", signature.SignMethod)
	httpReq.Header.Set("access_token", "")
	httpReq.Header.Set("nonce", signature.Nonce)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("token request to %s failed: %w", fullURL, err)
	}
	defer resp.Body.Close()

	respBodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read token response from %s: %w", fullURL, err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("token request to %s returned non-200 status code: %d, body: %s", fullURL, resp.StatusCode, string(respBodyBytes))
	}

	var tuyaResp response
	if err := json.Unmarshal(respBodyBytes, &tuyaResp); err != nil {
		return nil, fmt.Errorf("failed to decode token response from %s: %w", fullURL, err)
	}

	return &tuyaResp, nil
}

func (c *Client) updateToken() error {
	resp, err := c.getToken()
	if err != nil {
		return fmt.Errorf("failed to get token: %w", err)
	}

	if !resp.Success {
		return fmt.Errorf("Tuya token request failed with code %d: %s", resp.Code, resp.Msg)
	}

	var newToken Token
	if err := json.Unmarshal(resp.Result, &newToken); err != nil {
		return fmt.Errorf("failed to unmarshal token result: %w", err)
	}

	newToken.ExpireTime = time.Now().Unix() + newToken.ExpireTime

	c.token = &newToken
	return nil
}

func (c *Client) ensureValidToken() error {
	c.tokenLock.Lock()
	defer c.tokenLock.Unlock()

	if c.token != nil && c.token.ExpireTime > time.Now().Unix() {
		return nil
	}

	return c.updateToken()
}
