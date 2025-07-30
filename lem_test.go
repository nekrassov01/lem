package lem

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	gitDir = dummyGitDir
	statePathFunc = dummyStatePath
	defer func() {
		gitDir = defaultGitDir
		statePathFunc = defaultStatePath
		_ = os.Remove("testdata/sandbox/state")
	}()
	m.Run()
}

func prepareState(path, stage string) {
	statePath, err := dummyStatePath()
	if err != nil {
		panic(err)
	}
	m := map[string]map[string]string{
		path: {
			"stage": stage,
		},
	}
	data, err := json.Marshal(m)
	if err != nil {
		panic(err)
	}
	if err := os.WriteFile(statePath, data, 0o0600); err != nil {
		panic(err)
	}
}

func TestWithWriter(t *testing.T) {
	type args struct {
		w io.Writer
	}
	type expected struct {
		w io.Writer
	}
	tests := []struct {
		name     string
		args     args
		expected expected
	}{
		{
			name:     "basic",
			args:     args{w: &bytes.Buffer{}},
			expected: expected{w: &bytes.Buffer{}},
		},
		{
			name:     "nil",
			args:     args{w: nil},
			expected: expected{w: os.Stdout},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := &Config{}
			WithWriter(tt.args.w)(actual)
			if tt.expected.w == os.Stdout {
				assert.Equal(t, os.Stdout, actual.w)
			} else {
				_, ok := actual.w.(*bytes.Buffer)
				assert.True(t, ok)
			}
		})
	}
}

func TestWithSize(t *testing.T) {
	type args struct {
		size int
	}
	type expected struct {
		size int
	}
	tests := []struct {
		name     string
		args     args
		expected expected
	}{
		{
			name:     "basic",
			args:     args{size: 1},
			expected: expected{size: 1},
		},
		{
			name:     "zero",
			args:     args{size: 0},
			expected: expected{size: 32},
		},
		{
			name:     "negative",
			args:     args{size: -1},
			expected: expected{size: 32},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := &Config{}
			WithSize(tt.args.size)(actual)
			assert.Equal(t, tt.expected.size, actual.size)
		})
	}
}

