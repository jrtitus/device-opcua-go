// -*- Mode: Go; indent-tabs-mode: t -*-
//
// Copyright (C) 2022 Schneider Electric
//
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"encoding/json"

	"github.com/edgexfoundry/go-mod-core-contracts/v4/models"
	"github.com/go-playground/validator/v10"
)

// Config struct details for OPCUA device list protocol properties
type Config struct {
	Endpoint  string   `json:"Endpoint" validate:"required"`
	Policy    string   `json:"Policy" validate:"oneof=None Basic128Rsa15 Basic256 Basic256Sha256 Aes128Sha256RsaOaep Aes256Sha256RsaPss"`
	Mode      string   `json:"Mode" validate:"oneof=None Sign SignAndEncrypt"`
	CertFile  string   `json:"CertFile" validate:"required_unless=Policy None Mode None"`
	KeyFile   string   `json:"KeyFile" validate:"required_unless=Policy None Mode None"`
	Resources []string `json:"Resources"`
}

// NewConfig converts a properties map to a Config struct
func NewConfig(props models.ProtocolProperties) (*Config, error) {
	var c *Config
	bytes, err := json.Marshal(props)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(bytes, &c); err != nil {
		return nil, err
	}

	return c, nil
}

// Validate makes sure the connection properties are valid
func Validate(cfg *Config) error {
	validate := validator.New()
	return validate.Struct(cfg)
}
