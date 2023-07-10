// -*- Mode: Go; indent-tabs-mode: t -*-
//
// Copyright (C) 2022 Schneider Electric
//
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"context"
	"sync"

	"github.com/edgexfoundry/device-sdk-go/v3/pkg/interfaces"
	"github.com/gopcua/opcua"
)

type CancelContext struct {
	ctx    context.Context
	cancel context.CancelFunc
}

type Client struct {
	*opcua.Client
	ctx context.Context
}

type Server struct {
	deviceName  string
	resourceMap map[uint32]string
	context     *CancelContext
	client      *Client
	sdk         interfaces.DeviceServiceSDK
	mu          sync.Mutex
}

func NewServer(deviceName string, sdk interfaces.DeviceServiceSDK) *Server {
	server := &Server{
		deviceName:  deviceName,
		resourceMap: make(map[uint32]string),
		sdk:         sdk,
	}
	server.newContext()
	return server
}

func (s *Server) Cleanup(recreateContext bool) {
	if s.context != nil {
		s.context.cancel()
		s.context = nil
	}
	if recreateContext {
		s.newContext()
	}
}

func (s *Server) newContext() {
	ctxbg := context.Background()
	ctx, cancel := context.WithCancel(ctxbg)

	s.context = &CancelContext{
		ctx:    ctx,
		cancel: cancel,
	}
}
