package client

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"sync"
	"testing"
)

func TestConnectionPool_clearUnusedConnections(t *testing.T) {
	type fields struct {
		idleConnections []*LXDClient
		usedConnections []*LXDClient
		config          PoolConfig
	}
	tests := []struct {
		name   string
		fields fields
		want   []*LXDClient
	}{
		{
			name: "clear unused 2 connections",
			fields: fields{
				idleConnections: []*LXDClient{
					{id: 1},
					{id: 2},
					{id: 3},
					{id: 4},
					{id: 5},
				},
				usedConnections: nil,
				config: PoolConfig{
					MaxPoolSize:        5,
					MaxIdleConnections: 3,
				},
			},
			want: []*LXDClient{
				{id: 1},
				{id: 2},
				{id: 3},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &ConnectionPool{
				idleConnections: tt.fields.idleConnections,
				usedConnections: tt.fields.usedConnections,
				mutex:           sync.Mutex{},
				config:          tt.fields.config,
			}
			c.clearUnusedConnections()

			assert.Equal(t, tt.want, c.idleConnections)
		})
	}
}

func TestConnectionPool_Get(t *testing.T) {
	type fields struct {
		idleConnections []*LXDClient
		usedConnections []*LXDClient
		config          PoolConfig
	}
	tests := []struct {
		name          string
		fields        fields
		want          *LXDClient
		wantIdleConns []*LXDClient
		wantUsedConns []*LXDClient
		wantErr       assert.ErrorAssertionFunc
	}{
		{
			name: "get client",
			fields: fields{
				idleConnections: []*LXDClient{
					{id: 1},
					{id: 2},
					{id: 3},
					{id: 4},
					{id: 5},
				},
				usedConnections: nil,
				config: PoolConfig{
					MaxPoolSize:        5,
					MaxIdleConnections: 3,
				},
			},
			wantIdleConns: []*LXDClient{
				{id: 2},
				{id: 3},
				{id: 4},
				{id: 5},
			},
			wantUsedConns: []*LXDClient{
				{id: 1},
			},
			want:    &LXDClient{id: 1},
			wantErr: assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &ConnectionPool{
				idleConnections: tt.fields.idleConnections,
				usedConnections: tt.fields.usedConnections,
				mutex:           sync.Mutex{},
				config:          tt.fields.config,
			}
			got, err := c.Get()
			if !tt.wantErr(t, err, fmt.Sprintf("Get()")) {
				return
			}
			assert.Equalf(t, tt.want, got, "Get()")
			assert.Equalf(t, tt.wantIdleConns, c.idleConnections, "Idle Connections")
			assert.Equalf(t, tt.wantUsedConns, c.usedConnections, "Used Connections")
		})
	}
}
