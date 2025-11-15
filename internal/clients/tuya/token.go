package tuya

import (
	"github.com/avagenc/agentic-tuya-smart/internal/models"
	"net/http"
)

const tuyaTokenEndpoint = "/v1.0/token?grant_type=1"

func (c *Client) getToken() (*models.TuyaResponse, error) {
	tuyaReq := models.TuyaRequest{
		Method:  http.MethodGet,
		URLPath: tuyaTokenEndpoint,
	}
	return c.doRequest(tuyaReq)
}
