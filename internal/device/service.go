package device

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"sync"
)

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

type TuyaUIDGetter func(ctx context.Context, userID string) (string, error)

type TuyaClient interface {
	List(tuyaUID string) (json.RawMessage, error)
	SendCommands(deviceID string, commands any) (json.RawMessage, error)
	GetMultiChannelName(deviceID string) (json.RawMessage, error)
}

type service struct {
	getTuyaID TuyaUIDGetter
	tuya      TuyaClient
}

func NewService(getTuyaID TuyaUIDGetter, tuya TuyaClient) *service {
	return &service{getTuyaID: getTuyaID, tuya: tuya}
}

func (s *service) List(ctx context.Context, userID string) ([]Device, error) {
	tuyaUID, err := s.getTuyaID(ctx, userID)
	if err != nil {
		return nil, err
	}

	raw, err := s.tuya.List(tuyaUID)
	if err != nil {
		return nil, err
	}

	var devices []Device
	if err := json.Unmarshal(raw, &devices); err != nil {
		return nil, fmt.Errorf("failed to unmarshal device list: %w", err)
	}

	if len(devices) == 0 {
		return []Device{}, nil
	}

	if err := s.enrichDevices(devices); err != nil {
		return nil, fmt.Errorf("failed to enrich devices: %w", err)
	}

	return devices, nil
}

var ErrDeviceNotOwned = fmt.Errorf("device does not belong to user")

func (s *service) SendCommands(ctx context.Context, userID string, deviceID string, commands []DataPoint) (json.RawMessage, error) {
	tuyaUID, err := s.getTuyaID(ctx, userID)
	if err != nil {
		return nil, err
	}

	deviceIDs, err := s.getUserDeviceIDs(tuyaUID)
	if err != nil {
		return nil, fmt.Errorf("failed to verify device ownership: %w", err)
	}

	if !slices.Contains(deviceIDs, deviceID) {
		return nil, ErrDeviceNotOwned
	}

	result, err := s.tuya.SendCommands(deviceID, commands)
	if err != nil {
		return nil, fmt.Errorf("failed to send commands: %w", err)
	}
	return result, nil
}

func (s *service) getUserDeviceIDs(tuyaUID string) ([]string, error) {
	raw, err := s.tuya.List(tuyaUID)
	if err != nil {
		return nil, err
	}
	var devices []Device
	if err := json.Unmarshal(raw, &devices); err != nil {
		return nil, fmt.Errorf("failed to unmarshal device list: %w", err)
	}
	ids := make([]string, len(devices))
	for i, d := range devices {
		ids[i] = d.ID
	}
	return ids, nil
}

func (s *service) enrichDevices(devices []Device) error {
	var devicesToEnrich []*Device
	for i := range devices {
		device := &devices[i]
		category := strings.ToLower(device.Category)
		device.CodeNameMapping = []Channel{}

		if (category == "kg" || strings.HasPrefix(category, "cz")) && device.ID != "" {
			devicesToEnrich = append(devicesToEnrich, device)
		}
	}

	if len(devicesToEnrich) > 0 {
		return s.enrichWithChannelNames(devicesToEnrich)
	}
	return nil
}

func (s *service) enrichWithChannelNames(devices []*Device) error {
	var wg sync.WaitGroup
	errs := make(chan error, len(devices))

	for _, device := range devices {
		wg.Add(1)
		go func(device *Device) {
			defer wg.Done()

			raw, err := s.tuya.GetMultiChannelName(device.ID)
			if err != nil {
				errs <- fmt.Errorf("failed to get channel name for device %s: %w", device.ID, err)
				return
			}

			var channels []Channel
			if len(raw) > 0 {
				if err := json.Unmarshal(raw, &channels); err != nil {
					errs <- fmt.Errorf("failed to decode channels for device %s: %w", device.ID, err)
					return
				}
			}
			device.CodeNameMapping = channels
		}(device)
	}

	wg.Wait()
	close(errs)

	var allErrors []string
	for err := range errs {
		if err != nil {
			allErrors = append(allErrors, err.Error())
		}
	}

	if len(allErrors) > 0 {
		return fmt.Errorf("encountered %d error(s): %s", len(allErrors), strings.Join(allErrors, "; "))
	}

	return nil
}
