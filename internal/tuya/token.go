package tuya

import (
	"fmt"
	"net/http"
)

const tokenEndpoint = "/v1.0/token"

type Token struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpireTime   int64  `json:"expire_time"`
	UID          string `json:"uid"`
}

func (c *Client) getToken() (*response, error) {
	path := fmt.Sprintf("%s?grant_type=1", tokenEndpoint)
	return c.doTokenRequest(http.MethodGet, path)
}
