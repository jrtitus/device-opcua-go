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

	"github.com/edgexfoundry/device-opcua-go/pkg/command"
	sdkModel "github.com/edgexfoundry/device-sdk-go/v4/pkg/models"
	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/ua"
)

func (s *Server) ProcessWriteCommands(reqs []sdkModel.CommandRequest, params []*sdkModel.CommandValue) error {
	for i, req := range reqs {
		err := s.handleWriteCommandRequest(req, params[i])
		if err != nil {
			s.sdk.LoggingClient().Errorf("Driver.HandleWriteCommands: Handle write commands failed: %v", err)
			return err
		}
	}

	return nil
}

func (s *Server) handleWriteCommandRequest(req sdkModel.CommandRequest,
	param *sdkModel.CommandValue) error {

	id, err := getNodeID(req.Attributes, NODE)
	if err != nil {
		return fmt.Errorf("Driver.handleWriteCommands: invalid node id: %v", err)
	}

	value, err := command.NewValue(req.Type, param)
	if err != nil {
		return err
	}

	v, err := ua.NewVariant(value)
	if err != nil {
		return fmt.Errorf("Driver.handleWriteCommands: invalid value: %v", err)
	}

	request := &ua.WriteRequest{
		NodesToWrite: []*ua.WriteValue{
			{
				NodeID:      id,
				AttributeID: ua.AttributeIDValue,
				Value: &ua.DataValue{
					EncodingMask: ua.DataValueValue, // encoding mask
					Value:        v,
				},
			},
		},
	}

	if s.client == nil || s.client.State() == opcua.Closed || s.client.State() == opcua.Disconnected {
		if err := s.Connect(); err != nil {
			return fmt.Errorf("Driver.handleWriteCommands: client not initialized: %s", err)
		}
	}

	resp, err := s.client.Write(s.client.ctx, request)
	if err != nil {
		s.sdk.LoggingClient().Errorf("Driver.handleWriteCommands: Write value %v failed: %s", v, err)
		return err
	}
	s.sdk.LoggingClient().Infof("Driver.handleWriteCommands: write sucessfully, %v", resp.Results[0])
	return nil
}
