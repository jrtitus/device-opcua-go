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
	sdkModel "github.com/edgexfoundry/device-sdk-go/v4/pkg/models"
	"github.com/edgexfoundry/go-mod-core-contracts/v4/clients/logger"
	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/ua"
)

type ResultToRequest map[int][]int

func createResult(req sdkModel.CommandRequest, variant *ua.Variant, logger logger.LoggingClient) (response *sdkModel.CommandValue) {
	var err error
	if response, err = result.NewResult(req, variant.Value()); err != nil {
		logger.Errorf("Driver.handleReadCommands: Error: %v", err)
	}
	return response
}

func (rr ResultToRequest) buildCommandValues(reqs []sdkModel.CommandRequest, resp *ua.ReadResponse, logger logger.LoggingClient) []*sdkModel.CommandValue {
	responses := make([]*sdkModel.CommandValue, len(reqs))
	for i := 0; i < len(resp.Results); i++ {
		if resp.Results[i].Status != ua.StatusOK {
			logger.Debugf("Driver.handleReadCommands: Status not OK: %v", resp.Results[i].Status)
			continue
		}

		variant := resp.Results[i].Value
		if variant == nil || variant.Value() == nil {
			continue
		}

		if reqIndexes, ok := rr[i]; ok {
			for _, reqIndex := range reqIndexes {
				responses[reqIndex] = createResult(reqs[reqIndex], variant, logger)
			}
		}
	}

	return responses
}

func buildNodesToReadRequest(reqs []sdkModel.CommandRequest) (nodesToRead []*ua.ReadValueID, resultToRequest ResultToRequest, err error) {
	nodesToRead = make([]*ua.ReadValueID, 0, len(reqs))
	resultToRequest = make(map[int][]int, len(reqs))
	nodesIdToResultIndex := make(map[string]int, len(reqs))

	for reqIndex, req := range reqs {
		if _, isMethod := req.Attributes[METHOD]; isMethod {
			return nil, nil, fmt.Errorf("not allowed to call command on method: %s", req.DeviceResourceName)
		}

		id, err := getNodeID(req.Attributes, NODE)
		if err != nil {
			return nil, nil, fmt.Errorf("Driver.handleReadCommands: Invalid node id = %v", err)
		}

		if resultIndex, ok := nodesIdToResultIndex[id.String()]; ok {
			resultToRequest[resultIndex] = append(resultToRequest[resultIndex], reqIndex)
		} else {
			nodesToRead = append(nodesToRead, &ua.ReadValueID{NodeID: id})
			resultIndex = len(nodesToRead) - 1
			nodesIdToResultIndex[id.String()] = resultIndex
			resultToRequest[resultIndex] = []int{reqIndex}
		}
	}

	return nodesToRead, resultToRequest, nil
}

func (s *Server) ProcessReadCommands(reqs []sdkModel.CommandRequest) (responses []*sdkModel.CommandValue, err error) {
	responses = make([]*sdkModel.CommandValue, len(reqs))

	nodesToRead, resultToRequest, err := buildNodesToReadRequest(reqs)
	if err != nil {
		s.sdk.LoggingClient().Error(err.Error())
		return responses, err
	}

	request := &ua.ReadRequest{
		MaxAge:             2000,
		NodesToRead:        nodesToRead,
		TimestampsToReturn: ua.TimestampsToReturnBoth,
	}

	if len(request.NodesToRead) > 0 {
		if s.client == nil || s.client.State() == opcua.Closed || s.client.State() == opcua.Disconnected {
			if err := s.Connect(); err != nil {
				s.sdk.LoggingClient().Errorf("Driver.handleReadCommands: client not initialized: %v", err)
				return responses, err
			}
		}

		resp, err := s.client.Read(s.client.ctx, request)
		if err != nil {
			s.sdk.LoggingClient().Errorf("Driver.HandleReadCommands: Handle read commands failed: %v", err)
			return responses, err
		}

		responses = resultToRequest.buildCommandValues(reqs, resp, s.sdk.LoggingClient())
	}

	return responses, nil
}
