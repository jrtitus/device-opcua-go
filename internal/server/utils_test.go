// -*- Mode: Go; indent-tabs-mode: t -*-
//
// Copyright (C) 2021 Schneider Electric
//
// SPDX-License-Identifier: Apache-2.0

package server

import "testing"

func Test_getNodeID(t *testing.T) {
	type args struct {
		attrs map[string]interface{}
		id    string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name:    "NOK - key does not exist",
			args:    args{attrs: map[string]interface{}{NODE: "ns=2"}, id: "fail"},
			want:    "",
			wantErr: true,
		},
		{
			name:    "OK - node id returned",
			args:    args{attrs: map[string]interface{}{NODE: "ns=2;s=edgex/int32/var0"}, id: NODE},
			want:    "ns=2;s=edgex/int32/var0",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getNodeID(tt.args.attrs, tt.args.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("buildNodeID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != nil && got.String() != tt.want {
				t.Errorf("buildNodeID() = %v, want %v", got, tt.want)
			}
		})
	}
}
