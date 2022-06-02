// -*- Mode: Go; indent-tabs-mode: t -*-
//
// Copyright (C) 2021 Schneider Electric
//
// SPDX-License-Identifier: Apache-2.0

package driver

import (
	"reflect"
	"testing"

	"github.com/edgexfoundry/device-opcua-go/internal/server"
	sdkModel "github.com/edgexfoundry/device-sdk-go/v2/pkg/models"
	"github.com/edgexfoundry/go-mod-core-contracts/v2/clients/logger"
	"github.com/edgexfoundry/go-mod-core-contracts/v2/models"
)

func newMockDriver() *Driver {
	d := NewProtocolDriver().(*Driver)
	d.Logger = logger.MockLogger{}
	d.AsyncCh = make(chan<- *sdkModel.AsyncValues)
	d.serverMap = make(map[string]*server.Server)
	return d
}

func TestDriver_AddDevice(t *testing.T) {
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
			name:    "OK - device add success",
			args:    args{deviceName: "Test"},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := newMockDriver()
			if err := d.AddDevice(tt.args.deviceName, tt.args.protocols, tt.args.adminState); (err != nil) != tt.wantErr {
				t.Errorf("Driver.AddDevice() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
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
		{
			name:    "OK - device update success",
			args:    args{deviceName: "Test"},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := newMockDriver()
			if tt.wantErr == false {
				_ = d.AddDevice(tt.args.deviceName, tt.args.protocols, tt.args.adminState)
			}
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
		{
			name:    "OK - device removal success",
			args:    args{deviceName: "Test"},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := newMockDriver()
			if tt.wantErr == false {
				_ = d.AddDevice(tt.args.deviceName, tt.args.protocols, tt.args.adminState)
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
			d := newMockDriver()
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
			d := newMockDriver()
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
			d := newMockDriver()
			if err := d.HandleWriteCommands(tt.args.deviceName, tt.args.protocols, tt.args.reqs, tt.args.params); (err != nil) != tt.wantErr {
				t.Errorf("Driver.HandleWriteCommands() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDriver_Initialize(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
	}{
		{
			name:    "NOK - expect error from call to RunningService in test",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &Driver{}
			if err := d.Initialize(logger.MockLogger{}, make(chan<- *sdkModel.AsyncValues), make(chan<- []sdkModel.DiscoveredDevice)); (err != nil) != tt.wantErr {
				t.Errorf("Driver.Initialize() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
