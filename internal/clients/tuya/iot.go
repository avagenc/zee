package tuya

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/avagenc/agentic-tuya-smart/internal/models"
)

const (
	devicesEndpoint    = "/v1.0/devices"
	cloudThingEndpoint = "/v2.0/cloud/thing"
	homeEndpoint       = "/v1.0/homes"
)

func (c *Client) QueryProperties(deviceID string) (*models.TuyaResponse, error) {
	path := fmt.Sprintf("%s/%s/shadow/properties", cloudThingEndpoint, deviceID)
	tuyaReq := models.TuyaRequest{
		Method:  http.MethodGet,
		URLPath: path,
	}
	return c.doIoTRequest(tuyaReq)
}

func (c *Client) SendCommands(deviceID string, commands []models.TuyaDataPoint) (*models.TuyaResponse, error) {
	path := fmt.Sprintf("%s/%s/commands", devicesEndpoint, deviceID)
	bodyBytes, err := json.Marshal(struct {
		Commands []models.TuyaDataPoint `json:"commands"`
	}{
		Commands: commands,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal command payload: %w", err)
	}

	tuyaReq := models.TuyaRequest{
		Method:  http.MethodPost,
		URLPath: path,
		Body:    string(bodyBytes),
	}
	return c.doIoTRequest(tuyaReq)
}

func (c *Client) GetMultiChannelName(deviceID string) (*models.TuyaResponse, error) {
	path := fmt.Sprintf("%s/%s/multiple-names", devicesEndpoint, deviceID)
	tuyaReq := models.TuyaRequest{
		Method:  http.MethodGet,
		URLPath: path,
	}
	return c.doIoTRequest(tuyaReq)
}

func (c *Client) QueryDevicesInHome(homeID string) (*models.TuyaResponse, error) {
	path := fmt.Sprintf("%s/%s/devices", homeEndpoint, homeID)
	tuyaReq := models.TuyaRequest{
		Method:  http.MethodGet,
		URLPath: path,
	}
	return c.doIoTRequest(tuyaReq)
}
