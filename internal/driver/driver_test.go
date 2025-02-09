// -*- Mode: Go; indent-tabs-mode: t -*-
//
// Copyright (C) 2021 Schneider Electric
//
// SPDX-License-Identifier: Apache-2.0

package driver

import (
	"fmt"
	"net/http"
	"reflect"
	"testing"

	"github.com/edgexfoundry/device-opcua-go/internal/server"
	"github.com/edgexfoundry/device-opcua-go/internal/test"
	"github.com/edgexfoundry/device-sdk-go/v4/pkg/interfaces/mocks"
	sdkModel "github.com/edgexfoundry/device-sdk-go/v4/pkg/models"
	"github.com/edgexfoundry/go-mod-core-contracts/v4/models"
	"github.com/stretchr/testify/mock"
)

func newMockDriver(t *testing.T) (*Driver, *mocks.DeviceServiceSDK) {
	d := NewProtocolDriver().(*Driver)
	dsMock := test.NewDSMock(t)

	d.sdk = dsMock
	d.serverMap = make(map[string]*server.Server)
	return d, dsMock
}

func TestDriver_UpdateDevice(t *testing.T) {
	type args struct {
		deviceName string
		protocols  map[string]models.ProtocolProperties
		adminState models.AdminState
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name:    "NOK - device not found",
			args:    args{deviceName: "Test"},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d, _ := newMockDriver(t)
			if err := d.UpdateDevice(tt.args.deviceName, tt.args.protocols, tt.args.adminState); (err != nil) != tt.wantErr {
				t.Errorf("Driver.UpdateDevice() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDriver_RemoveDevice(t *testing.T) {
	type args struct {
		deviceName string
		protocols  map[string]models.ProtocolProperties
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name:    "NOK - device not found",
			args:    args{deviceName: "Test"},
			wantErr: true,
		},
		{
			name: "OK - device removal success",
			args: args{deviceName: "Test"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d, _ := newMockDriver(t)
			if !tt.wantErr {
				d.serverMap[tt.args.deviceName] = &server.Server{}
			}
			if err := d.RemoveDevice(tt.args.deviceName, tt.args.protocols); (err != nil) != tt.wantErr {
				t.Errorf("Driver.RemoveDevice() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDriver_Stop(t *testing.T) {
	type args struct {
		force bool
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name:    "OK - device stopped",
			args:    args{force: false},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d, _ := newMockDriver(t)
			if err := d.Stop(tt.args.force); (err != nil) != tt.wantErr {
				t.Errorf("Driver.Stop() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDriver_HandleReadCommands(t *testing.T) {
	type args struct {
		deviceName string
		protocols  map[string]models.ProtocolProperties
		reqs       []sdkModel.CommandRequest
	}
	tests := []struct {
		name    string
		args    args
		want    []*sdkModel.CommandValue
		wantErr bool
	}{
		{
			name:    "NOK - device not found",
			args:    args{deviceName: "Test", reqs: []sdkModel.CommandRequest{{}}},
			wantErr: true,
			want:    nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d, _ := newMockDriver(t)
			got, err := d.HandleReadCommands(tt.args.deviceName, tt.args.protocols, tt.args.reqs)
			if (err != nil) != tt.wantErr {
				t.Errorf("Driver.HandleReadCommands() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Driver.HandleReadCommands() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDriver_HandleWriteCommands(t *testing.T) {
	type args struct {
		deviceName string
		protocols  map[string]models.ProtocolProperties
		reqs       []sdkModel.CommandRequest
		params     []*sdkModel.CommandValue
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name:    "NOK - device not found",
			args:    args{deviceName: "Test", reqs: []sdkModel.CommandRequest{{}}},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d, _ := newMockDriver(t)
			if err := d.HandleWriteCommands(tt.args.deviceName, tt.args.protocols, tt.args.reqs, tt.args.params); (err != nil) != tt.wantErr {
				t.Errorf("Driver.HandleWriteCommands() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDriver_Initialize(t *testing.T) {
	tests := []struct {
		name    string
		devices []models.Device
		err     error
		wantErr bool
	}{
		{
			name:    "NOK - error adding route",
			err:     fmt.Errorf("error"),
			wantErr: true,
		},
		{
			name:    "OK - no devices",
			devices: []models.Device{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d, dsMock := newMockDriver(t)
			dsMock.On("AddCustomRoute", "/api/v4/call", mock.Anything, mock.AnythingOfType("func(echo.Context) error"), http.MethodPost).Return(tt.err)
			if tt.err == nil {
				dsMock.On("Devices").Return(tt.devices)
			}
			if err := d.Initialize(dsMock); (err != nil) != tt.wantErr {
				t.Errorf("Driver.Initialize() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDriver_ValidateDevice(t *testing.T) {
	tests := []struct {
		name    string
		device  models.Device
		wantErr bool
	}{
		{
			name: "NOK - invalid protocol properties",
			device: models.Device{Protocols: map[string]models.ProtocolProperties{"opcua": {
				"Foobar": make(chan int, 1), // forces marshalling error
			}}},
			wantErr: true,
		},
		{
			name: "OK - valid device",
			device: models.Device{Protocols: map[string]models.ProtocolProperties{"opcua": {
				"Endpoint":  "opc.tcp://test",
				"Policy":    "None",
				"Mode":      "None",
				"Resources": []string{"A", "B", "C"},
				"CertFile":  "",
				"KeyFile":   "",
			}}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d, _ := newMockDriver(t)
			if err := d.ValidateDevice(tt.device); (err != nil) != tt.wantErr {
				t.Errorf("Driver.ValidateDevice() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
