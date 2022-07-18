// -*- Mode: Go; indent-tabs-mode: t -*-
//
// Copyright (C) 2021 Schneider Electric
//
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"fmt"

	"github.com/gopcua/opcua/ua"
)

const (
	NODE     string = "nodeId"
	OBJECT   string = "objectId"
	METHOD   string = "methodId"
	INPUTMAP string = "inputMap"
)

func getNodeID(attrs map[string]interface{}, id string) (*ua.NodeID, error) {
	identifier, ok := attrs[id]
	if !ok {
		return nil, fmt.Errorf("attribute %s does not exist", id)
	}

	return ua.ParseNodeID(identifier.(string))
}
