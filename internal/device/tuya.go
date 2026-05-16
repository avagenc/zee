package device

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type TuyaClient interface {
	Do(method, path string, body []byte) (json.RawMessage, error)
}

type tuyaIoTClient struct {
	client TuyaClient
}

func NewTuyaIoTClient(client TuyaClient) TuyaIoTClient {
	return &tuyaIoTClient{client: client}
}

func (c *tuyaIoTClient) List(tuyaUID string) ([]Device, error) {
	path := fmt.Sprintf("/v1.0/users/%s/devices", tuyaUID)
	result, err := c.client.Do(http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var devices []Device
	if err := json.Unmarshal(result, &devices); err != nil {
		return nil, fmt.Errorf("failed to unmarshal device list: %w", err)
	}

	return devices, nil
}

func (c *tuyaIoTClient) SendCommands(deviceID string, commands any) (json.RawMessage, error) {
	path := fmt.Sprintf("/v1.0/iot-03/devices/%s/commands", deviceID)
	bodyBytes, err := json.Marshal(struct {
		Commands any `json:"commands"`
	}{
		Commands: commands,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal command payload: %w", err)
	}

	return c.client.Do(http.MethodPost, path, bodyBytes)
}

func (c *tuyaIoTClient) GetMultiChannelName(deviceID string) (json.RawMessage, error) {
	path := fmt.Sprintf("/v1.0/devices/%s/multiple-names", deviceID)
	return c.client.Do(http.MethodGet, path, nil)
}
