package tuya

import (
	"encoding/json"
	"fmt"
	"net/http"
)

func (c *Client) List(tuyaUID string) (json.RawMessage, error) {
	path := fmt.Sprintf("/v1.0/users/%s/devices", tuyaUID)
	return c.Do(http.MethodGet, path, nil)
}

func (c *Client) SendCommands(deviceID string, commands any) (json.RawMessage, error) {
	path := fmt.Sprintf("/v1.0/iot-03/devices/%s/commands", deviceID)
	body, err := json.Marshal(struct {
		Commands any `json:"commands"`
	}{Commands: commands})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal command payload: %w", err)
	}
	return c.Do(http.MethodPost, path, body)
}

func (c *Client) GetMultiChannelName(deviceID string) (json.RawMessage, error) {
	path := fmt.Sprintf("/v1.0/devices/%s/multiple-names", deviceID)
	return c.Do(http.MethodGet, path, nil)
}
