package services

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/avagenc/agentic-tuya-smart/internal/models"
	"github.com/avagenc/agentic-tuya-smart/internal/clients/tuya"
)

type DeviceService struct {
	tuyaClient *tuya.Client
}

func NewDeviceService(tuyaClient *tuya.Client) *DeviceService {
	return &DeviceService{tuyaClient: tuyaClient}
}

func (s *DeviceService) SendCommands(deviceID string, commands []models.TuyaDataPoint) (json.RawMessage, error) {
	resp, err := s.tuyaClient.SendCommands(deviceID, commands)
	if err != nil {
		return nil, fmt.Errorf("tuya client failed to send commands: %w", err)
	}

	return resp.Result, nil
}

func (s *DeviceService) GetAllByHomeId(homeID string) ([]models.Device, error) {
	resp, err := s.tuyaClient.QueryDevicesInHome(homeID)
	if err != nil {
		return nil, err
	}

	var devices []models.Device
	if len(resp.Result) > 0 {
		if err := json.Unmarshal(resp.Result, &devices); err != nil {
			return nil, fmt.Errorf("failed to decode home devices response: %w", err)
		}
	}

	if len(devices) == 0 {
		return []models.Device{}, nil
	}

	var devicesToEnrich []*models.Device
	for i := range devices {
		device := &devices[i]
		category := strings.ToLower(device.Category)

		device.CodeNameMapping = []models.TuyaChannel{}

		if (category == "kg" || strings.HasPrefix(category, "cz")) && device.DeviceID != "" {
			devicesToEnrich = append(devicesToEnrich, device)
		}
	}

	if len(devicesToEnrich) > 0 {
		if err := s.enrichWithChannelNames(devicesToEnrich); err != nil {
			fmt.Printf("Warning: %v\n", err)
		}
	}

	return devices, nil
}

func (s *DeviceService) enrichWithChannelNames(devicesToEnrich []*models.Device) error {
	var wg sync.WaitGroup
	errs := make(chan error, len(devicesToEnrich))

	for _, device := range devicesToEnrich {
		wg.Add(1)
		go func(device *models.Device) {
			defer wg.Done()

			resp, err := s.tuyaClient.GetMultiChannelName(device.DeviceID)
			if err != nil {
				errs <- fmt.Errorf("failed to get multi-channel name for device %s: %w", device.DeviceID, err)
				return
			}

			var multiChannelNames []models.TuyaChannel
			if len(resp.Result) > 0 {
				if err := json.Unmarshal(resp.Result, &multiChannelNames); err != nil {
					errs <- fmt.Errorf("failed to decode channel names for device %s: %w", device.DeviceID, err)
					return
				}
			}
			device.CodeNameMapping = multiChannelNames
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
		return fmt.Errorf("encountered %d error(s) while fetching channel names: %s", len(allErrors), strings.Join(allErrors, "; "))
	}

	return nil
}
