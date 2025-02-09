// -*- Mode: Go; indent-tabs-mode: t -*-
//
// Copyright (C) 2021 Schneider Electric
//
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/edgexfoundry/device-opcua-go/internal/test"
	sdkModel "github.com/edgexfoundry/device-sdk-go/v4/pkg/models"
	"github.com/edgexfoundry/go-mod-core-contracts/v4/clients/logger"
	"github.com/edgexfoundry/go-mod-core-contracts/v4/common"
	"github.com/edgexfoundry/go-mod-core-contracts/v4/models"
	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/ua"
	"github.com/spf13/cast"
)

const (
	Protocol string = "opcua"
	Endpoint string = "Endpoint"
)

func TestDriver_ProcessReadCommands(t *testing.T) {
	type args struct {
		deviceName string
		protocols  map[string]models.ProtocolProperties
		reqs       []sdkModel.CommandRequest
	}
	tests := []struct {
		name        string
		args        args
		want        []*sdkModel.CommandValue
		wantErr     bool
		endpointErr bool
		nilClient   bool
	}{
		{
			name: "NOK - no endpoint defined",
			args: args{
				deviceName: "Test",
				protocols:  map[string]models.ProtocolProperties{Protocol: {}},
				reqs:       []sdkModel.CommandRequest{{DeviceResourceName: "TestVar1"}},
			},
			want:        nil,
			wantErr:     true,
			endpointErr: true,
		},
		// non-existent resource will have a nil response and be ignored by edgex
		{
			name: "OK - non-existent variable",
			args: args{
				deviceName: "Test",
				protocols: map[string]models.ProtocolProperties{
					Protocol: {Endpoint: test.Protocol + test.Address},
				},
				reqs: []sdkModel.CommandRequest{{
					DeviceResourceName: "TestVar1",
					Attributes:         map[string]interface{}{NODE: "ns=2;s=fake"},
					Type:               common.ValueTypeInt32,
				}},
			},
			want: make([]*sdkModel.CommandValue, 1),
		},
		{
			name: "NOK - read command - invalid node id",
			args: args{
				deviceName: "Test",
				protocols: map[string]models.ProtocolProperties{
					Protocol: {Endpoint: test.Protocol + test.Address},
				},
				reqs: []sdkModel.CommandRequest{{
					DeviceResourceName: "TestResource1",
					Attributes:         map[string]interface{}{NODE: "ns=2;i=22;z=43"},
					Type:               common.ValueTypeInt32,
				}},
			},
			want:    make([]*sdkModel.CommandValue, 1),
			wantErr: true,
		},
		{
			name: "NOK - not allowed to call method with reader",
			args: args{
				deviceName: "Test",
				protocols: map[string]models.ProtocolProperties{
					Protocol: {Endpoint: test.Protocol + test.Address},
				},
				reqs: []sdkModel.CommandRequest{{
					DeviceResourceName: "SquareResource",
					Attributes:         map[string]interface{}{METHOD: "ns=2;s=square", OBJECT: "ns=2;s=main"},
					Type:               common.ValueTypeInt64,
				}},
			},
			want:    make([]*sdkModel.CommandValue, 1),
			wantErr: true,
		},
		{
			name: "OK - client is nil, but reconnect successful",
			args: args{
				deviceName: "Test",
				protocols: map[string]models.ProtocolProperties{
					Protocol: {Endpoint: test.Protocol + test.Address},
				},
				reqs: []sdkModel.CommandRequest{{
					DeviceResourceName: "TestVar1",
					Attributes:         map[string]interface{}{NODE: "ns=2;s=ro_int32"},
					Type:               common.ValueTypeInt32,
				}},
			},
			want: []*sdkModel.CommandValue{{
				DeviceResourceName: "TestVar1",
				Type:               common.ValueTypeInt32,
				Value:              int32(5),
				Tags:               make(map[string]string),
			}},
			nilClient: true,
		},
		{
			name: "OK - read value from mock server",
			args: args{
				deviceName: "Test",
				protocols: map[string]models.ProtocolProperties{
					Protocol: {Endpoint: test.Protocol + test.Address},
				},
				reqs: []sdkModel.CommandRequest{{
					DeviceResourceName: "TestVar1",
					Attributes:         map[string]interface{}{NODE: "ns=2;s=ro_int32"},
					Type:               common.ValueTypeInt32,
				}},
			},
			want: []*sdkModel.CommandValue{{
				DeviceResourceName: "TestVar1",
				Type:               common.ValueTypeInt32,
				Value:              int32(5),
				Tags:               make(map[string]string),
			}},
			wantErr: false,
		},
		{
			name: "OK - read many values from mock server",
			args: args{
				deviceName: "Test",
				protocols: map[string]models.ProtocolProperties{
					Protocol: {Endpoint: test.Protocol + test.Address},
				},
				reqs: []sdkModel.CommandRequest{{
					DeviceResourceName: "TestVar1",
					Attributes:         map[string]interface{}{NODE: "ns=2;s=ro_int32"},
					Type:               common.ValueTypeInt32,
				}, {
					DeviceResourceName: "TestVar2",
					Attributes:         map[string]interface{}{NODE: "ns=2;s=ro_bool"},
					Type:               common.ValueTypeBool,
				}},
			},
			want: []*sdkModel.CommandValue{{
				DeviceResourceName: "TestVar1",
				Type:               common.ValueTypeInt32,
				Value:              int32(5),
				Tags:               make(map[string]string),
			}, {
				DeviceResourceName: "TestVar2",
				Type:               common.ValueTypeBool,
				Value:              true,
				Tags:               make(map[string]string),
			}},
			wantErr: false,
		},
	}

	server := test.NewServer("../test/opcua_server.py")
	defer server.Close()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// create device client and open connection
			endpoint := cast.ToString(tt.args.protocols[Protocol][Endpoint])
			client, err := opcua.NewClient(endpoint, opcua.SecurityMode(ua.MessageSecurityModeNone))
			if err != nil {
				t.Fatalf("unable to create opcua client %v", err)
			}
			ctx := context.Background()
			defer client.Close(ctx)
			if err := client.Connect(ctx); err != nil {
				if !tt.wantErr || !tt.endpointErr {
					t.Errorf("Unable to connect to server: %v", err)
				}
				return
			}

			dsMock := test.NewDSMock(t)
			s := NewServer(tt.args.deviceName, dsMock)
			if tt.nilClient {
				s.client = nil
				dsMock.On("GetDeviceByName", tt.args.deviceName).Return(models.Device{Name: tt.args.deviceName, Protocols: tt.args.protocols}, nil)
			} else {
				s.client = &Client{client, context.Background()}
			}
			got, err := s.ProcessReadCommands(tt.args.reqs)
			if (err != nil) != tt.wantErr {
				t.Errorf("Driver.HandleReadCommands() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			// Ignore Origin for DeepEqual
			for i := range got {
				if got[i] != nil {
					got[i].Origin = 0
				}
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Driver.HandleReadCommands() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDriver_ProcessReadCommandsNoServer(t *testing.T) {
	type args struct {
		deviceName string
		protocols  map[string]models.ProtocolProperties
		reqs       []sdkModel.CommandRequest
	}
	tests := []struct {
		name        string
		args        args
		want        []*sdkModel.CommandValue
		wantErr     bool
		endpointErr bool
		nilClient   bool
	}{
		{
			name: "NOK - error from nil client",
			args: args{
				deviceName: "Test",
				protocols: map[string]models.ProtocolProperties{
					Protocol: {Endpoint: test.Protocol + test.Address},
				},
				reqs: []sdkModel.CommandRequest{{
					DeviceResourceName: "TestVar1",
					Attributes:         map[string]interface{}{NODE: "ns=2;s=ro_int32"},
					Type:               common.ValueTypeInt32,
				}},
			},
			want:      []*sdkModel.CommandValue{nil},
			wantErr:   true,
			nilClient: true,
		},
		{
			name: "NOK - error from disconnected server",
			args: args{
				deviceName: "Test",
				protocols: map[string]models.ProtocolProperties{
					Protocol: {Endpoint: test.Protocol + test.Address},
				},
				reqs: []sdkModel.CommandRequest{{
					DeviceResourceName: "TestVar1",
					Attributes:         map[string]interface{}{NODE: "ns=2;s=ro_int32"},
					Type:               common.ValueTypeInt32,
				}},
			},
			want:    []*sdkModel.CommandValue{nil},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// create device client and open connection
			endpoint := cast.ToString(tt.args.protocols[Protocol][Endpoint])
			client, err := opcua.NewClient(endpoint, opcua.SecurityMode(ua.MessageSecurityModeNone))
			if err != nil {
				t.Fatalf("unable to create opcua client %v", err)
			}
			ctx := context.Background()
			defer client.Close(ctx)

			dsMock := test.NewDSMock(t)
			s := NewServer(tt.args.deviceName, dsMock)
			if tt.nilClient {
				s.client = nil
			} else {
				s.client = &Client{client, context.Background()}
			}
			dsMock.On("GetDeviceByName", tt.args.deviceName).Return(models.Device{}, fmt.Errorf("error"))

			got, err := s.ProcessReadCommands(tt.args.reqs)
			if (err != nil) != tt.wantErr {
				t.Errorf("Driver.HandleReadCommands() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			// Ignore Origin for DeepEqual
			if len(got) > 0 && got[0] != nil {
				got[0].Origin = 0
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Driver.HandleReadCommands() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBuildReadRequest(t *testing.T) {
	tests := []struct {
		name                    string
		reqNodeIds              []string
		expectedNodeIds         map[string]struct{}
		expectedResultToRequest ResultToRequest
	}{
		{
			name:                    "OK - Empty Request",
			reqNodeIds:              []string{},
			expectedNodeIds:         map[string]struct{}{},
			expectedResultToRequest: map[int][]int{},
		}, {
			name:       "OK - On Read Request",
			reqNodeIds: []string{"ns=1;i=1"},
			expectedNodeIds: map[string]struct{}{
				"ns=1;i=1": {},
			},
			expectedResultToRequest: map[int][]int{
				0: {0},
			},
		}, {
			name: "OK - Multi Read Request",
			reqNodeIds: []string{
				"ns=1;i=1",
				"ns=1;i=2",
				"ns=1;i=3",
				"ns=1;i=4",
				"ns=1;i=5",
			},
			expectedNodeIds: map[string]struct{}{
				"ns=1;i=1": {},
				"ns=1;i=2": {},
				"ns=1;i=3": {},
				"ns=1;i=4": {},
				"ns=1;i=5": {},
			},
			expectedResultToRequest: map[int][]int{
				0: {0},
				1: {1},
				2: {2},
				3: {3},
				4: {4},
			},
		}, {
			name: "OK - Two Overlapping Read Requests",
			reqNodeIds: []string{
				"ns=1;i=1",
				"ns=1;i=1",
			},
			expectedNodeIds: map[string]struct{}{
				"ns=1;i=1": {},
			},
			expectedResultToRequest: map[int][]int{
				0: {0, 1},
			},
		}, {
			name: "OK - Complex Read Requests",
			reqNodeIds: []string{
				"ns=1;i=1",
				"ns=1;i=1",
				"ns=1;i=2",
				"ns=1;i=1",
				"ns=1;i=3",
				"ns=1;i=2",
				"ns=1;i=1",
			},
			expectedNodeIds: map[string]struct{}{
				"ns=1;i=1": {},
				"ns=1;i=2": {},
				"ns=1;i=3": {},
			},
			expectedResultToRequest: map[int][]int{
				0: {0, 1, 3, 6},
				1: {2, 5},
				2: {4},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reqs := make([]sdkModel.CommandRequest, 0, len(tt.reqNodeIds))
			for _, id := range tt.reqNodeIds {
				reqs = append(reqs, sdkModel.CommandRequest{Attributes: map[string]interface{}{NODE: id}})
			}

			nodesToRead, resultToRequest, err := buildNodesToReadRequest(reqs)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			for _, nodes := range nodesToRead {
				id := nodes.NodeID.String()
				if _, ok := tt.expectedNodeIds[id]; !ok {
					t.Fatalf("Node %s not found in request", id)
				} else {
					delete(tt.expectedNodeIds, id)
				}
			}

			if len(tt.expectedNodeIds) > 0 {
				t.Fatalf("Unexpected node in request: %+v", nodesToRead)
			}

			if !reflect.DeepEqual(resultToRequest, tt.expectedResultToRequest) {
				t.Fatalf("Unexpected result to request: expected %+v; got %+v", tt.expectedResultToRequest, resultToRequest)
			}
		})
	}
}

func TestBuildReadRequestOnMethod(t *testing.T) {
	reqs := []sdkModel.CommandRequest{
		{
			Attributes: map[string]interface{}{METHOD: true},
		},
	}
	_, _, err := buildNodesToReadRequest(reqs)

	if err == nil {
		t.Fatalf("Method request should not be allowed")
	}
}

func TestBuildReadRequestOnMissingNodeId(t *testing.T) {
	reqs := []sdkModel.CommandRequest{
		{
			Attributes: map[string]interface{}{METHOD: false},
		},
	}
	_, _, err := buildNodesToReadRequest(reqs)

	if err == nil {
		t.Fatalf("Node Id is missing from properties; error expected")
	}
}

func TestBuildCommandValues(t *testing.T) {
	reqs := []sdkModel.CommandRequest{
		{
			DeviceResourceName: "Res1",
			Type:               common.ValueTypeInt32,
		},
		{
			DeviceResourceName: "Res2",
			Type:               common.ValueTypeInt32,
		},
		{
			DeviceResourceName: "Res3",
			Type:               common.ValueTypeInt32,
		},
	}

	lc := logger.NewMockClient()

	t.Run("Read on one node", func(t *testing.T) {
		var resultToRequest ResultToRequest = map[int][]int{0: {0, 1, 2}}

		uaResponse := &ua.ReadResponse{
			Results: []*ua.DataValue{
				{
					Value: ua.MustVariant(int32(1)),
				},
			},
		}

		commandValues := resultToRequest.buildCommandValues(reqs, uaResponse, lc)

		if len(commandValues) != 3 {
			t.Fatalf("Expected number of command values 3; got %d;", len(commandValues))
		}

		if commandValues[0].DeviceResourceName != "Res1" {
			t.Fatalf("Expected device resource name [0] Res1; got %s", commandValues[0].DeviceResourceName)
		}

		if commandValues[1].DeviceResourceName != "Res2" {
			t.Fatalf("Expected device resource name [1] Res2; got %s", commandValues[1].DeviceResourceName)
		}

		if commandValues[2].DeviceResourceName != "Res3" {
			t.Fatalf("Expected device resource name [2] Res3; got %s", commandValues[2].DeviceResourceName)
		}

		if commandValues[0].Value != int32(1) {
			t.Fatalf("Expected device resource value [0] 1; got %v", commandValues[0].Value)
		}

		if commandValues[1].Value != int32(1) {
			t.Fatalf("Expected device resource value [1] 1; got %v", commandValues[1].Value)
		}

		if commandValues[2].Value != int32(1) {
			t.Fatalf("Expected device resource value [2] 1; got %v", commandValues[2].Value)
		}

	})

	t.Run("Read on multiple nodes", func(t *testing.T) {
		var resultToRequest ResultToRequest = map[int][]int{0: {0}, 1: {1}, 2: {2}}

		uaResponse := &ua.ReadResponse{
			Results: []*ua.DataValue{
				{
					Value: ua.MustVariant(int32(1)),
				}, {
					Value: ua.MustVariant(int32(2)),
				}, {
					Value: ua.MustVariant(int32(3)),
				},
			},
		}

		commandValues := resultToRequest.buildCommandValues(reqs, uaResponse, lc)

		if len(commandValues) != 3 {
			t.Fatalf("Expected number of command values 3; got %d;", len(commandValues))
		}

		if commandValues[0].DeviceResourceName != "Res1" {
			t.Fatalf("Expected device resource name [0] Res1; got %s", commandValues[0].DeviceResourceName)
		}

		if commandValues[1].DeviceResourceName != "Res2" {
			t.Fatalf("Expected device resource name [1] Res2; got %s", commandValues[1].DeviceResourceName)
		}

		if commandValues[2].DeviceResourceName != "Res3" {
			t.Fatalf("Expected device resource name [2] Res3; got %s", commandValues[2].DeviceResourceName)
		}

		if commandValues[0].Value != int32(1) {
			t.Fatalf("Expected device resource value [0] 1; got %v", commandValues[0].Value)
		}

		if commandValues[1].Value != int32(2) {
			t.Fatalf("Expected device resource value [1] 2; got %v", commandValues[1].Value)
		}

		if commandValues[2].Value != int32(3) {
			t.Fatalf("Expected device resource value [2] 3; got %v", commandValues[2].Value)
		}

	})
}
