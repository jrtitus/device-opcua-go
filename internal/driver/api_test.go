// -*- Mode: Go; indent-tabs-mode: t -*-
//
// Copyright (C) 2022 Schneider Electric
//
// SPDX-License-Identifier: Apache-2.0

package driver

import (
	"bytes"
	"io"
	"net/http"
	"testing"

	"github.com/edgexfoundry/device-opcua-go/internal/server"
	"github.com/edgexfoundry/device-opcua-go/internal/test"
	"github.com/edgexfoundry/go-mod-core-contracts/v3/models"
)

func TestMethodRequest_validate(t *testing.T) {
	type fields struct {
		DeviceName string
		MethodName string
		Parameters []string
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name:    "NOK - missing device name",
			fields:  fields{MethodName: "Method"},
			wantErr: true,
		},
		{
			name:    "NOK - missing method name",
			fields:  fields{DeviceName: "Device"},
			wantErr: true,
		},
		{
			name:   "OK",
			fields: fields{DeviceName: "Device", MethodName: "Method"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &MethodRequest{
				DeviceName: tt.fields.DeviceName,
				MethodName: tt.fields.MethodName,
				Parameters: tt.fields.Parameters,
			}
			if err := r.validate(); (err != nil) != tt.wantErr {
				t.Errorf("MethodRequest.validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_handleMethodCall(t *testing.T) {
	type args struct {
		w    http.ResponseWriter
		body io.Reader
	}
	tests := []struct {
		name       string
		args       args
		deviceName string
		methodName string
		resource   models.DeviceResource
	}{
		{
			name: "NOK - no body",
			args: args{w: new(test.ResponseWriterMock), body: nil},
		},
		{
			name: "NOK - invalid body",
			args: args{w: new(test.ResponseWriterMock), body: bytes.NewBufferString("")},
		},
		{
			name: "NOK - invalid request",
			args: args{w: new(test.ResponseWriterMock), body: bytes.NewBufferString("{}")},
		},
		{
			name: "NOK - device not found",
			args: args{w: new(test.ResponseWriterMock), body: bytes.NewBufferString(`{"device":"test","method":"test"}`)},
		},
		{
			name:       "NOK - hidden resource",
			args:       args{w: new(test.ResponseWriterMock), body: bytes.NewBufferString(`{"device":"test","method":"test"}`)},
			deviceName: "test",
			methodName: "test",
			resource:   models.DeviceResource{Name: "test", IsHidden: true},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d, dsMock := newMockDriver(t)
			if tt.deviceName != "" {
				d.serverMap[tt.deviceName] = server.NewServer(tt.deviceName, dsMock)

				dsMock.On("DeviceResource", tt.deviceName, tt.methodName).Return(tt.resource, true)
			}
			request, _ := http.NewRequest(http.MethodPost, "", tt.args.body)
			handleMethodCall(tt.args.w, request)
		})
	}
}
