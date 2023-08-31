// -*- Mode: Go; indent-tabs-mode: t -*-
//
// Copyright (C) 2021 Schneider Electric
//
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"context"
	"reflect"
	"testing"

	"github.com/edgexfoundry/device-opcua-go/internal/test"
	"github.com/edgexfoundry/go-mod-core-contracts/v3/models"
	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/ua"
	"github.com/stretchr/testify/mock"
)

func TestDriver_ProcessMethodCall(t *testing.T) {
	type args struct {
		resource   models.DeviceResource
		method     string
		parameters []string
	}
	tests := []struct {
		name    string
		args    args
		want    interface{}
		wantErr bool
	}{
		{
			name: "NOK - method call - method not found",
			args: args{
				method: "TestResource0",
			},
			wantErr: true,
		},
		{
			name: "NOK - method call - method is hidden",
			args: args{
				resource: models.DeviceResource{Name: "TestResource1", IsHidden: true},
			},
			wantErr: true,
		},
		{
			name: "NOK - method call - invalid object node id",
			args: args{
				resource: models.DeviceResource{
					Name:       "TestResource1",
					Attributes: map[string]interface{}{METHOD: "ns=2;s=test"},
				},
			},
			wantErr: true,
		},
		{
			name: "NOK - method call - invalid method node id",
			args: args{
				resource: models.DeviceResource{
					Name:       "TestResource1",
					Attributes: map[string]interface{}{OBJECT: "ns=2;s=main"},
				},
			},
			wantErr: true,
		},
		{
			name: "NOK - method call - method does not exist",
			args: args{
				resource: models.DeviceResource{
					Name: "TestResource1",
					Attributes: map[string]interface{}{
						METHOD: "ns=2;s=test",
						OBJECT: "ns=2;s=main",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "OK - call method from mock server",
			args: args{
				resource: models.DeviceResource{
					Name: "TestResource1",
					Attributes: map[string]interface{}{
						METHOD: "ns=2;s=square",
						OBJECT: "ns=2;s=main",
					},
				},
				parameters: []string{"2"},
			},
			want: "4",
		},
	}

	server := test.NewServer("../test/opcua_server.py")
	defer server.Close()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// create device client and open connection
			endpoint := test.Protocol + test.Address
			client := opcua.NewClient(endpoint, opcua.SecurityMode(ua.MessageSecurityModeNone))
			ctx := context.Background()
			defer client.Close(ctx)
			if err := client.Connect(ctx); err != nil {
				if !tt.wantErr {
					t.Errorf("Unable to connect to server: %v", err)
				}
				return
			}

			dsMock := test.NewDSMock(t)
			s := NewServer("test", dsMock)
			s.client = &Client{client, context.Background()}
			dsMock.On("DeviceResource", mock.Anything, tt.args.method).Return(tt.args.resource, tt.args.resource.Name != "")
			got, err := s.ProcessMethodCall(tt.args.method, tt.args.parameters)
			if (err != nil) != tt.wantErr {
				t.Errorf("Driver.HandleReadCommands() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Driver.HandleReadCommands() = %v, want %v", got, tt.want)
			}
		})
	}
}
