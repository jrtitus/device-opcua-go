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
	"github.com/edgexfoundry/go-mod-core-contracts/v2/clients/logger"
	"github.com/edgexfoundry/go-mod-core-contracts/v2/models"
	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/ua"
)

func TestDriver_makeMethodCall(t *testing.T) {
	type args struct {
		resource   models.DeviceResource
		parameters []string
	}
	tests := []struct {
		name    string
		args    args
		want    interface{}
		wantErr bool
	}{
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
			defer client.Close()
			if err := client.Connect(context.Background()); err != nil {
				if !tt.wantErr {
					t.Errorf("Unable to connect to server: %v", err)
				}
				return
			}

			s := &Server{
				logger: &logger.MockLogger{},
				client: &Client{
					client,
					context.Background(),
				},
			}
			got, err := s.makeMethodCall(tt.args.resource, tt.args.parameters)
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
