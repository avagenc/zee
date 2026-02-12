package device

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/avagenc/zee/internal/domain"
)

type TuyaUIDGetter func(ctx context.Context, userID string) (string, error)

type TuyaIoTClient interface {
	SendCommands(deviceID string, commands any) (json.RawMessage, error)
	GetMultiChannelName(deviceID string) (json.RawMessage, error)
	List(tuyaUID string) ([]domain.Device, error)
}

type service struct {
	getTuyaID TuyaUIDGetter
	tuya      TuyaIoTClient
}

func NewService(getTuyaID TuyaUIDGetter, tuya TuyaIoTClient) *service {
	return &service{getTuyaID: getTuyaID, tuya: tuya}
}

func (s *service) List(ctx context.Context, userID string) ([]domain.Device, error) {
	tuyaUID, err := s.getTuyaID(ctx, userID)
	if err != nil {
		return nil, err
	}

	devices, err := s.tuya.List(tuyaUID)
	if err != nil {
		return nil, err
	}

	if len(devices) == 0 {
		return []domain.Device{}, nil
	}

	if err := s.enrichDevices(devices); err != nil {
		fmt.Printf("Warning: %v\n", err)
	}

	return devices, nil
}

func (s *service) SendCommands(ctx context.Context, userID string, deviceID string, commands []domain.DataPoint) (json.RawMessage, error) {
	tuyaUID, err := s.getTuyaID(ctx, userID)
	if err != nil {
		return nil, err
	}

	deviceIDs, err := s.getUserDeviceIDs(tuyaUID)
	if err != nil {
		return nil, fmt.Errorf("failed to verify device ownership: %w", err)
	}

	if !contains(deviceIDs, deviceID) {
		return nil, domain.ErrDeviceNotOwned
	}

	result, err := s.tuya.SendCommands(deviceID, commands)
	if err != nil {
		return nil, fmt.Errorf("failed to send commands: %w", err)
	}
	return result, nil
}

func (s *service) getUserDeviceIDs(tuyaUID string) ([]string, error) {
	devices, err := s.tuya.List(tuyaUID)
	if err != nil {
		return nil, err
	}
	ids := make([]string, len(devices))
	for i, d := range devices {
		ids[i] = d.ID
	}
	return ids, nil
}

func (s *service) enrichDevices(devices []domain.Device) error {
	var devicesToEnrich []*domain.Device
	for i := range devices {
		device := &devices[i]
		category := strings.ToLower(device.Category)
		device.CodeNameMapping = []domain.Channel{}

		if (category == "kg" || strings.HasPrefix(category, "cz")) && device.ID != "" {
			devicesToEnrich = append(devicesToEnrich, device)
		}
	}

	if len(devicesToEnrich) > 0 {
		return s.enrichWithChannelNames(devicesToEnrich)
	}
	return nil
}

func (s *service) enrichWithChannelNames(devices []*domain.Device) error {
	var wg sync.WaitGroup
	errs := make(chan error, len(devices))

	for _, device := range devices {
		wg.Add(1)
		go func(device *domain.Device) {
			defer wg.Done()

			result, err := s.tuya.GetMultiChannelName(device.ID)
			if err != nil {
				errs <- fmt.Errorf("failed to get channel name for device %s: %w", device.ID, err)
				return
			}

			var channels []domain.Channel
			if len(result) > 0 {
				if err := json.Unmarshal(result, &channels); err != nil {
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

func contains(slice []string, item string) bool {
	for _, v := range slice {
		if v == item {
			return true
		}
	}
	return false
}
