// -*- Mode: Go; indent-tabs-mode: t -*-
//
// Copyright (C) 2022 Schneider Electric
//
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"context"
	"fmt"
	"sync"

	"github.com/edgexfoundry/device-sdk-go/v4/pkg/interfaces"
	"github.com/edgexfoundry/go-mod-core-contracts/v4/models"
	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/ua"
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
	config      *Config
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

func (s *Server) Connect() error {
	device, err := s.sdk.GetDeviceByName(s.deviceName)
	if err != nil {
		return err
	}

	if device.AdminState == models.Locked || device.OperatingState == models.Down {
		return fmt.Errorf("client not started for [%s]: device is locked or down", s.deviceName)
	}

	serverConfig, err := NewConfig(device.Protocols["opcua"])
	if err != nil {
		return err
	}
	s.mu.Lock()
	s.config = serverConfig
	s.mu.Unlock()

	if err := s.initClient(); err != nil {
		return err
	}

	if err := s.client.Connect(s.client.ctx); err != nil {
		s.sdk.LoggingClient().Warnf("[%s] failed to connect OPCUA client: %v", s.deviceName, err)
		return err
	}

	return nil
}

func (s *Server) Cleanup(recreateContext bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.client != nil {
		// Connection could have been opened from
		// subscriptionlistener, readhandler, writehandler, or methodhandler
		if err := s.client.Close(s.client.ctx); err != nil {
			s.sdk.LoggingClient().Warnf("[%s] failed to close OPCUA client: %v", s.deviceName, err)
		}
		s.client = nil
	}
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

func (s *Server) initClient() error {

	endpoints, err := opcua.GetEndpoints(s.context.ctx, s.config.Endpoint)
	if err != nil {
		return err
	}

	ep, err := opcua.SelectEndpoint(endpoints, s.config.Policy, ua.MessageSecurityModeFromString(s.config.Mode))
	if err != nil {
		s.sdk.LoggingClient().Error(err.Error())
		return fmt.Errorf("[%s] failed to find suitable endpoint", s.deviceName)
	}
	ep.EndpointURL = s.config.Endpoint

	opts := []opcua.Option{
		opcua.SecurityPolicy(s.config.Policy),
		opcua.SecurityModeString(s.config.Mode),
		opcua.CertificateFile(s.config.CertFile),
		opcua.PrivateKeyFile(s.config.KeyFile),
		opcua.AuthAnonymous(),
		opcua.SecurityFromEndpoint(ep, ua.UserTokenTypeAnonymous),
	}

	uaClient, err := opcua.NewClient(ep.EndpointURL, opts...)
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.client = &Client{
		uaClient,
		context.Background(),
	}

	return nil
}
