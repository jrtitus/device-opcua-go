// -*- Mode: Go; indent-tabs-mode: t -*-
//
// Copyright (C) 2022 Schneider Electric
//
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"reflect"
	"testing"

	"github.com/edgexfoundry/go-mod-core-contracts/v4/models"
)

func TestNewConfig(t *testing.T) {
	tests := []struct {
		name    string
		props   models.ProtocolProperties
		want    *Config
		wantErr bool
	}{
		{
			name: "OK - endpoint and resources",
			props: models.ProtocolProperties{
				Endpoint: "opc.tcp://test", "Policy": "None", "Mode": "None", "Resources": []string{"A", "B", "C"}},
			want: &Config{
				Endpoint: "opc.tcp://test", Policy: "None", Mode: "None", Resources: []string{"A", "B", "C"}, CertFile: "", KeyFile: ""},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewConfig(tt.props)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewConfig() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *Config
		wantErr bool
	}{
		{
			name:    "NOK - nil props",
			wantErr: true,
		},
		{
			name:    "NOK - missing Endpoint",
			cfg:     &Config{},
			wantErr: true,
		},
		{
			name: "NOK - invalid policy or mode",
			cfg: &Config{
				Endpoint: "opc.tcp://test",
			},
			wantErr: true,
		},
		{
			name: "NOK - missing certfile or keyfile",
			cfg: &Config{
				Endpoint: "opc.tcp://test",
				Policy:   "Basic256",
				Mode:     "Sign",
			},
			wantErr: true,
		},
		{
			name: "OK - endpoint and resources",
			cfg: &Config{
				Endpoint: "opc.tcp://test", Policy: "None", Mode: "None", Resources: []string{"A", "B", "C"}, CertFile: "", KeyFile: ""},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := Validate(tt.cfg); (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
