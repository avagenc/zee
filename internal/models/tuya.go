package models

import (
	 "encoding/json"
)

type TuyaDataPoint struct {
	Code  string `json:"code"`
	Value any    `json:"value"`
}

type TuyaChannel struct {
	Identifier string `json:"identifier"`
	Name       string `json:"name"`
}

type TuyaDeviceProperty struct {
	Code       string `json:"code"`
	CustomName string `json:"custom_name"`
	DPID       int    `json:"dp_id"`
	Time       int64  `json:"time"`
	Type       string `json:"type"`
	Value      any    `json:"value"`
}

type TuyaResponse struct {
	Success bool   `json:"success"`
	T       int64  `json:"t"`
	Tid     string `json:"tid"`

	// Exists when success == true
	Result json.RawMessage `json:"result,omitempty"`

	// Exists when success == false
	Code int    `json:"code,omitempty"`
	Msg  string `json:"msg,omitempty"`
}

type TuyaRequest struct {
	Method  string `json:"method"`
	URLPath string `json:"url_path"`
	Body    string `json:"body,omitempty"`
}

type TuyaSignature struct {
	Sign        string `json:"sign"`
	Timestamp   string `json:"t"`
	Nonce       string `json:"nonce"`
	SignMethod  string `json:"sign_method"`
	AccessToken string `json:"access_token"`
}

type TuyaToken struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpireTime   int64  `json:"expire_time"`
	UID          string `json:"uid"`
}
