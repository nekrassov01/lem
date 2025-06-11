package lem

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

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
						"default": "master/.env",
						"dev":     "master/.env.development",
					},
					Group: map[string]Group{
						"api": {
							Prefix:        "API",
							Dir:           "./api",
							Replaceable:   []string{"REPLACEABLE1", "REPLACEABLE2"},
							IsCheck:       true,
							DirenvSupport: []string{"api", "ui"},
						},
						"ui": {
							Prefix:        "UI",
							Dir:           "./ui",
							Replaceable:   []string{"REPLACEABLE1"},
							IsCheck:       false,
							DirenvSupport: []string{"ui"},
						},
					},
					path: "testdata/sandbox/lem.toml",
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
						"default": "master/.env",
						"dev":     "master/.env.development",
					},
					Group: map[string]Group{
						"api": {
							Prefix:        "API",
							Dir:           "./api",
							Replaceable:   []string{"REPLACEABLE1", "REPLACEABLE2"},
							IsCheck:       true,
							DirenvSupport: []string{"api", "ui"},
						},
						"ui": {
							Prefix:        "UI",
							Dir:           "./ui",
							Replaceable:   []string{"REPLACEABLE1"},
							IsCheck:       false,
							DirenvSupport: []string{"ui"},
						},
					},
					path: "testdata/sandbox/lem.toml",
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
					path:  "testdata/sandbox/lem.empty.toml",
					size:  32,
					w:     os.Stdout,
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
				w:    os.Stdout,
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
				w:    os.Stdout,
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
				w:    os.Stdout,
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
				w:    os.Stdout,
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
				w:    os.Stdout,
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
				w:    os.Stdout,
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
				w:    os.Stdout,
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
				w:    os.Stdout,
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
				w:    os.Stdout,
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
				w:    os.Stdout,
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
						Replaceable: []string{"REPLACEABLE1", ""},
						IsCheck:     true,
					},
				},
				path: "testdata/sandbox/lem.toml",
				size: 32,
				w:    os.Stdout,
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
				w:    os.Stdout,
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
				w:    os.Stdout,
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

func TestConfig_Run(t *testing.T) {
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
		path    string
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
				w:    os.Stdout,
			},
			args: args{
				stage: "default",
			},
			expected: expected{
				path:    "testdata/sandbox/master/.env",
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
						Replaceable: []string{"REPLACEABLE1", "REPLACEABLE2"},
						IsCheck:     true,
					},
				},
				path: "testdata/sandbox/lem.toml",
				size: 32,
				w:    os.Stdout,
			},
			args: args{
				stage: "default",
			},
			expected: expected{
				path:    "",
				isError: true,
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
				w:    os.Stdout,
			},
			args: args{
				stage: "default",
			},
			expected: expected{
				path:    "",
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
			args: args{
				stage: "default",
			},
			expected: expected{
				path:    "",
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
				w:    os.Stdout,
			},
			args: args{
				stage: "default",
			},
			expected: expected{
				path:    "",
				isError: true,
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
				w:    os.Stdout,
			},
			args: args{
				stage: "default",
			},
			expected: expected{
				path:    "",
				isError: true,
			},
		},
		{
			name: "invalid stage passed",
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
				},
				path: "testdata/sandbox/lem.toml",
				size: 32,
				w:    os.Stdout,
			},
			args: args{
				stage: "dummy",
			},
			expected: expected{
				path:    "",
				isError: true,
			},
		},
		{
			name: "empty value in central env",
			fields: fields{
				Stage: map[string]string{
					"default": "testdata/sandbox/master/.env.error",
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
				w:    os.Stdout,
			},
			args: args{
				stage: "default",
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
			actual, err := cfg.Run(tt.args.stage)
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
	type args struct {
		stage string
	}
	type expected struct {
		path    string
		isError bool
	}
	tests := []struct {
		name     string
		fields   fields
		args     args
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
				w:    os.Stdout,
			},
			args: args{
				stage: "default",
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
			actual, err := cfg.Watch(tt.args.stage)
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
					"default": "testdata/sandbox/master/.env",
				},
				Group: map[string]Group{
					"api": {
						Prefix:        "API",
						Dir:           "testdata/sandbox/api",
						Replaceable:   []string{"REPLACEABLE1", "REPLACEABLE2"},
						IsCheck:       true,
						DirenvSupport: []string{"api", "ui"},
					},
					"ui": {
						Prefix:        "UI",
						Dir:           "testdata/sandbox/ui",
						Replaceable:   []string{"REPLACEABLE1"},
						IsCheck:       false,
						DirenvSupport: []string{"ui"},
					},
				},
			},
			args: args{
				group: Group{
					Prefix:        "API",
					Dir:           "testdata/sandbox/api",
					Replaceable:   []string{"REPLACEABLE1", "REPLACEABLE2"},
					IsCheck:       true,
					DirenvSupport: []string{"api", "ui"},
				},
				dir: t.TempDir(),
			},
			expected: expected{
				content: func() string {
					a, _ := filepath.Abs("testdata/sandbox/api/.env")
					b, _ := filepath.Abs("testdata/sandbox/ui/.env")
					return fmt.Sprintf("watch_file %s\ndotenv %s\nwatch_file %s\ndotenv %s\n", a, a, b, b)
				}(),
				isError: false,
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
			path, err := cfg.createEnvrc(tt.args.group, tt.args.dir)
			if tt.expected.isError {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			content, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("failed to read written file: %v", err)
			}
			assert.Contains(t, string(content), tt.expected.content)
		})
	}
}

func Test_readEnv(t *testing.T) {
	type args struct {
		path string
		size int
	}
	type expected struct {
		env     map[string]string
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
				env: map[string]string{
					"API_1_ENV":          "111",
					"API_2_ENV":          "\"222\"",
					"API_3_ENV":          "'333'",
					"API_4_ENV":          "`444`",
					"UI_5_ENV":           "555",
					"REPLACEABLE1_6_ENV": "6 7 8",
				},
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
				env:     map[string]string{},
				isError: false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, err := readEnv(tt.args.path, tt.args.size)
			if tt.expected.isError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.expected.env, m)
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
			content, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("failed to read written file: %v", err)
			}
			assert.Equal(t, tt.expected.content, string(content))
		})
	}
}
