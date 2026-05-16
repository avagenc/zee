package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/avagenc/zee-agent/internal/account"
	"github.com/avagenc/zee-agent/internal/device"
	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/functiontool"
)

type AccountReader interface {
	Get(ctx context.Context, ownerID string) (account.Account, error)
}

type DeviceReader interface {
	List(ctx context.Context, userID string) ([]device.Device, error)
}

type CommandSender interface {
	SendCommands(ctx context.Context, userID, deviceID string, commands []device.DataPoint) (json.RawMessage, error)
}

type Services struct {
	Account AccountReader
	Device  DeviceReader
	Sender  CommandSender
}

type Tool struct {
	GetAccount            tool.Tool
	ListDevices           tool.Tool
	SendCommandsToADevice tool.Tool
}

func Load(svc Services) (*Tool, error) {
	getAccount, err := functiontool.New(
		functiontool.Config{
			Name:        "get_account",
			Description: "Use this tool to retrieve the Tuya account linked to the current authenticated avagenc user.",
		},
		func(ctx tool.Context, args struct{}) (any, error) {
			result, err := svc.Account.Get(ctx, ctx.UserID())
			if err != nil {
				fmt.Printf("[tool:get_account] error: %v\n", err)
				return nil, err
			}
			return result, nil
		},
	)
	if err != nil {
		return nil, err
	}

	listDevices, err := functiontool.New(
		functiontool.Config{
			Name:        "list_devices",
			Description: "Use this tool to retrieve all Tuya IoT devices linked to the linked user's Tuya account.",
		},
		func(ctx tool.Context, args struct{}) (map[string]any, error) {
			result, err := svc.Device.List(ctx, ctx.UserID())
			if err != nil {
				fmt.Printf("[tool:list_devices] error: %v\n", err)
				return nil, err
			}
			return map[string]any{"devices": result}, nil
		},
	)
	if err != nil {
		return nil, err
	}

	sendCommandsToADevice, err := functiontool.New(
		functiontool.Config{
			Name:        "send_commands_to_a_device",
			Description: `Use this tool to send commands to a device by the device "id". You can get "id" of devices from list_devices tool. Always pay attention to the "id" of the device, make sure you input the accurate device "id" or else it would be fatal. in string form but array of maps`,
		},
		func(ctx tool.Context, args struct {
			DeviceID string             `json:"device_id"`
			Commands []device.DataPoint `json:"commands"`
		}) (any, error) {
			result, err := svc.Sender.SendCommands(ctx, ctx.UserID(), args.DeviceID, args.Commands)
			if err != nil {
				fmt.Printf("[tool:send_commands_to_a_device] error: %v\n", err)
				return nil, err
			}
			return result, nil
		},
	)
	if err != nil {
		return nil, err
	}

	return &Tool{
		GetAccount:            getAccount,
		ListDevices:           listDevices,
		SendCommandsToADevice: sendCommandsToADevice,
	}, nil
}
