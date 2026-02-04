package models

import "github.com/avagenc/zee-api/internal/tuya"

type DeviceCommandsRequest struct {
	DeviceID string           `json:"device_id"`
	Commands []tuya.DataPoint `json:"commands"`
}
