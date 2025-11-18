package tuya

import (
	"fmt"
	"net/http"

	"github.com/avagenc/agentic-tuya-smart/internal/models"
)

const tokenEndpoint = "/v1.0/token"

func (c *Client) getToken() (*models.TuyaResponse, error) {
	path := fmt.Sprintf("%s?grant_type=1", tokenEndpoint)
	tuyaReq := models.TuyaRequest{
		Method:  http.MethodGet,
		URLPath: path,
	}
	return c.doTokenRequest(tuyaReq)
}
