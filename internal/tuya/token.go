package tuya

import (
	"fmt"
	"net/http"

	
)

const tokenEndpoint = "/v1.0/token"

func (c *Client) getToken() (*Response, error) {
	path := fmt.Sprintf("%s?grant_type=1", tokenEndpoint)
	tuyaReq := Request{
		Method:  http.MethodGet,
		URLPath: path,
	}
	return c.doTokenRequest(tuyaReq)
}
