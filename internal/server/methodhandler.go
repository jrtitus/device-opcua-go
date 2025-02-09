// -*- Mode: Go; indent-tabs-mode: t -*-
//
// Copyright (C) 2022 Schneider Electric
//
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"fmt"

	"github.com/edgexfoundry/go-mod-core-contracts/v4/models"
	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/ua"
)

func (s *Server) ProcessMethodCall(method string, parameters []string) (interface{}, error) {
	device, err := s.sdk.GetDeviceByName(s.deviceName)
	if err != nil {
		return nil, fmt.Errorf("device not found: %v", err)
	}

	if device.AdminState == models.Locked || device.OperatingState == models.Down {
		return nil, fmt.Errorf("method [%s] not processed for [%s]: device is locked or down", method, s.deviceName)
	}

	resource, ok := s.sdk.DeviceResource(s.deviceName, method)
	if !ok {
		return nil, fmt.Errorf("method not found")
	}

	return s.makeMethodCall(resource, parameters)
}

func (s *Server) makeMethodCall(resource models.DeviceResource, parameters []string) (interface{}, error) {
	if resource.IsHidden {
		return nil, fmt.Errorf("Server.makeMethodCall: method call not allowed")
	}

	oid, err := getNodeID(resource.Attributes, OBJECT)
	if err != nil {
		return nil, fmt.Errorf("Server.makeMethodCall: %v", err)
	}

	mid, err := getNodeID(resource.Attributes, METHOD)
	if err != nil {
		return nil, fmt.Errorf("Server.makeMethodCall: %v", err)
	}

	var inputs []*ua.Variant
	if len(parameters) > 0 {
		inputs = make([]*ua.Variant, len(parameters))
		for i := 0; i < len(parameters); i++ {
			inputs[i] = ua.MustVariant(parameters[i])
		}
	}

	request := &ua.CallMethodRequest{
		ObjectID:       oid,
		MethodID:       mid,
		InputArguments: inputs,
	}

	if s.client == nil || s.client.State() == opcua.Closed || s.client.State() == opcua.Disconnected {
		if err := s.Connect(); err != nil {
			return nil, fmt.Errorf("Server.makeMethodCall: client not initialized: %s", err)
		}
	}

	resp, err := s.client.Call(s.client.ctx, request)
	if err != nil {
		return nil, fmt.Errorf("Server.makeMethodCall: Method call failed: %s", err)
	}
	if resp.StatusCode != ua.StatusOK {
		return nil, fmt.Errorf("Server.makeMethodCall: Method status not OK: %v", resp.StatusCode)
	}

	return resp.OutputArguments[0].Value(), nil
}
