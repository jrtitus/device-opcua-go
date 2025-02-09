// -*- Mode: Go; indent-tabs-mode: t -*-
//
// Copyright (C) 2022 Schneider Electric
//
// SPDX-License-Identifier: Apache-2.0

package driver

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/edgexfoundry/go-mod-core-contracts/v4/dtos/common"
	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	"github.com/spf13/cast"
)

var validate *validator.Validate

type MethodRequest struct {
	DeviceName string   `json:"device" validate:"required"`
	MethodName string   `json:"method" validate:"required"`
	Parameters []string `json:"parameters,omitempty"`
}

func (r *MethodRequest) validate() error {
	if validate == nil {
		validate = validator.New()
	}

	return validate.Struct(r)
}

func handleMethodCall(e echo.Context) error {
	w := e.Response()
	r := e.Request()
	w.Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSON)

	id := r.Header.Get("X-Correlation-ID")

	if r.Body == nil {
		return echo.NewHTTPError(http.StatusBadRequest, "request body required")
	}
	defer r.Body.Close()

	var req MethodRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		driver.sdk.LoggingClient().Errorf("invalid request: %v", err)
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}

	if err := req.validate(); err != nil {
		msg := fmt.Sprintf("invalid request: %v", err)
		driver.sdk.LoggingClient().Error(msg)
		return echo.NewHTTPError(http.StatusBadRequest, msg)
	}

	// get device from server map
	server, ok := driver.serverMap[req.DeviceName]
	if !ok {
		return echo.NewHTTPError(http.StatusInternalServerError, "error interacting with device")
	}

	// call to method with parameters - see methodhandler
	response, err := server.ProcessMethodCall(req.MethodName, req.Parameters)
	if err != nil {
		driver.sdk.LoggingClient().Errorf(err.Error())
		return echo.NewHTTPError(http.StatusInternalServerError, "error interacting with device")
	}

	baseResponse := common.NewBaseResponse(id, cast.ToString(response), http.StatusOK)
	return e.JSON(http.StatusOK, baseResponse)
}
