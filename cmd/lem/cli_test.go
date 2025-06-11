package main

import (
	"context"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_cli(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		isError bool
	}{
		{
			name:    "run basic",
			args:    []string{"lem", "run", "--config", "testdata/1/lem.toml", "--stage", "default"},
			isError: false,
		},
		{
			name:    "run config is empty",
			args:    []string{"lem", "run", "--config", "testdata/1/lem.empty.toml", "--stage", "default"},
			isError: true,
		},
		{
			name:    "run config is invalid",
			args:    []string{"lem", "run", "--config", "testdata/1/lem.invalid.toml", "--stage", "default"},
			isError: true,
		},
		{
			name:    "run stage not found",
			args:    []string{"lem", "run", "--config", "testdata/1/lem.toml", "--stage", "dummy"},
			isError: true,
		},
		{
			name:    "watch config is empty",
			args:    []string{"lem", "watch", "--config", "testdata/1/lem.empty.toml", "--stage", "default"},
			isError: true,
		},
		{
			name:    "watch config is invalid",
			args:    []string{"lem", "watch", "--config", "testdata/1/lem.invalid.toml", "--stage", "default"},
			isError: true,
		},
		{
			name:    "watch stage not found",
			args:    []string{"lem", "watch", "--config", "testdata/1/lem.toml", "--stage", "dummy"},
			isError: true,
		},
		{
			name:    "validate basic",
			args:    []string{"lem", "validate", "--config", "testdata/1/lem.toml"},
			isError: false,
		},
		{
			name:    "validate config is empty",
			args:    []string{"lem", "validate", "--config", "testdata/1/lem.empty.toml"},
			isError: true,
		},
		{
			name:    "validate config is invalid",
			args:    []string{"lem", "validate", "--config", "testdata/1/lem.invalid.toml"},
			isError: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := newApp(io.Discard, io.Discard).Run(context.Background(), tt.args)
			if tt.isError {
				assert.Error(t, err)
			}
		})
	}
}
