package models

type DeviceCommandsRequest struct {
	DeviceID string          `json:"device_id"`
	Commands []TuyaDataPoint `json:"commands"`
}
