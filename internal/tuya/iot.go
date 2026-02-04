package tuya

import (
	"encoding/json"
	"fmt"
	"net/http"

	
)

const (
	devicesEndpoint    = "/v1.0/devices"
	cloudThingEndpoint = "/v2.0/cloud/thing"
	homeEndpoint       = "/v1.0/homes"
)

func (c *Client) QueryProperties(deviceID string) (*Response, error) {
	path := fmt.Sprintf("%s/%s/shadow/properties", cloudThingEndpoint, deviceID)
	tuyaReq := Request{
		Method:  http.MethodGet,
		URLPath: path,
	}
	return c.doIoTRequest(tuyaReq)
}

func (c *Client) SendCommands(deviceID string, commands []DataPoint) (*Response, error) {
	path := fmt.Sprintf("%s/%s/commands", devicesEndpoint, deviceID)
	bodyBytes, err := json.Marshal(struct {
		Commands []DataPoint `json:"commands"`
	}{
		Commands: commands,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal command payload: %w", err)
	}

	tuyaReq := Request{
		Method:  http.MethodPost,
		URLPath: path,
		Body:    string(bodyBytes),
	}
	return c.doIoTRequest(tuyaReq)
}

func (c *Client) GetMultiChannelName(deviceID string) (*Response, error) {
	path := fmt.Sprintf("%s/%s/multiple-names", devicesEndpoint, deviceID)
	tuyaReq := Request{
		Method:  http.MethodGet,
		URLPath: path,
	}
	return c.doIoTRequest(tuyaReq)
}

func (c *Client) QueryDevicesInHome(homeID string) (*Response, error) {
	path := fmt.Sprintf("%s/%s/devices", homeEndpoint, homeID)
	tuyaReq := Request{
		Method:  http.MethodGet,
		URLPath: path,
	}
	return c.doIoTRequest(tuyaReq)
}
