package models

import "github.com/avagenc/zee-api/internal/tuya"

type Device struct {
	DeviceID        string           `json:"id"`
	Category        string           `json:"category"`
	Name            string           `json:"name"`
	Status          []tuya.DataPoint `json:"status"`
	CodeNameMapping []tuya.Channel   `json:"code_name_mapping"`
}
