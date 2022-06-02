// -*- Mode: Go; indent-tabs-mode: t -*-
//
// Copyright (C) 2021 Schneider Electric
//
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"testing"

	"github.com/edgexfoundry/device-opcua-go/internal/test"
	sdkModel "github.com/edgexfoundry/device-sdk-go/v2/pkg/models"
	"github.com/edgexfoundry/go-mod-core-contracts/v2/clients/logger"
	"github.com/gopcua/opcua/ua"
)

func Test_StartSubscriptionListener(t *testing.T) {
	t.Run("call and exit", func(t *testing.T) {
		s := &Server{
			deviceName: "Test",
		}
		err := s.StartSubscriptionListener()
		if err == nil {
			t.Error("expected err to exist in test environment")
		}
	})
}

func Test_configureMonitoredItems(t *testing.T) {
	t.Run("call and exit", func(t *testing.T) {
		s := &Server{
			deviceName: "Test",
		}
		err := s.configureMonitoredItems(nil, "a,b,c")
		if err == nil {
			t.Error("expected err to exist in test environment")
		}
	})
}

func Test_onIncomingDataReceived(t *testing.T) {
	t.Run("set reading and exit", func(t *testing.T) {
		s := &Server{
			deviceName: "Test",
		}
		err := s.onIncomingDataReceived("42", "TestResource")
		if err == nil {
			t.Error("expected err to exist in test environment")
		}
	})
}

func TestDriver_initClient(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name:    "NOK - no endpoint configured",
			config:  &Config{},
			wantErr: true,
		},
		{
			name:    "NOK - no server connection",
			config:  &Config{Endpoint: "opc.tcp://test"},
			wantErr: true,
		},
		{
			name: "OK",
			config: &Config{
				Endpoint: test.Protocol + test.Address,
				Policy:   "None",
				Mode:     "None",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ch := make(chan *sdkModel.AsyncValues)
			s := NewServer("Test", logger.MockLogger{}, ch)

			if !tt.wantErr {
				server := test.NewServer("../test/opcua_server.py")
				defer server.Close()
			}

			err := s.initClient(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("Driver.getClient() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestDriver_handleDataChange(t *testing.T) {
	tests := []struct {
		name        string
		resourceMap map[uint32]string
		dcn         *ua.DataChangeNotification
	}{
		{
			name: "OK - no monitored items",
			dcn:  &ua.DataChangeNotification{MonitoredItems: make([]*ua.MonitoredItemNotification, 0)},
		},
		{
			name:        "OK - call onIncomingDataReceived",
			resourceMap: map[uint32]string{123456: "TestResource"},
			dcn: &ua.DataChangeNotification{
				MonitoredItems: []*ua.MonitoredItemNotification{
					{ClientHandle: 123456, Value: &ua.DataValue{Value: ua.MustVariant("42")}},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ch := make(chan *sdkModel.AsyncValues)
			s := NewServer("Test", logger.MockLogger{}, ch)
			s.handleDataChange(tt.dcn)
		})
	}
}
