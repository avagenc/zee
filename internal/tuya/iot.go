package tuya

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/avagenc/zee-api/internal/domain"
)

const (
	devicesEndpoint    = "/v1.0/iot-03/devices"
	cloudThingEndpoint = "/v2.0/cloud/thing"
	homeEndpoint       = "/v1.0/homes"
	userEndpoint       = "/v1.0/users"
)

// Deprecated: Unused. Kept for future reference.
func (c *Client) QueryProperties(deviceID string) (json.RawMessage, error) {
	path := fmt.Sprintf("%s/%s/shadow/properties", cloudThingEndpoint, deviceID)
	tuyaReq := Request{
		Method:  http.MethodGet,
		URLPath: path,
	}
	resp, err := c.doIoTRequest(tuyaReq)
	if err != nil {
		return nil, err
	}
	return resp.Result, nil
}

func (c *Client) SendCommands(deviceID string, commands any) (json.RawMessage, error) {
	path := fmt.Sprintf("%s/%s/commands", devicesEndpoint, deviceID)
	bodyBytes, err := json.Marshal(struct {
		Commands any `json:"commands"`
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
	resp, err := c.doIoTRequest(tuyaReq)
	if err != nil {
		return nil, err
	}
	return resp.Result, nil
}

func (c *Client) GetMultiChannelName(deviceID string) (json.RawMessage, error) {
	path := fmt.Sprintf("%s/%s/multiple-names", devicesEndpoint, deviceID)
	tuyaReq := Request{
		Method:  http.MethodGet,
		URLPath: path,
	}
	resp, err := c.doIoTRequest(tuyaReq)
	if err != nil {
		return nil, err
	}
	return resp.Result, nil
}

// Deprecated: Unused. Kept for future reference.
func (c *Client) QueryDevicesInHome(homeID string) (json.RawMessage, error) {
	path := fmt.Sprintf("%s/%s/devices", homeEndpoint, homeID)
	tuyaReq := Request{
		Method:  http.MethodGet,
		URLPath: path,
	}
	resp, err := c.doIoTRequest(tuyaReq)
	if err != nil {
		return nil, err
	}
	return resp.Result, nil
}

func (c *Client) GetUserDeviceList(tuyaUID string) ([]domain.Device, error) {
	path := fmt.Sprintf("%s/%s/devices", userEndpoint, tuyaUID)
	tuyaReq := Request{
		Method:  http.MethodGet,
		URLPath: path,
	}
	resp, err := c.doIoTRequest(tuyaReq)
	if err != nil {
		return nil, err
	}

	var devices []domain.Device
	if err := json.Unmarshal(resp.Result, &devices); err != nil {
		return nil, fmt.Errorf("failed to unmarshal device list: %w", err)
	}

	return devices, nil
}

func (c *Client) GetUserDeviceIDs(tuyaUID string) ([]string, error) {
	devices, err := c.GetUserDeviceList(tuyaUID)
	if err != nil {
		return nil, err
	}

	ids := make([]string, len(devices))
	for i, d := range devices {
		ids[i] = d.ID
	}

	return ids, nil
}
