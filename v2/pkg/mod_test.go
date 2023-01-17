package pkg

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReadMod(t *testing.T) {
	type params struct {
		file string
	}
	type expected struct {
		err        bool
		modulePath string
	}
	tests := []struct {
		name     string
		params   params
		expected expected
	}{
		{
			name: "should read mod file",
			params: params{
				file: "../../go.mod",
			},
			expected: expected{
				err:        false,
				modulePath: "github.com/javiercbk/swago",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			modFile, err := ReadMod(tt.params.file)
			assert.Equal(t, tt.expected.err, err != nil)
			if err == nil {
				assert.Equal(t, tt.expected.modulePath, modFile.Name)
			}
		})
	}
}

func TestSplitModulePackage(t *testing.T) {
	type fields struct {
		require map[string]string
	}
	type params struct {
		url string
	}
	type expected struct {
		module string
		pkg    string
	}
	tests := []struct {
		name     string
		fields   fields
		params   params
		expected expected
	}{
		{
			name:   "should return empty string if module not found",
			fields: fields{},
			params: params{
				url: "github.com/stretchr/testify/assert",
			},
			expected: expected{
				module: "",
				pkg:    "",
			},
		},
		{
			name: "should return proper module and package",
			fields: fields{
				require: map[string]string{
					"github.com/stretchr/testify": "v1.8.1",
				},
			},
			params: params{
				url: "github.com/stretchr/testify/assert",
			},
			expected: expected{
				module: "github.com/stretchr/testify",
				pkg:    "assert",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := Module{
				Require: tt.fields.require,
			}
			mod, pkg := m.SplitModulePackage(tt.params.url)
			assert.Equal(t, tt.expected.module, mod)
			assert.Equal(t, tt.expected.pkg, pkg)
		})
	}
}
