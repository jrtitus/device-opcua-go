// -*- Mode: Go; indent-tabs-mode: t -*-
//
// Copyright (C) 2018 Canonical Ltd
// Copyright (C) 2018 IOTech Ltd
// Copyright (C) 2021 Schneider Electric
//
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"context"
	"fmt"
	"time"

	"github.com/edgexfoundry/device-opcua-go/pkg/result"
	sdkModels "github.com/edgexfoundry/device-sdk-go/v3/pkg/models"
	"github.com/edgexfoundry/go-mod-core-contracts/v3/models"
	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/ua"
)

// StartSubscriptionListener initializes a new OPCUA client and subscribes to the resources
// specified by the user in the device protocol configuration
func (s *Server) StartSubscriptionListener() error {
	device, err := s.sdk.GetDeviceByName(s.deviceName)
	if err != nil {
		return err
	}

	if device.AdminState == models.Locked || device.OperatingState == models.Down {
		s.sdk.LoggingClient().Warnf("subscription listener not started for [%s]: device is locked or down", s.deviceName)
		return nil
	}

	serverConfig, err := NewConfig(device.Protocols["opcua"])
	if err != nil {
		return err
	}

	if err := s.initClient(serverConfig); err != nil {
		return err
	}

	if err := s.client.Connect(s.client.ctx); err != nil {
		s.sdk.LoggingClient().Warnf("[%s] failed to connect OPCUA client: %v", s.deviceName, err)
		return err
	}
	defer s.client.Close(s.client.ctx)

	notifyCh := make(chan *opcua.PublishNotificationData)

	sub, err := s.client.Subscribe(s.client.ctx,
		&opcua.SubscriptionParameters{
			Interval: time.Duration(500) * time.Millisecond,
		}, notifyCh)
	if err != nil {
		return err
	}
	defer sub.Cancel(s.client.ctx) //nolint:errcheck

	if err := s.configureMonitoredItems(sub, serverConfig.Resources); err != nil {
		return err
	}

	// read from subscription's notification channel until ctx is cancelled
	for {
		select {
		// context return
		case <-s.context.ctx.Done():
			return nil
		// receive Publish Notification Data
		case res := <-notifyCh:
			if res.Error != nil {
				s.sdk.LoggingClient().Debug(res.Error.Error())
				continue
			}
			switch dataChangeNotification := res.Value.(type) {
			// result type: DateChange StatusChange
			case *ua.DataChangeNotification:
				s.handleDataChange(dataChangeNotification)
			}
		}
	}
}

func (s *Server) initClient(config *Config) error {

	endpoints, err := opcua.GetEndpoints(s.context.ctx, config.Endpoint)
	if err != nil {
		return err
	}

	ep := opcua.SelectEndpoint(endpoints, config.Policy, ua.MessageSecurityModeFromString(config.Mode))
	if ep == nil {
		return fmt.Errorf("[%s] failed to find suitable endpoint", s.deviceName)
	}
	ep.EndpointURL = config.Endpoint

	opts := []opcua.Option{
		opcua.SecurityPolicy(config.Policy),
		opcua.SecurityModeString(config.Mode),
		opcua.CertificateFile(config.CertFile),
		opcua.PrivateKeyFile(config.KeyFile),
		opcua.AuthAnonymous(),
		opcua.SecurityFromEndpoint(ep, ua.UserTokenTypeAnonymous),
	}

	uaClient, err := opcua.NewClient(ep.EndpointURL, opts...)
	if err != nil {
		return err
	}

	s.client = &Client{
		uaClient,
		context.Background(),
	}

	return nil
}

func (s *Server) configureMonitoredItems(sub *opcua.Subscription, resources []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, resource := range resources {
		deviceResource, ok := s.sdk.DeviceResource(s.deviceName, resource)
		if !ok {
			s.sdk.LoggingClient().Warnf("[%s] unable to find resource with name %s", s.deviceName, resource)
			continue
		}

		id, err := getNodeID(deviceResource.Attributes, NODE)
		if err != nil {
			return err
		}

		// arbitrary client handle for the monitoring item
		handle := uint32(i + 42)
		// map the client handle so we know what the value returned represents
		s.resourceMap[handle] = resource
		miCreateRequest := opcua.NewMonitoredItemCreateRequestWithDefaults(id, ua.AttributeIDValue, handle)
		res, err := sub.Monitor(s.client.ctx, ua.TimestampsToReturnBoth, miCreateRequest)
		if err != nil || res.Results[0].StatusCode != ua.StatusOK {
			return err
		}

		s.sdk.LoggingClient().Infof("[%s] start incoming data listening for %s", s.deviceName, resource)
	}

	return nil
}

func (s *Server) handleDataChange(dcn *ua.DataChangeNotification) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, item := range dcn.MonitoredItems {
		var data any

		variant := item.Value.Value
		if variant != nil {
			data = variant.Value()
		} else {
			continue
		}
		resourceName := s.resourceMap[item.ClientHandle]
		if err := s.onIncomingDataReceived(data, resourceName); err != nil {
			s.sdk.LoggingClient().Errorf("%v", err)
		}
	}
}

func (s *Server) onIncomingDataReceived(data interface{}, nodeResourceName string) error {
	deviceResource, ok := s.sdk.DeviceResource(s.deviceName, nodeResourceName)
	if !ok {
		return fmt.Errorf("[%s] Incoming reading ignored. No DeviceObject found: deviceResource=%v value=%v", s.deviceName, nodeResourceName, data)
	}

	req := sdkModels.CommandRequest{
		DeviceResourceName: nodeResourceName,
		Type:               deviceResource.Properties.ValueType,
	}

	reading := data
	result, err := result.NewResult(req, reading)
	if err != nil {
		return fmt.Errorf("[%s] Incoming reading ignored. deviceResource=%v value=%v", s.deviceName, nodeResourceName, data)
	}

	asyncValues := &sdkModels.AsyncValues{
		DeviceName:    s.deviceName,
		CommandValues: []*sdkModels.CommandValue{result},
	}

	s.sdk.LoggingClient().Infof("[%s] Incoming reading received: deviceResource=%v value=%v", s.deviceName, nodeResourceName, data)

	s.sdk.AsyncValuesChannel() <- asyncValues

	return nil
}
