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
	"strings"
	"time"

	"github.com/edgexfoundry/device-opcua-go/pkg/result"
	sdkModels "github.com/edgexfoundry/device-sdk-go/v2/pkg/models"
	"github.com/edgexfoundry/device-sdk-go/v2/pkg/service"
	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/ua"
)

// StartSubscriptionListener initializes a new OPCUA client and subscribes to the resources
// specified by the user in the device protocol configuration
func (s *Server) StartSubscriptionListener() error {
	ds := service.RunningService()
	if ds == nil {
		return fmt.Errorf("[%s] unable to get running device service", s.deviceName)
	}

	device, err := ds.GetDeviceByName(s.deviceName)
	if err != nil {
		return err
	}

	serverConfig, err := NewConfig(device.Protocols["opcua"])
	if err != nil {
		return err
	}

	if err := s.initClient(serverConfig); err != nil {
		return err
	}

	if err := s.client.Connect(s.client.ctx); err != nil {
		s.logger.Warnf("[%s] failed to connect OPCUA client: %v", s.deviceName, err)
		return err
	}
	defer s.client.Close()

	notifyCh := make(chan *opcua.PublishNotificationData)

	sub, err := s.client.Subscribe(
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
				s.logger.Debug(res.Error.Error())
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

	s.client = &Client{
		opcua.NewClient(ep.EndpointURL, opts...),
		context.Background(),
	}

	return nil
}

func (s *Server) configureMonitoredItems(sub *opcua.Subscription, resources string) error {
	ds := service.RunningService()
	if ds == nil {
		return fmt.Errorf("[%s] unable to get running device service", s.deviceName)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	for i, resource := range strings.Split(resources, ",") {
		deviceResource, ok := ds.DeviceResource(s.deviceName, resource)
		if !ok {
			s.logger.Warnf("[%s] unable to find resource with name %s", s.deviceName, resource)
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
		res, err := sub.Monitor(ua.TimestampsToReturnBoth, miCreateRequest)
		if err != nil || res.Results[0].StatusCode != ua.StatusOK {
			return err
		}

		s.logger.Infof("[%s] start incoming data listening for %s", s.deviceName, resource)
	}

	return nil
}

func (s *Server) handleDataChange(dcn *ua.DataChangeNotification) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, item := range dcn.MonitoredItems {
		data := item.Value.Value.Value()
		resourceName := s.resourceMap[item.ClientHandle]
		if err := s.onIncomingDataReceived(data, resourceName); err != nil {
			s.logger.Errorf("%v", err)
		}
	}
}

func (s *Server) onIncomingDataReceived(data interface{}, nodeResourceName string) error {
	ds := service.RunningService()
	if ds == nil {
		return fmt.Errorf("[%s] unable to get running device service", s.deviceName)
	}

	deviceResource, ok := ds.DeviceResource(s.deviceName, nodeResourceName)
	if !ok {
		s.logger.Warnf("[%s] Incoming reading ignored. No DeviceObject found: deviceResource=%v value=%v", s.deviceName, nodeResourceName, data)
		return nil
	}

	req := sdkModels.CommandRequest{
		DeviceResourceName: nodeResourceName,
		Type:               deviceResource.Properties.ValueType,
	}

	reading := data
	result, err := result.NewResult(req, reading)
	if err != nil {
		s.logger.Warnf("[%s] Incoming reading ignored. deviceResource=%v value=%v", s.deviceName, nodeResourceName, data)
		return nil
	}

	asyncValues := &sdkModels.AsyncValues{
		DeviceName:    s.deviceName,
		CommandValues: []*sdkModels.CommandValue{result},
	}

	s.logger.Infof("[%s] Incoming reading received: deviceResource=%v value=%v", s.deviceName, nodeResourceName, data)

	s.asyncChannel <- asyncValues

	return nil
}
