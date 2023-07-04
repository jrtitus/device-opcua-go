// -*- Mode: Go; indent-tabs-mode: t -*-
//
// Copyright (C) 2022 Schneider Electric
//
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"fmt"

	"github.com/edgexfoundry/go-mod-core-contracts/v3/models"
	"github.com/gopcua/opcua/ua"
)

func (s *Server) ProcessMethodCall(method string, parameters []string) (interface{}, error) {
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

	resp, err := s.client.Call(request)
	if err != nil {
		return nil, fmt.Errorf("Server.makeMethodCall: Method call failed: %s", err)
	}
	if resp.StatusCode != ua.StatusOK {
		return nil, fmt.Errorf("Server.makeMethodCall: Method status not OK: %v", resp.StatusCode)
	}

	return resp.OutputArguments[0].Value(), nil
}
