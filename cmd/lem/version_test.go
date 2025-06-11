package main

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_version(t *testing.T) {
	tests := []struct {
		name     string
		version  string
		revision string
		expected string
	}{
		{
			name:     "basic",
			revision: "1234567",
			expected: fmt.Sprintf("%s (revision: 1234567)", version),
		},
		{
			name:     "no revision",
			version:  version,
			revision: "",
			expected: version,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			revision = tt.revision
			actual := getVersion()
			assert.Equal(t, tt.expected, actual)
		})
	}
}
