// -*- Mode: Go; indent-tabs-mode: t -*-
//
// Copyright (C) 2022 Schneider Electric
//
// SPDX-License-Identifier: Apache-2.0

package driver

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/edgexfoundry/device-opcua-go/internal/server"
	"github.com/stretchr/testify/mock"
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

type responseWriterMock struct {
	mock.Mock
}

func (r *responseWriterMock) Write([]byte) (int, error) {
	return 0, nil
}

func (r *responseWriterMock) Header() http.Header {
	return make(http.Header)
}

func (r *responseWriterMock) WriteHeader(statusCode int) {
	fmt.Printf("StatusCode=%d", statusCode)
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
	}{
		{
			name: "NOK - no body",
			args: args{w: new(responseWriterMock), body: nil},
		},
		{
			name: "NOK - invalid body",
			args: args{w: new(responseWriterMock), body: bytes.NewBufferString("")},
		},
		{
			name: "NOK - invalid request",
			args: args{w: new(responseWriterMock), body: bytes.NewBufferString("{}")},
		},
		{
			name: "NOK - device not found",
			args: args{w: new(responseWriterMock), body: bytes.NewBufferString(`{"device":"test","method":"test"}`)},
		},
		{
			name:       "NOK - cannot make method call in unit test",
			args:       args{w: new(responseWriterMock), body: bytes.NewBufferString(`{"device":"test","method":"test"}`)},
			deviceName: "test",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := newMockDriver()
			if tt.deviceName != "" {
				d.serverMap[tt.deviceName] = new(server.Server)
			}
			request, _ := http.NewRequest(http.MethodPost, "", tt.args.body)
			handleMethodCall(tt.args.w, request)
		})
	}
}
