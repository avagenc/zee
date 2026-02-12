package domain

import "errors"

var ErrDeviceNotOwned = errors.New("device does not belong to user")

type DataPoint struct {
	Code  string `json:"code"`
	Value any    `json:"value"`
}

type Channel struct {
	Identifier string `json:"identifier"`
	Name       string `json:"name"`
}

type Device struct {
	ID              string      `json:"id"`
	Category        string      `json:"category"`
	Name            string      `json:"name"`
	Status          []DataPoint `json:"status"`
	CodeNameMapping []Channel   `json:"code_name_mapping"`
}
