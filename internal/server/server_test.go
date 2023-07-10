// -*- Mode: Go; indent-tabs-mode: t -*-
//
// Copyright (C) 2022 Schneider Electric
//
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"testing"

	"github.com/edgexfoundry/device-sdk-go/v3/pkg/interfaces/mocks"
)

func TestNewServer(t *testing.T) {
	t.Run("create server", func(t *testing.T) {

		s := NewServer("test", mocks.NewDeviceServiceSDK(t))
		if s == nil {
			t.Error("NewServer() failed")
		}
		s.Cleanup(false)
	})
}

func TestServer_Cleanup(t *testing.T) {
	tests := []struct {
		name             string
		recreateContext  bool
		startWithContext bool
	}{
		{
			name:             "Mock update",
			recreateContext:  true,
			startWithContext: true,
		},
		{
			name:             "Mock delete",
			recreateContext:  false,
			startWithContext: true,
		},
		{
			name:             "Start with no context",
			startWithContext: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Server{}
			if tt.startWithContext {
				s.newContext()
			}
			s.Cleanup(tt.recreateContext)
		})
	}
}
