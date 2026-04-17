package tools

import (
	"context"
	"fmt"

	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/functiontool"
)

type ZeeAPIClient interface {
	GetAccount(ctx context.Context, userID string) (any, error)
	ListDevices(ctx context.Context, userID string) (any, error)
	SendCommands(ctx context.Context, userID string, deviceID string, commands any) (any, error)
}

type Tool struct {
	GetAccount            tool.Tool
	ListDevices           tool.Tool
	SendCommandsToADevice tool.Tool
}

func Load(client ZeeAPIClient) (*Tool, error) {
	getAccount, err := functiontool.New(
		functiontool.Config{
			Name:        "get_account",
			Description: "Use this tool to retrieve the Tuya account linked to the current authenticated avagenc user.",
		},
		func(ctx tool.Context, args struct{}) (any, error) {
			userID := ctx.UserID()
			result, err := client.GetAccount(ctx, userID)
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
			userID := ctx.UserID()
			result, err := client.ListDevices(ctx, userID)
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
			DeviceID string `json:"device_id"`
			Commands any    `json:"commands"`
		}) (any, error) {
			userID := ctx.UserID()
			result, err := client.SendCommands(ctx, userID, args.DeviceID, args.Commands)
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
