// -*- Mode: Go; indent-tabs-mode: t -*-
//
// Copyright (C) 2018 Canonical Ltd
// Copyright (C) 2018 IOTech Ltd
// Copyright (C) 2021 Schneider Electric
//
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"fmt"

	"github.com/edgexfoundry/device-opcua-go/pkg/result"
	sdkModel "github.com/edgexfoundry/device-sdk-go/v3/pkg/models"
	"github.com/gopcua/opcua/ua"
)

func (s *Server) ProcessReadCommands(reqs []sdkModel.CommandRequest) ([]*sdkModel.CommandValue, error) {
	var responses = make([]*sdkModel.CommandValue, len(reqs))

	for i, req := range reqs {
		res, err := s.handleReadCommandRequest(req)
		if err != nil {
			s.sdk.LoggingClient().Errorf("Driver.HandleReadCommands: Handle read commands failed: %v", err)
			return responses, err
		}
		s.sdk.LoggingClient().Infof("Read command finished: %v", res)
		responses[i] = res
	}

	return responses, nil
}

func (s *Server) handleReadCommandRequest(req sdkModel.CommandRequest) (*sdkModel.CommandValue, error) {
	if _, isMethod := req.Attributes[METHOD]; isMethod {
		return nil, fmt.Errorf("not allowed to call command on method: %s", req.DeviceResourceName)
	}

	return s.makeReadRequest(req)
}

func (s *Server) makeReadRequest(req sdkModel.CommandRequest) (*sdkModel.CommandValue, error) {
	id, err := getNodeID(req.Attributes, NODE)
	if err != nil {
		return nil, fmt.Errorf("Driver.handleReadCommands: Invalid node id = %v", err)
	}

	request := &ua.ReadRequest{
		MaxAge: 2000,
		NodesToRead: []*ua.ReadValueID{
			{NodeID: id},
		},
		TimestampsToReturn: ua.TimestampsToReturnBoth,
	}
	resp, err := s.client.Read(s.client.ctx, request)
	if err != nil {
		return nil, fmt.Errorf("Driver.handleReadCommands: Read failed: %s", err)
	}
	if resp.Results[0].Status != ua.StatusOK {
		return nil, fmt.Errorf("Driver.handleReadCommands: Status not OK: %v", resp.Results[0].Status)
	}

	reading := resp.Results[0].Value.Value()
	return result.NewResult(req, reading)
}
