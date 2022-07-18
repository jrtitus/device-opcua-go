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

	"github.com/edgexfoundry/go-mod-core-contracts/v2/dtos/common"
	"github.com/go-playground/validator/v10"
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

func writeResponse(w http.ResponseWriter, id, message string, status int) {
	response := common.NewBaseResponse(id, message, status)
	bytes, _ := json.Marshal(response)
	_, _ = w.Write(bytes) // nolint:errcheck
}

func handleMethodCall(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")

	id := r.Header.Get("X-Correlation-ID")

	if r.Body == nil {
		writeResponse(w, id, "request body required", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var req MethodRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		driver.Logger.Errorf("invalid request: %v", err)
		writeResponse(w, id, "invalid request", http.StatusBadRequest)
		return
	}

	if err := req.validate(); err != nil {
		msg := fmt.Sprintf("invalid request: %v", err)
		driver.Logger.Error(msg)
		writeResponse(w, id, msg, http.StatusBadRequest)
		return
	}

	// get device from server map
	server, ok := driver.serverMap[req.DeviceName]
	if !ok {
		writeResponse(w, id, "error interacting with device", http.StatusInternalServerError)
		return
	}

	// call to method with parameters - see methodhandler
	response, err := server.ProcessMethodCall(req.MethodName, req.Parameters)
	if err != nil {
		driver.Logger.Errorf(err.Error())
		writeResponse(w, id, "error interacting with device", http.StatusInternalServerError)
		return
	}

	writeResponse(w, id, cast.ToString(response), http.StatusOK)
}
