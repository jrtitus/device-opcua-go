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
	"sync"

	"github.com/edgexfoundry/device-opcua-go/internal/server"
	sdkModel "github.com/edgexfoundry/device-sdk-go/v2/pkg/models"
	"github.com/edgexfoundry/device-sdk-go/v2/pkg/service"
	"github.com/edgexfoundry/go-mod-core-contracts/v2/clients/logger"
	"github.com/edgexfoundry/go-mod-core-contracts/v2/models"
)

var once sync.Once
var driver *Driver

// Driver struct
type Driver struct {
	Logger    logger.LoggingClient
	AsyncCh   chan<- *sdkModel.AsyncValues
	mu        sync.Mutex
	serverMap map[string]*server.Server
}

// NewProtocolDriver returns a new protocol driver object
func NewProtocolDriver() sdkModel.ProtocolDriver {
	once.Do(func() {
		driver = new(Driver)
	})
	return driver
}

// Initialize performs protocol-specific initialization for the device service
func (d *Driver) Initialize(lc logger.LoggingClient, asyncCh chan<- *sdkModel.AsyncValues, deviceCh chan<- []sdkModel.DiscoveredDevice) error {
	d.Logger = lc
	d.AsyncCh = asyncCh
	d.mu.Lock()
	d.serverMap = make(map[string]*server.Server)
	d.mu.Unlock()

	ds := service.RunningService()
	if ds == nil {
		return fmt.Errorf("unable to get device service instance")
	}

	// When the service is initialized, add pre-existing devices to the server map
	for _, v := range ds.Devices() {
		if err := d.AddDevice(v.Name, v.Protocols, v.AdminState); err != nil {
			d.Logger.Errorf("[%s] error adding device to server map: %v", v.Name, err)
		}
	}

	return nil
}

// AddDevice is a callback function that is invoked
// when a new Device associated with this Device Service is added
func (d *Driver) AddDevice(deviceName string, protocols map[string]models.ProtocolProperties, adminState models.AdminState) error {
	d.Logger.Debugf("Device %s is added. Starting subscription mechanism...", deviceName)
	d.mu.Lock()
	s := server.NewServer(deviceName, d.Logger, d.AsyncCh)
	d.serverMap[deviceName] = s
	d.mu.Unlock()

	go s.StartSubscriptionListener() // nolint:errcheck
	return nil
}

// UpdateDevice is a callback function that is invoked
// when a Device associated with this Device Service is updated
func (d *Driver) UpdateDevice(deviceName string, protocols map[string]models.ProtocolProperties, adminState models.AdminState) error {
	d.Logger.Debugf("Device %s is updated. Restarting subscription mechanism...", deviceName)
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
	d.Logger.Debugf("Device %s is removed. Cleaning up...", deviceName)
	if s, ok := d.serverMap[deviceName]; ok {
		s.Cleanup(false)
		d.serverMap[deviceName] = nil
		return nil
	}
	return serverNotFoundError(deviceName)
}

// Stop the protocol-specific DS code to shutdown gracefully, or
// if the force parameter is 'true', immediately. The driver is responsible
// for closing any in-use channels, including the channel used to send async
// readings (if supported).
func (d *Driver) Stop(force bool) error {
	d.mu.Lock()
	d.serverMap = nil
	d.AsyncCh = nil
	d.mu.Unlock()
	return nil
}

// HandleReadCommands triggers a protocol Read operation for the specified device.
func (d *Driver) HandleReadCommands(deviceName string, protocols map[string]models.ProtocolProperties,
	reqs []sdkModel.CommandRequest) ([]*sdkModel.CommandValue, error) {

	d.Logger.Debugf("Driver.HandleReadCommands: protocols: %v resource: %v attributes: %v", protocols, reqs[0].DeviceResourceName, reqs[0].Attributes)

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

	d.Logger.Debugf("Driver.HandleWriteCommands: protocols: %v, resource: %v, parameters: %v", protocols, reqs[0].DeviceResourceName, params)

	s, ok := d.serverMap[deviceName]
	if !ok {
		return serverNotFoundError(deviceName)
	}

	return s.ProcessWriteCommands(reqs, params)
}

func serverNotFoundError(deviceName string) error {
	return fmt.Errorf("unable to find device %s in server map", deviceName)
}
