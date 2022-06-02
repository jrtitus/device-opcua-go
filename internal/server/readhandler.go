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
	sdkModel "github.com/edgexfoundry/device-sdk-go/v2/pkg/models"
	"github.com/gopcua/opcua/ua"
)

func (s *Server) ProcessReadCommands(reqs []sdkModel.CommandRequest) ([]*sdkModel.CommandValue, error) {
	var responses = make([]*sdkModel.CommandValue, len(reqs))

	for i, req := range reqs {
		// handle every reqs
		res, err := s.handleReadCommandRequest(req)
		if err != nil {
			s.logger.Errorf("Driver.HandleReadCommands: Handle read commands failed: %v", err)
			return responses, err
		}
		responses[i] = res
	}

	return responses, nil
}

func (s *Server) handleReadCommandRequest(req sdkModel.CommandRequest) (*sdkModel.CommandValue, error) {
	var result *sdkModel.CommandValue
	var err error

	_, isMethod := req.Attributes[METHOD]

	if isMethod {
		result, err = s.makeMethodCall(req)
		s.logger.Infof("Method command finished: %v", result)
	} else {
		result, err = s.makeReadRequest(req)
		s.logger.Infof("Read command finished: %v", result)
	}

	return result, err
}

func (s *Server) makeReadRequest(req sdkModel.CommandRequest) (*sdkModel.CommandValue, error) {
	nodeID, err := getNodeID(req.Attributes, NODE)
	if err != nil {
		return nil, fmt.Errorf("Driver.handleReadCommands: %v", err)
	}

	id, err := ua.ParseNodeID(nodeID)
	if err != nil {
		return nil, fmt.Errorf("Driver.handleReadCommands: Invalid node id=%s; %v", nodeID, err)
	}

	request := &ua.ReadRequest{
		MaxAge: 2000,
		NodesToRead: []*ua.ReadValueID{
			{NodeID: id},
		},
		TimestampsToReturn: ua.TimestampsToReturnBoth,
	}
	resp, err := s.client.Read(request)
	if err != nil {
		return nil, fmt.Errorf("Driver.handleReadCommands: Read failed: %s", err)
	}
	if resp.Results[0].Status != ua.StatusOK {
		return nil, fmt.Errorf("Driver.handleReadCommands: Status not OK: %v", resp.Results[0].Status)
	}

	// make new result
	reading := resp.Results[0].Value.Value()
	return result.NewResult(req, reading)
}

func (s *Server) makeMethodCall(req sdkModel.CommandRequest) (*sdkModel.CommandValue, error) {
	var inputs []*ua.Variant

	objectID, err := getNodeID(req.Attributes, OBJECT)
	if err != nil {
		return nil, fmt.Errorf("Driver.handleReadCommands: %v", err)
	}
	oid, err := ua.ParseNodeID(objectID)
	if err != nil {
		return nil, fmt.Errorf("Driver.handleReadCommands: %v", err)
	}

	methodID, err := getNodeID(req.Attributes, METHOD)
	if err != nil {
		return nil, fmt.Errorf("Driver.handleReadCommands: %v", err)
	}
	mid, err := ua.ParseNodeID(methodID)
	if err != nil {
		return nil, fmt.Errorf("Driver.handleReadCommands: %v", err)
	}

	inputMap, ok := req.Attributes[INPUTMAP]
	if ok {
		imElements := inputMap.([]interface{})
		if len(imElements) > 0 {
			inputs = make([]*ua.Variant, len(imElements))
			for i := 0; i < len(imElements); i++ {
				inputs[i] = ua.MustVariant(imElements[i].(string))
			}
		}
	}

	request := &ua.CallMethodRequest{
		ObjectID:       oid,
		MethodID:       mid,
		InputArguments: inputs,
	}

	resp, err := s.client.Call(request)
	if err != nil {
		return nil, fmt.Errorf("Driver.handleReadCommands: Method call failed: %s", err)
	}
	if resp.StatusCode != ua.StatusOK {
		return nil, fmt.Errorf("Driver.handleReadCommands: Method status not OK: %v", resp.StatusCode)
	}

	return result.NewResult(req, resp.OutputArguments[0].Value())
}
