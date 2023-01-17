package pkg

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetNameFromModule(t *testing.T) {
	type params struct {
		basepath    string
		module      string
		packageName string
		version     string
	}
	type expected struct {
		err         bool
		packageName string
	}
	modCachDir := os.Getenv("GOMODCACHE")
	tests := []struct {
		name     string
		params   params
		expected expected
	}{
		{
			name: "should read a package name from a module",
			params: params{
				basepath:    modCachDir,
				module:      "github.com/stretchr/testify",
				version:     "v1.8.1",
				packageName: "assert",
			},
			expected: expected{
				err:         false,
				packageName: "assert",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			packageName, err := GetNameFromModule(tt.params.basepath, tt.params.module, tt.params.version, tt.params.packageName)
			assert.Equal(t, tt.expected.err, err != nil)
			if err == nil {
				assert.Equal(t, tt.expected.packageName, packageName)
			}
		})
	}
}
