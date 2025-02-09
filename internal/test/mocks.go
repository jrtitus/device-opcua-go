// -*- Mode: Go; indent-tabs-mode: t -*-
//
// Copyright (C) 2021 Schneider Electric
//
// SPDX-License-Identifier: Apache-2.0

package test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/edgexfoundry/device-sdk-go/v4/pkg/interfaces/mocks"
	"github.com/edgexfoundry/device-sdk-go/v4/pkg/models"
	"github.com/edgexfoundry/go-mod-core-contracts/v4/clients/logger"
	"github.com/stretchr/testify/mock"
)

func NewDSMock(t *testing.T) *mocks.DeviceServiceSDK {
	dsMock := mocks.NewDeviceServiceSDK(t)
	logMock := logger.NewMockClient()
	dsMock.On("LoggingClient").Return(logMock).Maybe()
	dsMock.On("AsyncValuesChannel").Return(make(chan *models.AsyncValues, 1)).Maybe()

	return dsMock
}

type ResponseWriterMock struct {
	mock.Mock
}

func (r *ResponseWriterMock) Write([]byte) (int, error) {
	return 0, nil
}

func (r *ResponseWriterMock) Header() http.Header {
	return make(http.Header)
}

func (r *ResponseWriterMock) WriteHeader(statusCode int) {
	fmt.Printf("StatusCode=%d", statusCode)
}