func TestInit(t *testing.T) {
	type expected struct {
		isError bool
	}
	tests := []struct {
		name     string
		expected expected
	}{
		{
			name:     "basic",
			expected: expected{isError: false},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Init()
			if tt.expected.isError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestLoad(t *testing.T) {
	type args struct {
		path string
		opts []Option
	}
	type expected struct {
		cfg     *Config
		isError bool
	}
	tests := []struct {
		name     string
		args     args
		expected expected
	}{
		{
			name: "basic",
			args: args{
				path: "testdata/sandbox/lem.toml",
				opts: nil,
			},
			expected: expected{
				cfg: &Config{
					Stage: map[string]string{
						"default":  "master/.env",
						"dev":      "master/.env.development",
						"noexists": "master/.env.noexists",
					},
					Group: map[string]Group{
						"api": {
							Prefix:        "API",
							Dir:           "./api",
							Replaceable:   []string{"REPLACEABLE1", "REPLACEABLE2"},
							Plain:         []string{"FOO", "BAR"},
							DirenvSupport: []string{"api", "ui"},
							IsCheck:       true,
						},
						"ui": {
							Prefix:        "UI",
							Dir:           "./ui",
							Replaceable:   []string{"REPLACEABLE1"},
							Plain:         []string{"BAZ"},
							DirenvSupport: []string{"ui"},
							IsCheck:       false,
						},
					},
					path: func() string {
						path, _ := filepath.Abs("testdata/sandbox/lem.toml")
						return path
					}(),
					dir: func() string {
						path, _ := filepath.Abs("testdata/sandbox")
						return path
					}(),
					root: func() string {
						path, _ := filepath.Abs("testdata/sandbox")
						return path
					}(),
					size: 32,
					w:    os.Stdout,
				},
				isError: false,
			},
		},
		{
			name: "with options",
			args: args{
				path: "testdata/sandbox/lem.toml",
				opts: []Option{
					WithSize(1),
					WithWriter(&bytes.Buffer{}),
				},
			},
			expected: expected{
				cfg: &Config{
					Stage: map[string]string{
						"default":  "master/.env",
						"dev":      "master/.env.development",
						"noexists": "master/.env.noexists",
					},
					Group: map[string]Group{
						"api": {
							Prefix:        "API",
							Dir:           "./api",
							Replaceable:   []string{"REPLACEABLE1", "REPLACEABLE2"},
							Plain:         []string{"FOO", "BAR"},
							DirenvSupport: []string{"api", "ui"},
							IsCheck:       true,
						},
						"ui": {
							Prefix:        "UI",
							Dir:           "./ui",
							Replaceable:   []string{"REPLACEABLE1"},
							Plain:         []string{"BAZ"},
							DirenvSupport: []string{"ui"},
							IsCheck:       false,
						},
					},
					path: func() string {
						path, _ := filepath.Abs("testdata/sandbox/lem.toml")
						return path
					}(),
					dir: func() string {
						path, _ := filepath.Abs("testdata/sandbox")
						return path
					}(),
					root: func() string {
						path, _ := filepath.Abs("testdata/sandbox")
						return path
					}(),
					size: 1,
					w:    &bytes.Buffer{},
				},
				isError: false,
			},
		},
		{
			name: "empty file",
			args: args{
				path: "testdata/sandbox/lem.empty.toml",
				opts: nil,
			},
			expected: expected{
				cfg: &Config{
					Stage: nil,
					Group: nil,
					path: func() string {
						path, _ := filepath.Abs("testdata/sandbox/lem.empty.toml")
						return path
					}(),
					dir: func() string {
						path, _ := filepath.Abs("testdata/sandbox")
						return path
					}(),
					root: func() string {
						path, _ := filepath.Abs("testdata/sandbox")
						return path
					}(),
					size: 32,
					w:    os.Stdout,
				},
				isError: false,
			},
		},
		{
			name: "invalid file",
			args: args{
				path: "testdata/sandbox/lem.invalid.toml",
				opts: nil,
			},
			expected: expected{
				cfg:     nil,
				isError: true,
			},
		},
		{
			name: "file not found",
			args: args{
				path: "testdata/sandbox/lem.dummy.toml",
				opts: nil,
			},
			expected: expected{
				cfg:     nil,
				isError: true,
			},
		},
		{
			name: "is a directory",
			args: args{
				path: "testdata/sandbox/",
				opts: nil,
			},
			expected: expected{
				cfg:     nil,
				isError: true,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := Load(tt.args.path, tt.args.opts...)
			if (err != nil) && tt.expected.isError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.expected.cfg, cfg)
		})
	}
}

func TestConfig_Validate(t *testing.T) {
	type fields struct {
		Stage map[string]string
		Group map[string]Group
		path  string
		size  int
		w     io.Writer
	}
	type expected struct {
		isError bool
	}
	tests := []struct {
		name     string
		fields   fields
		expected expected
	}{
		{
			name: "basic",
			fields: fields{
				Stage: map[string]string{
					"default": "testdata/sandbox/master/.env",
				},
				Group: map[string]Group{
					"api": {
						Prefix:      "API",
						Dir:         "testdata/sandbox/api",
						Replaceable: []string{"REPLACEABLE1", "REPLACEABLE2"},
						IsCheck:     true,
					},
					"ui": {
						Prefix:      "UI",
						Dir:         "testdata/sandbox/ui",
						Replaceable: []string{"REPLACEABLE1"},
						IsCheck:     false,
					},
				},
				path: "testdata/sandbox/lem.toml",
				size: 32,
				w:    io.Discard,
			},
			expected: expected{
				isError: false,
			},
		},
		{
			name: "stage table not found",
			fields: fields{
				Stage: nil,
				Group: map[string]Group{
					"api": {
						Prefix:      "API",
						Dir:         "testdata/sandbox/api",
						Replaceable: []string{"REPLACEABLE1"},
						IsCheck:     false,
					},
				},
				path: "testdata/sandbox/lem.toml",
				size: 32,
				w:    io.Discard,
			},
			expected: expected{
				isError: true,
			},
		},
		{
			name: "invalid stage path",
			fields: fields{
				Stage: map[string]string{
					"dummy": "../.env",
				},
				Group: map[string]Group{
					"api": {
						Prefix:      "API",
						Dir:         "testdata/sandbox/api",
						Replaceable: []string{"REPLACEABLE1"},
						IsCheck:     false,
					},
				},
				path: "testdata/sandbox/lem.toml",
				size: 32,
				w:    io.Discard,
			},
			expected: expected{
				isError: true,
			},
		},
		{
			name: "stage path not found",
			fields: fields{
				Stage: map[string]string{
					"default": "./.dummy",
				},
				Group: map[string]Group{
					"api": {
						Prefix:      "API",
						Dir:         "testdata/sandbox/api",
						Replaceable: []string{"REPLACEABLE1"},
						IsCheck:     false,
					},
				},
				path: "testdata/sandbox/lem.toml",
				size: 32,
				w:    io.Discard,
			},
			expected: expected{
				isError: true,
			},
		},
		{
			name: "stage is a directory",
			fields: fields{
				Stage: map[string]string{
					"dummy": "testdata/sandbox",
				},
				Group: map[string]Group{
					"api": {
						Prefix:      "API",
						Dir:         "testdata/sandbox/api",
						Replaceable: []string{"REPLACEABLE1"},
						IsCheck:     false,
					},
				},
				path: "testdata/sandbox/lem.toml",
				size: 32,
				w:    io.Discard,
			},
			expected: expected{
				isError: true,
			},
		},
		{
			name: "group table not found",
			fields: fields{
				Stage: map[string]string{
					"default": "testdata/sandbox/master/.env",
				},
				Group: nil,
				path:  "testdata/sandbox/lem.toml",
				size:  32,
				w:     os.Stdout,
			},
			expected: expected{
				isError: true,
			},
		},
		{
			name: "empty group prefix",
			fields: fields{
				Stage: map[string]string{
					"default": "testdata/sandbox/master/.env",
				},
				Group: map[string]Group{
					"api": {
						Prefix:      "",
						Dir:         "testdata/sandbox/api",
						Replaceable: []string{"REPLACEABLE1", "REPLACEABLE2"},
						IsCheck:     true,
					},
				},
				path: "testdata/sandbox/lem.toml",
				size: 32,
				w:    io.Discard,
			},
			expected: expected{
				isError: true,
			},
		},
		{
			name: "empty group dir",
			fields: fields{
				Stage: map[string]string{
					"default": "testdata/sandbox/master/.env",
				},
				Group: map[string]Group{
					"api": {
						Prefix:      "API",
						Dir:         "",
						Replaceable: []string{"REPLACEABLE1", "REPLACEABLE2"},
						IsCheck:     true,
					},
				},
				path: "testdata/sandbox/lem.toml",
				size: 32,
				w:    io.Discard,
			},
			expected: expected{
				isError: true,
			},
		},
		{
			name: "invalid group path",
			fields: fields{
				Stage: map[string]string{
					"default": "testdata/sandbox/master/.env",
				},
				Group: map[string]Group{
					"api": {
						Prefix:      "API",
						Dir:         "../api",
						Replaceable: []string{"REPLACEABLE1", "REPLACEABLE2"},
						IsCheck:     true,
					},
				},
				path: "testdata/sandbox/lem.toml",
				size: 32,
				w:    io.Discard,
			},
			expected: expected{
				isError: true,
			},
		},
		{
			name: "group path not found",
			fields: fields{
				Stage: map[string]string{
					"default": "testdata/sandbox/master/.env",
				},
				Group: map[string]Group{
					"api": {
						Prefix:      "API",
						Dir:         "testdata/sandbox/api/.env",
						Replaceable: []string{"REPLACEABLE1", "REPLACEABLE2"},
						IsCheck:     true,
					},
				},
				path: "testdata/sandbox/lem.toml",
				size: 32,
				w:    io.Discard,
			},
			expected: expected{
				isError: true,
			},
		},
		{
			name: "group path is not a directory",
			fields: fields{
				Stage: map[string]string{
					"default": "testdata/sandbox/master/.env",
				},
				Group: map[string]Group{
					"api": {
						Prefix:      "API",
						Dir:         "testdata/sandbox/dummy",
						Replaceable: []string{"REPLACEABLE1", "REPLACEABLE2"},
						IsCheck:     true,
					},
				},
				path: "testdata/sandbox/lem.toml",
				size: 32,
				w:    io.Discard,
			},
			expected: expected{
				isError: true,
			},
		},
		{
			name: "group replaceable array contains empty string",
			fields: fields{
				Stage: map[string]string{
					"default": "testdata/sandbox/master/.env",
				},
				Group: map[string]Group{
					"api": {
						Prefix:      "API",
						Dir:         "testdata/sandbox/api/",
						Replaceable: []string{"FOO", ""},
						IsCheck:     true,
					},
				},
				path: "testdata/sandbox/lem.toml",
				size: 32,
				w:    io.Discard,
			},
			expected: expected{
				isError: true,
			},
		},
		{
			name: "group plain array contains empty string",
			fields: fields{
				Stage: map[string]string{
					"default": "testdata/sandbox/master/.env",
				},
				Group: map[string]Group{
					"api": {
						Prefix:  "API",
						Dir:     "testdata/sandbox/api/",
						Plain:   []string{"FOO", "BAR", ""},
						IsCheck: true,
					},
				},
				path: "testdata/sandbox/lem.toml",
				size: 32,
				w:    io.Discard,
			},
			expected: expected{
				isError: true,
			},
		},
		{
			name: "group direnv array contains empty string",
			fields: fields{
				Stage: map[string]string{
					"default": "testdata/sandbox/master/.env",
				},
				Group: map[string]Group{
					"api": {
						Prefix:        "API",
						Dir:           "testdata/sandbox/api/",
						Replaceable:   []string{"REPLACEABLE1", "REPLACEABLE2"},
						IsCheck:       true,
						DirenvSupport: []string{"api", ""},
					},
				},
				path: "testdata/sandbox/lem.toml",
				size: 32,
				w:    io.Discard,
			},
			expected: expected{
				isError: true,
			},
		},
		{
			name: "group direnv array contains invalid id",
			fields: fields{
				Stage: map[string]string{
					"default": "testdata/sandbox/master/.env",
				},
				Group: map[string]Group{
					"api": {
						Prefix:        "API",
						Dir:           "testdata/sandbox/api/",
						Replaceable:   []string{"REPLACEABLE1", "REPLACEABLE2"},
						IsCheck:       true,
						DirenvSupport: []string{"api", "invalid"},
					},
				},
				path: "testdata/sandbox/lem.toml",
				size: 32,
				w:    io.Discard,
			},
			expected: expected{
				isError: true,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Stage: tt.fields.Stage,
				Group: tt.fields.Group,
				path:  tt.fields.path,
				size:  tt.fields.size,
				w:     tt.fields.w,
			}
			err := cfg.Validate()
			if tt.expected.isError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestConfig_Current(t *testing.T) {
	type fields struct {
		Stage map[string]string
		Group map[string]Group
		path  string
		size  int
		w     io.Writer
	}
	type expected struct {
		isError bool
	}
	tests := []struct {
		name     string
		fields   fields
		expected expected
		setup    func()
	}{
		{
			name: "basic",
			fields: fields{
				Stage: map[string]string{
					"default": "testdata/sandbox/master/.env",
				},
				path: "testdata/sandbox/lem.toml",
				size: 32,
				w:    io.Discard,
			},
			expected: expected{
				isError: false,
			},
			setup: func() {
				prepareState("testdata/sandbox/lem.toml", "default")
			},
		},
		{
			name: "stage table not found",
			fields: fields{
				Stage: nil,
				path:  "testdata/sandbox/lem.toml",
				size:  32,
				w:     io.Discard,
			},
			expected: expected{
				isError: true,
			},
			setup: func() {
				prepareState("testdata/sandbox/lem.toml", "default")
			},
		},
		{
			name: "missing stage in config",
			fields: fields{
				Stage: map[string]string{
					"default": "testdata/sandbox/master/.env",
				},
				path: "testdata/sandbox/lem.toml",
				size: 32,
				w:    io.Discard,
			},
			expected: expected{
				isError: true,
			},
			setup: func() {
				prepareState("testdata/sandbox/lem.toml", "dummy")
			},
		},
		{
			name: "missing env file",
			fields: fields{
				Stage: map[string]string{
					"default": "testdata/sandbox/master/.env",
				},
				path: "testdata/sandbox/lem.toml",
				size: 32,
				w:    io.Discard,
			},
			expected: expected{
				isError: true,
			},
			setup: func() {
				prepareState("testdata/sandbox/lem.toml", "noexists")
			},
		},
		{
			name: "missing config path in state",
			fields: fields{
				Stage: map[string]string{
					"default": "testdata/sandbox/master/.env",
				},
				path: "testdata/sandbox/lem.toml",
				size: 32,
				w:    io.Discard,
			},
			expected: expected{
				isError: true,
			},
			setup: func() {
				prepareState("testdata/sandbox/invalid", "default")
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			cfg := &Config{
				Stage: tt.fields.Stage,
				Group: tt.fields.Group,
				path:  tt.fields.path,
				size:  tt.fields.size,
				w:     tt.fields.w,
			}
			err := cfg.Current()
			if tt.expected.isError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestConfig_Switch(t *testing.T) {
	type fields struct {
		Stage map[string]string
		Group map[string]Group
		path  string
		size  int
		w     io.Writer
	}
	type args struct {
		stage string
	}
	type expected struct {
		isError bool
	}
	tests := []struct {
		name     string
		fields   fields
		args     args
		expected expected
		setup    func()
	}{
		{
			name: "basic",
			fields: fields{
				Stage: map[string]string{
					"default": "testdata/sandbox/master/.env",
				},
				path: "testdata/sandbox/lem.toml",
				size: 32,
				w:    io.Discard,
			},
			args: args{
				stage: "default",
			},
			expected: expected{
				isError: false,
			},
			setup: func() {
				prepareState("testdata/sandbox/lem.toml", "default")
			},
		},
		{
			name: "stage table not found",
			fields: fields{
				Stage: nil,
				path:  "testdata/sandbox/lem.toml",
				size:  32,
				w:     io.Discard,
			},
			args: args{
				stage: "default",
			},
			expected: expected{
				isError: true,
			},
			setup: func() {
				prepareState("testdata/sandbox/lem.toml", "default")
			},
		},
		{
			name: "missing stage in config",
			fields: fields{
				Stage: map[string]string{
					"default": "testdata/sandbox/master/.env",
				},
				path: "testdata/sandbox/lem.toml",
				size: 32,
				w:    io.Discard,
			},
			args: args{
				stage: "dummy",
			},
			expected: expected{
				isError: true,
			},
			setup: func() {
				prepareState("testdata/sandbox/lem.toml", "default")
			},
		},
		{
			name: "missing config path in state",
			fields: fields{
				Stage: map[string]string{
					"default": "testdata/sandbox/master/.env",
				},
				path: "testdata/sandbox/lem.toml",
				size: 32,
				w:    io.Discard,
			},
			args: args{
				stage: "default",
			},
			expected: expected{
				isError: false, // Written as a new config path
			},
			setup: func() {
				prepareState("testdata/sandbox/invalid", "default")
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			cfg := &Config{
				Stage: tt.fields.Stage,
				Group: tt.fields.Group,
				path:  tt.fields.path,
				size:  tt.fields.size,
				w:     tt.fields.w,
			}
			err := cfg.Switch(tt.args.stage)
			if tt.expected.isError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestConfig_List(t *testing.T) {
	type fields struct {
		Stage map[string]string
		Group map[string]Group
		path  string
		size  int
		w     io.Writer
	}
	type expected struct {
		entries []Entry
		isError bool
	}
	tests := []struct {
		name     string
		fields   fields
		expected expected
		setup    func()
	}{
		{
			name: "basic",
			fields: fields{
				Stage: map[string]string{
					"default": "testdata/sandbox/master/.env",
				},
				Group: map[string]Group{
					"api": {
						Prefix:        "API",
						Dir:           "./api",
						Replaceable:   []string{"REPLACEABLE1", "REPLACEABLE2"},
						Plain:         []string{"FOO", "BAR"},
						IsCheck:       true,
						DirenvSupport: []string{"api", "ui"},
					},
					"ui": {
						Prefix:        "UI",
						Dir:           "./ui",
						Replaceable:   []string{"REPLACEABLE1"},
						Plain:         []string{"BAZ"},
						IsCheck:       false,
						DirenvSupport: []string{"ui"},
					},
				},
				path: "testdata/sandbox/lem.toml",
				size: 32,
				w:    io.Discard,
			},
			expected: expected{
				entries: []Entry{
					{Group: "api", Prefix: "API", Type: "direct", Name: "1_ENV", Value: "111"},
					{Group: "api", Prefix: "API", Type: "direct", Name: "2_ENV", Value: "\"222\""},
					{Group: "api", Prefix: "API", Type: "direct", Name: "3_ENV", Value: "'333'"},
					{Group: "api", Prefix: "API", Type: "direct", Name: "4_ENV", Value: "`444`"},
					{Group: "api", Prefix: "API", Type: "indirect", Name: "6_ENV", Value: "6 7 8"},
					{Group: "api", Prefix: "API", Type: "plain", Name: "BAR", Value: "bar"},
					{Group: "api", Prefix: "API", Type: "plain", Name: "FOO", Value: "foo"},
					{Group: "ui", Prefix: "UI", Type: "direct", Name: "5_ENV", Value: "555"},
					{Group: "ui", Prefix: "UI", Type: "indirect", Name: "6_ENV", Value: "6 7 8"},
					{Group: "ui", Prefix: "UI", Type: "plain", Name: "BAZ", Value: "baz"},
				},
				isError: false,
			},
			setup: func() {
				prepareState("testdata/sandbox/lem.toml", "default")
			},
		},
		{
			name: "stage table not found",
			fields: fields{
				Stage: nil,
				path:  "testdata/sandbox/lem.toml",
				size:  32,
				w:     io.Discard,
			},
			expected: expected{
				isError: true,
			},
			setup: func() {
				prepareState("testdata/sandbox/lem.toml", "default")
			},
		},
		{
			name: "missing stage in config",
			fields: fields{
				Stage: map[string]string{
					"default": "testdata/sandbox/master/.env",
				},
				path: "testdata/sandbox/lem.toml",
				size: 32,
				w:    io.Discard,
			},
			expected: expected{
				isError: true,
			},
			setup: func() {
				prepareState("testdata/sandbox/lem.toml", "dummy")
			},
		},
		{
			name: "group table not found",
			fields: fields{
				Stage: map[string]string{
					"default": "testdata/sandbox/master/.env",
				},
				Group: nil,
				path:  "testdata/sandbox/lem.toml",
				size:  32,
				w:     os.Stdout,
			},
			expected: expected{
				isError: true,
			},
			setup: func() {
				prepareState("testdata/sandbox/lem.toml", "default")
			},
		},
		{
			name: "missing config path in state",
			fields: fields{
				Stage: map[string]string{
					"default": "testdata/sandbox/master/.env",
				},
				path: "testdata/sandbox/lem.toml",
				size: 32,
				w:    io.Discard,
			},
			expected: expected{
				isError: true,
			},
			setup: func() {
				prepareState("testdata/sandbox/invalid", "default")
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			cfg := &Config{
				Stage: tt.fields.Stage,
				Group: tt.fields.Group,
				path:  tt.fields.path,
				size:  tt.fields.size,
				w:     tt.fields.w,
			}
			actual, err := cfg.List()
			if tt.expected.isError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, actual, tt.expected.entries)
		})
	}
}

func TestConfig_Run(t *testing.T) {
	type fields struct {
		Stage map[string]string
		Group map[string]Group
		path  string
		size  int
		w     io.Writer
	}
	type expected struct {
		path    string
		isError bool
	}
	tests := []struct {
		name     string
		fields   fields
		expected expected
		setup    func()
	}{
		{
			name: "basic",
			fields: fields{
				Stage: map[string]string{
					"default": "testdata/sandbox/master/.env",
				},
				Group: map[string]Group{
					"api": {
						Prefix:        "API",
						Dir:           "testdata/sandbox/api",
						Replaceable:   []string{"REPLACEABLE1", "REPLACEABLE2"},
						IsCheck:       true,
						DirenvSupport: []string{"api"},
					},
				},
				path: "testdata/sandbox/lem.toml",
				size: 32,
				w:    io.Discard,
			},
			expected: expected{
				path:    "testdata/sandbox/master/.env",
				isError: false,
			},
			setup: func() {
				prepareState("testdata/sandbox/lem.toml", "default")
			},
		},
		{
			name: "stage table not found",
			fields: fields{
				Stage: nil,
				Group: map[string]Group{
					"api": {
						Prefix:      "API",
						Dir:         "testdata/sandbox/api",
						Replaceable: []string{"REPLACEABLE1", "REPLACEABLE2"},
						IsCheck:     true,
					},
				},
				path: "testdata/sandbox/lem.toml",
				size: 32,
				w:    io.Discard,
			},
			expected: expected{
				path:    "",
				isError: true,
			},
			setup: func() {
				prepareState("testdata/sandbox/lem.toml", "default")
			},
		},
		{
			name: "stage path not found",
			fields: fields{
				Stage: map[string]string{
					"default": "testdata/sandbox/dummy/.env",
				},
				Group: map[string]Group{
					"api": {
						Prefix:      "API",
						Dir:         "testdata/sandbox/api",
						Replaceable: []string{"REPLACEABLE1", "REPLACEABLE2"},
						IsCheck:     true,
					},
				},
				path: "testdata/sandbox/lem.toml",
				size: 32,
				w:    io.Discard,
			},
			expected: expected{
				path:    "",
				isError: true,
			},
			setup: func() {
				prepareState("testdata/sandbox/lem.toml", "default")
			},
		},
		{
			name: "group table not found",
			fields: fields{
				Stage: map[string]string{
					"default": "testdata/sandbox/master/.env",
				},
				Group: nil,
				path:  "testdata/sandbox/lem.toml",
				size:  32,
				w:     os.Stdout,
			},
			expected: expected{
				path:    "",
				isError: true,
			},
			setup: func() {
				prepareState("testdata/sandbox/lem.toml", "default")
			},
		},
		{
			name: "group path not found",
			fields: fields{
				Stage: map[string]string{
					"default": "testdata/sandbox/master/.env",
				},
				Group: map[string]Group{
					"api": {
						Prefix:      "API",
						Dir:         "testdata/sandbox/api/.env",
						Replaceable: []string{"REPLACEABLE1", "REPLACEABLE2"},
						IsCheck:     true,
					},
				},
				path: "testdata/sandbox/lem.toml",
				size: 32,
				w:    io.Discard,
			},
			expected: expected{
				path:    "",
				isError: true,
			},
			setup: func() {
				prepareState("testdata/sandbox/lem.toml", "default")
			},
		},
		{
			name: "central env not found",
			fields: fields{
				Stage: map[string]string{
					"default": "testdata/sandbox/master/.env.dummy",
				},
				Group: map[string]Group{
					"api": {
						Prefix:      "API",
						Dir:         "testdata/sandbox/api",
						Replaceable: []string{"REPLACEABLE1", "REPLACEABLE2"},
						IsCheck:     true,
					},
				},
				path: "testdata/sandbox/lem.toml",
				size: 32,
				w:    io.Discard,
			},
			expected: expected{
				path:    "",
				isError: true,
			},
			setup: func() {
				prepareState("testdata/sandbox/lem.toml", "default")
			},
		},
		{
			name: "empty value",
			fields: fields{
				Stage: map[string]string{
					"default": "testdata/sandbox/master/.env.error",
				},
				Group: map[string]Group{
					"api": {
						Prefix:        "API",
						Dir:           "testdata/sandbox/api",
						Replaceable:   []string{"REPLACEABLE1", "REPLACEABLE2"},
						IsCheck:       true,
						DirenvSupport: []string{"api"},
					},
				},
				path: "testdata/sandbox/lem.toml",
				size: 32,
				w:    io.Discard,
			},
			expected: expected{
				path:    "",
				isError: true,
			},
			setup: func() {
				prepareState("testdata/sandbox/lem.toml", "default")
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			cfg := &Config{
				Stage: tt.fields.Stage,
				Group: tt.fields.Group,
				path:  tt.fields.path,
				size:  tt.fields.size,
				w:     tt.fields.w,
			}
			actual, err := cfg.Run()
			if tt.expected.isError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.expected.path, actual)
		})
	}
}

func TestConfig_Watch(t *testing.T) {
	type fields struct {
		Stage map[string]string
		Group map[string]Group
		path  string
		size  int
		w     io.Writer
	}
	type expected struct {
		path    string
		isError bool
	}
	tests := []struct {
		name     string
		fields   fields
		expected expected
	}{
		{
			name: "stop at error",
			fields: fields{
				Stage: nil,
				Group: map[string]Group{
					"api": {
						Prefix:      "API",
						Dir:         "testdata/sandbox/api",
						Replaceable: []string{"REPLACEABLE1", "REPLACEABLE2"},
						IsCheck:     true,
					},
				},
				path: "testdata/sandbox/lem.toml",
				size: 32,
				w:    io.Discard,
			},
			expected: expected{
				path:    "",
				isError: true,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Stage: tt.fields.Stage,
				Group: tt.fields.Group,
				path:  tt.fields.path,
				size:  tt.fields.size,
				w:     tt.fields.w,
			}
			actual, err := cfg.Watch()
			if tt.expected.isError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.expected.path, actual)
		})
	}
}

func Test_createEnvrc(t *testing.T) {
	type fields struct {
		Stage map[string]string
		Group map[string]Group
		path  string
		dir   string
		root  string
		size  int
		w     io.Writer
	}
	type args struct {
		group Group
		dir   string
	}
	type expected struct {
		content string
		isError bool
	}
	tests := []struct {
		name     string
		fields   fields
		args     args
		expected expected
	}{
		{
			name: "basic",
			fields: fields{
				Stage: map[string]string{
					"default": "dummy",
				},
				Group: map[string]Group{
					"api": {
						Prefix: "API",
						Dir: func() string {
							path, _ := filepath.Abs("testdata/sandbox/api")
							return path
						}(),
						Replaceable:   []string{"REPLACEABLE1", "REPLACEABLE2"},
						IsCheck:       true,
						DirenvSupport: []string{"api", "ui"},
					},
					"ui": {
						Prefix: "UI",
						Dir: func() string {
							path, _ := filepath.Abs("testdata/sandbox/ui")
							return path
						}(),
						Replaceable:   []string{"REPLACEABLE1"},
						IsCheck:       false,
						DirenvSupport: []string{"ui"},
					},
				},
				dir: func() string {
					path, _ := filepath.Abs("testdata/sandbox")
					return path
				}(),
				root: func() string {
					path, _ := filepath.Abs("testdata/sandbox")
					return path
				}(),
			},
			args: args{
				group: Group{
					Prefix: "API",
					Dir: func() string {
						path, _ := filepath.Abs("testdata/sandbox/api")
						return path
					}(),
					Replaceable:   []string{"REPLACEABLE1", "REPLACEABLE2"},
					IsCheck:       true,
					DirenvSupport: []string{"api", "ui"},
				},
				dir: func() string {
					path, _ := filepath.Abs("testdata/sandbox/api")
					return path
				}(),
			},
			expected: expected{
				content: "watch_file ./.env\ndotenv_if_exists ./.env\nwatch_file ../ui/.env\ndotenv_if_exists ../ui/.env\n",
				isError: false,
			},
		},
		{
			name: "resolve error",
			fields: fields{
				Stage: map[string]string{
					"default": "dummy",
				},
				Group: map[string]Group{
					"api": {
						Prefix: "API",
						Dir: func() string {
							path, _ := filepath.Abs("testdata/sandbox/api")
							return path
						}(),
						Replaceable:   []string{"REPLACEABLE1", "REPLACEABLE2"},
						IsCheck:       true,
						DirenvSupport: []string{"api", "ui"},
					},
					"ui": {
						Prefix: "UI",
						Dir: func() string {
							path, _ := filepath.Abs("testdata/sandbox/ui")
							return path
						}(),
						Replaceable:   []string{"REPLACEABLE1"},
						IsCheck:       false,
						DirenvSupport: []string{"ui"},
					},
				},
				dir:  "testdata/sandbox",
				root: "testdata/sandbox",
			},
			args: args{
				group: Group{
					Prefix: "API",
					Dir: func() string {
						path, _ := filepath.Abs("testdata/sandbox/api")
						return path
					}(),
					Replaceable:   []string{"REPLACEABLE1", "REPLACEABLE2"},
					IsCheck:       true,
					DirenvSupport: []string{"api", "ui"},
				},
				dir: func() string {
					path, _ := filepath.Abs("testdata/sandbox/api")
					return path
				}(),
			},
			expected: expected{
				content: "",
				isError: true,
			},
		},
		{
			name: "directory but file",
			fields: fields{
				Stage: map[string]string{
					"default": "dummy",
				},
				Group: map[string]Group{
					"api": {
						Prefix: "API",
						Dir: func() string {
							path, _ := filepath.Abs("testdata/sandbox/api/.env")
							return path
						}(),
						Replaceable:   []string{"REPLACEABLE1", "REPLACEABLE2"},
						IsCheck:       true,
						DirenvSupport: []string{"api", "ui"},
					},
					"ui": {
						Prefix: "UI",
						Dir: func() string {
							path, _ := filepath.Abs("testdata/sandbox/ui")
							return path
						}(),
						Replaceable:   []string{"REPLACEABLE1"},
						IsCheck:       false,
						DirenvSupport: []string{"ui"},
					},
				},
				dir: func() string {
					path, _ := filepath.Abs("testdata/sandbox")
					return path
				}(),
			},
			args: args{
				group: Group{
					Prefix: "API",
					Dir: func() string {
						path, _ := filepath.Abs("testdata/sandbox/api/.env")
						return path
					}(),
					Replaceable:   []string{"REPLACEABLE1", "REPLACEABLE2"},
					IsCheck:       true,
					DirenvSupport: []string{"api", "ui"},
				},
				dir: func() string {
					path, _ := filepath.Abs("testdata/sandbox/api")
					return path
				}(),
			},
			expected: expected{
				content: "",
				isError: true,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Stage: tt.fields.Stage,
				Group: tt.fields.Group,
				path:  tt.fields.path,
				dir:   tt.fields.dir,
				root:  tt.fields.root,
				size:  tt.fields.size,
				w:     tt.fields.w,
			}
			path, err := cfg.createEnvrc(tt.args.group, tt.args.dir)
			if tt.expected.isError {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			content, err := os.ReadFile(filepath.Clean(path))
			if err != nil {
				t.Fatalf("failed to read written file: %v", err)
			}
			assert.Equal(t, string(content), tt.expected.content)
		})
	}
}

func Test_projectRoot(t *testing.T) {
	type args struct {
		dir string
	}
	type expected struct {
		dir string
	}
	tests := []struct {
		name     string
		args     args
		gitDir   string
		expected expected
	}{
		{
			name: "basic",
			args: args{
				dir: "testdata/sandbox",
			},
			expected: expected{
				dir: "testdata/sandbox",
			},
		},
		{
			name: "child",
			args: args{
				dir: "testdata/sandbox/api",
			},
			expected: expected{
				dir: "testdata/sandbox",
			},
		},
		{
			name: "nested",
			args: args{
				dir: "testdata/sandbox/api/subdir",
			},
			expected: expected{
				dir: "testdata/sandbox",
			},
		},
		{
			name: ".git not found",
			args: args{
				dir: "testdata/sandbox",
			},
			gitDir: ".notfound",
			expected: expected{
				dir: "testdata/sandbox",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.gitDir != "" {
				gitDir = tt.gitDir
			}
			actual := projectRoot(tt.args.dir)
			assert.Equal(t, tt.expected.dir, actual)
			gitDir = dummyGitDir
		})
	}
}

func Test_readEnv(t *testing.T) {
	type args struct {
		path string
		size int
	}
	type expected struct {
		e       map[string]string
		n       int
		isError bool
	}
	tests := []struct {
		name     string
		args     args
		expected expected
	}{
		{
			name: "patterns",
			args: args{
				path: "testdata/sandbox/master/.env",
				size: 32,
			},
			expected: expected{
				e: map[string]string{
					"API_1_ENV":          "111",
					"API_2_ENV":          "\"222\"",
					"API_3_ENV":          "'333'",
					"API_4_ENV":          "`444`",
					"BAR":                "bar",
					"BAZ":                "baz",
					"FOO":                "foo",
					"REPLACEABLE1_6_ENV": "6 7 8",
					"UI_5_ENV":           "555",
				},
				n:       9,
				isError: false,
			},
		},
		{
			name: "empty file",
			args: args{
				path: "testdata/sandbox/master/.env.empty",
				size: 32,
			},
			expected: expected{
				e:       map[string]string{},
				n:       0,
				isError: false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, n, err := readEnv(tt.args.path, tt.args.size)
			if tt.expected.isError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.expected.e, m)
			assert.Equal(t, tt.expected.n, n)
		})
	}
}

func Test_writeEnv(t *testing.T) {
	type args struct {
		env map[string]string
	}
	type expected struct {
		content string
		isError bool
	}
	tests := []struct {
		name     string
		args     args
		expected expected
	}{
		{
			name: "basic",
			args: args{
				env: map[string]string{
					"ZKEY": "zvalue",
					"AKEY": "avalue",
					"CKEY": "cvalue",
				},
			},
			expected: expected{
				content: "AKEY=avalue\nCKEY=cvalue\nZKEY=zvalue\n",
				isError: false,
			},
		},
		{
			name: "empty map",
			args: args{
				env: map[string]string{},
			},
			expected: expected{
				content: "",
				isError: false,
			},
		},
		{
			name: "single",
			args: args{
				env: map[string]string{
					"KEY1": "value1",
				},
			},
			expected: expected{
				content: "KEY1=value1\n",
				isError: false,
			},
		},
		{
			name: "contains spaces",
			args: args{
				env: map[string]string{
					"SPACES": "value with spaces",
					"TABS":   "value\twith\ttabs",
				},
			},
			expected: expected{
				content: "SPACES=value with spaces\nTABS=value\twith\ttabs\n",
				isError: false,
			},
		},
		{
			name: "empty value",
			args: args{
				env: map[string]string{
					"EMPTY": "",
					"FULL":  "content",
				},
			},
			expected: expected{
				content: "EMPTY=\nFULL=content\n",
				isError: false,
			},
		},
		{
			name: "special chars",
			args: args{
				env: map[string]string{
					"URL":     "https://example.com?a=b&c=d",
					"CONTROL": "line1\nline2",
					"HASH":    "value#with#hash",
				},
			},
			expected: expected{
				content: "CONTROL=line1\nline2\nHASH=value#with#hash\nURL=https://example.com?a=b&c=d\n",
				isError: false,
			},
		},
	}
	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), fmt.Sprintf("%d.env", i))
			err := writeEnv(path, tt.args.env)
			if tt.expected.isError {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			content, err := os.ReadFile(filepath.Clean(path))
			if err != nil {
				t.Fatalf("failed to read written file: %v", err)
			}
			assert.Equal(t, tt.expected.content, string(content))
		})
	}
}
