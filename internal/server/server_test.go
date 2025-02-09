// -*- Mode: Go; indent-tabs-mode: t -*-
//
// Copyright (C) 2022 Schneider Electric
//
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"fmt"
	"testing"

	"github.com/edgexfoundry/device-opcua-go/internal/test"
	"github.com/edgexfoundry/device-sdk-go/v4/pkg/interfaces/mocks"
	"github.com/edgexfoundry/go-mod-core-contracts/v4/models"
	"github.com/stretchr/testify/assert"
)

func TestNewServer(t *testing.T) {
	t.Run("create server", func(t *testing.T) {

		s := NewServer("test", mocks.NewDeviceServiceSDK(t))
		if s == nil {
			t.Error("NewServer() failed")
		}
		s.Cleanup(false)
	})
}

func TestServer_Cleanup(t *testing.T) {
	tests := []struct {
		name             string
		recreateContext  bool
		startWithContext bool
	}{
		{
			name:             "Mock update",
			recreateContext:  true,
			startWithContext: true,
		},
		{
			name:             "Mock delete",
			recreateContext:  false,
			startWithContext: true,
		},
		{
			name:             "Start with no context",
			startWithContext: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Server{}
			if tt.startWithContext {
				s.newContext()
			}
			s.Cleanup(tt.recreateContext)
		})
	}
}
func TestServer_Connect(t *testing.T) {
	deviceName := "testDevice"
	mockDevice := models.Device{
		Name:           deviceName,
		AdminState:     models.Unlocked,
		OperatingState: models.Up,
		Protocols: map[string]models.ProtocolProperties{
			"opcua": {
				"endpoint": "opc.tcp://localhost:48408",
			},
		},
	}

	s := test.NewServer("../test/opcua_server.py")
	defer s.Close()

	t.Run("Connect with unlocked and up device", func(t *testing.T) {
		mockSDK := mocks.NewDeviceServiceSDK(t)
		mockSDK.On("GetDeviceByName", deviceName).Return(mockDevice, nil)

		server := NewServer(deviceName, mockSDK)
		err := server.Connect()
		assert.NoError(t, err)
	})

	t.Run("Connect with locked device", func(t *testing.T) {
		mockDevice.AdminState = models.Locked
		mockSDK := mocks.NewDeviceServiceSDK(t)
		mockSDK.On("GetDeviceByName", deviceName).Return(mockDevice, nil)

		server := NewServer(deviceName, mockSDK)
		err := server.Connect()
		assert.Error(t, err)
		assert.EqualError(t, err, fmt.Sprintf("client not started for [%s]: device is locked or down", deviceName))
	})

	t.Run("Connect with device in down state", func(t *testing.T) {
		mockDevice.AdminState = models.Unlocked
		mockDevice.OperatingState = models.Down
		mockSDK := mocks.NewDeviceServiceSDK(t)
		mockSDK.On("GetDeviceByName", deviceName).Return(mockDevice, nil)

		server := NewServer(deviceName, mockSDK)
		err := server.Connect()
		assert.Error(t, err)
		assert.EqualError(t, err, fmt.Sprintf("client not started for [%s]: device is locked or down", deviceName))
	})

	t.Run("Connect with error getting server config", func(t *testing.T) {
		mockDevice.AdminState = models.Unlocked
		mockDevice.OperatingState = models.Up
		mockSDK := mocks.NewDeviceServiceSDK(t)
		mockSDK.On("GetDeviceByName", deviceName).Return(models.Device{}, fmt.Errorf("error getting device"))

		server := NewServer(deviceName, mockSDK)
		err := server.Connect()
		assert.Error(t, err)
		assert.EqualError(t, err, "error getting device")
	})
}
