// -*- Mode: Go; indent-tabs-mode: t -*-
//
// Copyright (C) 2018 Canonical Ltd
// Copyright (C) 2018 IOTech Ltd
// Copyright (C) 2021 Schneider Electric
//
// SPDX-License-Identifier: Apache-2.0

package driver

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/edgexfoundry/device-opcua-go/internal/server"
	"github.com/edgexfoundry/device-sdk-go/v4/pkg/interfaces"
	sdkModel "github.com/edgexfoundry/device-sdk-go/v4/pkg/models"
	"github.com/edgexfoundry/go-mod-core-contracts/v4/models"
)

var once sync.Once
var driver *Driver

// Driver struct
type Driver struct {
	mu        sync.Mutex
	serverMap map[string]*server.Server
	sdk       interfaces.DeviceServiceSDK
}

// NewProtocolDriver returns a new protocol driver object
func NewProtocolDriver() interfaces.ProtocolDriver {
	once.Do(func() {
		driver = new(Driver)
	})
	return driver
}

// Initialize performs protocol-specific initialization for the device service
func (d *Driver) Initialize(sdk interfaces.DeviceServiceSDK) error {
	d.sdk = sdk

	// Define custom API endpoints
	if err := d.sdk.AddCustomRoute("/api/v4/call", interfaces.Authenticated, handleMethodCall, http.MethodPost); err != nil {
		return fmt.Errorf("unable to add custom route to device service: %v", err)
	}

	d.mu.Lock()
	d.serverMap = make(map[string]*server.Server)
	d.mu.Unlock()

	// When the service is initialized, add pre-existing devices to the server map
	for _, v := range d.sdk.Devices() {
		if err := d.AddDevice(v.Name, v.Protocols, v.AdminState); err != nil {
			d.sdk.LoggingClient().Errorf("[%s] error adding device to server map: %v", v.Name, err)
		}
	}

	return nil
}

// AddDevice is a callback function that is invoked
// when a new Device associated with this Device Service is added
func (d *Driver) AddDevice(deviceName string, protocols map[string]models.ProtocolProperties, adminState models.AdminState) error {
	d.sdk.LoggingClient().Debugf("Device %s is added. Starting subscription mechanism...", deviceName)
	d.mu.Lock()
	s := server.NewServer(deviceName, d.sdk)
	d.serverMap[deviceName] = s
	d.mu.Unlock()

	go s.StartSubscriptionListener() // nolint:errcheck
	return nil
}

// UpdateDevice is a callback function that is invoked
// when a Device associated with this Device Service is updated
func (d *Driver) UpdateDevice(deviceName string, protocols map[string]models.ProtocolProperties, adminState models.AdminState) error {
	d.sdk.LoggingClient().Debugf("Device %s is updated. Restarting subscription mechanism...", deviceName)
	if s, ok := d.serverMap[deviceName]; ok {
		s.Cleanup(true)
		go s.StartSubscriptionListener() // nolint:errcheck
		return nil
	}

	return serverNotFoundError(deviceName)
}

// RemoveDevice is a callback function that is invoked
// when a Device associated with this Device Service is removed
func (d *Driver) RemoveDevice(deviceName string, protocols map[string]models.ProtocolProperties) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.sdk.LoggingClient().Debugf("Device %s is removed. Cleaning up...", deviceName)
	if s, ok := d.serverMap[deviceName]; ok {
		s.Cleanup(false)
		d.serverMap[deviceName] = nil
		return nil
	}

	return serverNotFoundError(deviceName)
}

func (d *Driver) ValidateDevice(device models.Device) error {
	cfg, err := server.NewConfig(device.Protocols["opcua"])
	if err != nil {
		return fmt.Errorf("error reading protocol properties, %v", err)
	}

	return server.Validate(cfg)
}

func (d *Driver) Discover() error {
	return fmt.Errorf("driver's Discover function isn't implemented")
}

func (d *Driver) Start() error {
	return nil
}

// Stop the protocol-specific DS code to shutdown gracefully, or
// if the force parameter is 'true', immediately. The driver is responsible
// for closing any in-use channels, including the channel used to send async
// readings (if supported).
func (d *Driver) Stop(force bool) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.serverMap = nil
	d.sdk = nil
	return nil
}

// HandleReadCommands triggers a protocol Read operation for the specified device.
func (d *Driver) HandleReadCommands(deviceName string, protocols map[string]models.ProtocolProperties,
	reqs []sdkModel.CommandRequest) ([]*sdkModel.CommandValue, error) {

	startTime := time.Now()

	defer func() {
		d.sdk.LoggingClient().Debugf("Driver.HandleReadCommands (%v)", time.Since(startTime))
	}()

	s, ok := d.serverMap[deviceName]
	if !ok {
		return nil, serverNotFoundError(deviceName)
	}

	return s.ProcessReadCommands(reqs)
}

// HandleWriteCommands passes a slice of CommandRequest struct each representing
// a ResourceOperation for a specific device resource (aka DeviceObject).
// Since the commands are actuation commands, params provide parameters for the individual
// command.
func (d *Driver) HandleWriteCommands(deviceName string, protocols map[string]models.ProtocolProperties,
	reqs []sdkModel.CommandRequest, params []*sdkModel.CommandValue) error {

	d.sdk.LoggingClient().Debugf("Driver.HandleWriteCommands: protocols: %v, resource: %v, parameters: %v", protocols, reqs[0].DeviceResourceName, params)

	s, ok := d.serverMap[deviceName]
	if !ok {
		return serverNotFoundError(deviceName)
	}

	return s.ProcessWriteCommands(reqs, params)
}

func serverNotFoundError(deviceName string) error {
	return fmt.Errorf("unable to find device %s in server map", deviceName)
}
