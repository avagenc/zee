package models

type Device struct {
	DeviceID        string          `json:"id"`
	Category        string          `json:"category"`
	Name            string          `json:"name"`
	Status          []TuyaDataPoint `json:"status"`
	CodeNameMapping []TuyaChannel   `json:"code_name_mapping"`
}
