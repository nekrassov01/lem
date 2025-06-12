package lem

import (
	"bufio"
	_ "embed"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/fatih/color"
	"github.com/fsnotify/fsnotify"
)

const initConfigPath = "lem.toml"

//go:embed lem.toml
var initConfig []byte

var gray = color.New(color.FgHiBlack).SprintFunc()
var cyan = color.New(color.FgHiCyan).SprintFunc()

// Config holds settings such as where the central env is located,
// how it is divided, and to which groups it is delivered.
// It is read from a configuration file in TOML format.
type Config struct {
	Stage map[string]string `toml:"stage"` // Stage holds the path to the central environment file.
	Group map[string]Group  `toml:"group"` // Group holds the configuration for each group of environment variables.

	path string
	dir  string
	size int
	w    io.Writer
}

// Group groups environment variables using several parameters.
type Group struct {
	Prefix        string   `toml:"prefix"`  // Prefix for the environment variable names
	Dir           string   `toml:"dir"`     // Directory to which the environment variables are delivered
	Replaceable   []string `toml:"replace"` // List of prefixes to be replaced with the group prefix
	IsCheck       bool     `toml:"check"`   // Whether to check for empty values
	DirenvSupport []string `toml:"direnv"`  // Whether to create .envrc for direnv support
}

// Option is an option given when loading the configuration file.
type Option func(*Config)

// WithSize sets the size to be allocated when reading the
// central env into the map. If not used, this value remains 32.
func WithSize(size int) Option {
	if size <= 0 {
		size = 32
	}
	return func(cfg *Config) {
		cfg.size = size
	}
}

// WithWriter sets the specified writer to the Config.
// If not used, the output remains standard output.
func WithWriter(w io.Writer) Option {
	if w == nil {
		w = os.Stdout
	}
	return func(cfg *Config) {
		cfg.w = w
	}
}

// Init initializes the configuration file with an example.
// You can use this to create a new configuration file.
func Init() error {
	if err := os.WriteFile(initConfigPath, initConfig, 0o644); err != nil {
		return fmt.Errorf("failed to initialize: %w", err)
	}
	fmt.Printf("%s %s\n", cyan("created:"), initConfigPath)
	return nil
}

// Load loads and instantiates the specified configuration file path.
func Load(path string, opts ...Option) (*Config, error) {
	absPath, idDir, err := sanitizeConfigPath(path)
	if err != nil {
		return nil, err
	}
	if idDir {
		return nil, fmt.Errorf("failed to load config: %s: is a directory", path)
	}
	cfg := &Config{}
	if _, err := toml.DecodeFile(absPath, cfg); err != nil {
		return nil, fmt.Errorf("failed to decode config file: %w", err)
	}
	cfg.path = absPath
	cfg.dir = filepath.Dir(absPath)
	cfg.size = 32
	cfg.w = os.Stdout
	for _, opt := range opts {
		opt(cfg)
	}
	return cfg, nil
}

// Validate verifies that the configuration file is executable.
// In addition to syntax checks, it also checks whether the path exists.
func (cfg *Config) Validate() error {
	if err := cfg.validateStage(); err != nil {
		return err
	}
	if err := cfg.validateGroup(); err != nil {
		return err
	}
	fmt.Fprintln(cfg.w, cyan("all checks passed!"))
	return nil
}

// Run reads the central environment and divides and distributes it
// to each group based on the configuration file. If necessary,
// it also checks if the environment variable values are empty.
func (cfg *Config) Run(stage string) (string, error) {
	// Validate the stage table exists in the configuration
	if err := cfg.validateStageTable(); err != nil {
		return "", err
	}
	// Validate the specified stage exists
	path, err := cfg.validateStagePair(stage)
	if err != nil {
		return "", err
	}

	// Validate the group table exists in the configuration
	if err := cfg.validateGroupTable(); err != nil {
		return "", err
	}

	// Read the central env
	e, err := readEnv(path, cfg.size)
	if err != nil {
		return "", fmt.Errorf("failed to read central env: %w", err)
	}

	msgs := make([]string, len(cfg.Group))
	i := 0
	fmt.Fprintf(cfg.w, "%s %s %s %s\n", gray("staged:"), stage, gray("->"), path)

	for id, group := range cfg.Group {
		// Validate the group configuration
		dir, err := cfg.validateGroupPair(id, group)
		if err != nil {
			return "", err
		}

		// Gathers entries from the central env that forward match the group prefix.
		// Also, replacement targets set in the group are added after replacing them with the group prefix.
		o := cfg.makeEnv(group, e)

		// Check for empty values if IsCheck is set
		if group.IsCheck {
			for k, v := range o {
				if v == "" || v == "''" || v == `""` || v == "``" {
					return "", fmt.Errorf("failed to validate: empty value: %s", k)
				}
			}
		}

		if len(group.DirenvSupport) != 0 {
			_, err = cfg.createEnvrc(group, dir)
			if err != nil {
				return "", err
			}
		}

		// Write the environment variables to the group's env file
		target := filepath.Join(dir, ".env")
		if err := writeEnv(target, o); err != nil {
			return "", fmt.Errorf("failed to write env file for group.%s: %w", id, err)
		}

		msgs[i] = fmt.Sprintf("%s group.%s %s %s", gray("distributed:"), id, gray("->"), target)
		i++
	}

	slices.Sort(msgs)
	for _, msg := range msgs {
		fmt.Fprintln(cfg.w, msg)
	}

	return path, nil
}

// Watch watches for changes in the env file for the specified
// stage and executes the run command when a change is detected.
// Monitoring continues as long as it is not interrupted.
func (cfg *Config) Watch(stage string) (string, error) {
	// Create a central env watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return "", fmt.Errorf("failed to create watcher: %w", err)
	}
	defer watcher.Close()

	// Run before monitoring starts
	stagePath, err := cfg.Run(stage)
	if err != nil {
		return "", err
	}

	// Add the directory of the stage file to the watcher
	dir := filepath.Dir(stagePath)
	if err := watcher.Add(dir); err != nil {
		return "", fmt.Errorf("failed to add dir %s to watcher: %w", dir, err)
	}

	// Watch for changes in the stage file
	done := make(chan error)

	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				var (
					isTarget      = event.Name == stagePath
					isCreateEvent = event.Op&fsnotify.Create == fsnotify.Create
					isWriteEvent  = event.Op&fsnotify.Write == fsnotify.Write
				)
				if isTarget && (isWriteEvent || isCreateEvent) {
					fmt.Fprintln(cfg.w, cyan("rerun..."))
					if _, err := cfg.Run(stage); err != nil {
						done <- err
						return
					}
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				done <- err
				return
			}
		}
	}()

	if err := <-done; err != nil {
		return stagePath, err
	}
	return stagePath, nil
}

func (cfg *Config) validateStage() error {
	if err := cfg.validateStageTable(); err != nil {
		return err
	}
	for stage := range cfg.Stage {
		if _, err := cfg.validateStagePair(stage); err != nil {
			return err
		}
	}
	return nil
}

func (cfg *Config) validateStageTable() error {
	if len(cfg.Stage) == 0 {
		return fmt.Errorf("failed to validate: stage not set in %s", cfg.path)
	}
	return nil
}

func (cfg *Config) validateStagePair(stage string) (string, error) {
	path, ok := cfg.Stage[stage]
	if !ok {
		return "", fmt.Errorf("failed to validate stage: %s: not set in %s", stage, cfg.path)
	}
	absPath, isDir, err := resolveEnvPath(cfg.dir, path)
	if err != nil {
		return "", fmt.Errorf("failed to resolve stage path: %s: %w", stage, err)
	}
	if isDir {
		return "", fmt.Errorf("failed to resolve stage path: %s: is a directory", stage)
	}
	return absPath, nil
}

func (cfg *Config) validateGroup() error {
	if err := cfg.validateGroupTable(); err != nil {
		return err
	}
	for id, group := range cfg.Group {
		if _, err := cfg.validateGroupPair(id, group); err != nil {
			return err
		}
	}
	return nil
}

func (cfg *Config) validateGroupTable() error {
	if len(cfg.Group) == 0 {
		return fmt.Errorf("failed to validate: group not set in %s", cfg.path)
	}
	return nil
}

func (cfg *Config) validateGroupPair(id string, group Group) (string, error) {
	if group.Prefix == "" {
		return "", fmt.Errorf("failed to validate group.%s: prefix not set in %s", id, cfg.path)
	}
	if group.Dir == "" {
		return "", fmt.Errorf("failed to validate group.%s: dir not set in %s", id, cfg.path)
	}
	absPath, isDir, err := resolveEnvPath(cfg.dir, group.Dir)
	if err != nil {
		return "", fmt.Errorf("failed to resolve group.%s: %w", id, err)
	}
	if !isDir {
		return "", fmt.Errorf("failed to resolve group.%s: is not a directory", id)
	}
	if slices.Contains(group.Replaceable, "") {
		return "", fmt.Errorf("failed to validate: group.%s: `replace` contains empty", id)
	}
	if slices.Contains(group.DirenvSupport, "") {
		return "", fmt.Errorf("failed to validate: group.%s: `direnv` contains empty", id)
	}
	for _, s := range group.DirenvSupport {
		if _, ok := cfg.Group[s]; !ok {
			return "", fmt.Errorf("failed to validate: group.%s: invalid id: %s", id, s)
		}
	}
	return absPath, nil
}

func (cfg *Config) createEnvrc(group Group, dir string) (string, error) {
	dest := filepath.Join(dir, ".envrc")
	b := strings.Builder{}
	b.Grow(2048)
	for _, id := range group.DirenvSupport {
		g := cfg.Group[id]
		envDir, isDir, err := resolveEnvPath(cfg.dir, g.Dir)
		if err != nil {
			return "", fmt.Errorf("direnv-support: %w", err)
		}
		if !isDir {
			return "", fmt.Errorf("direnv-support: failed to resolve group.%s: is not a directory", id)
		}
		envPath := filepath.Join(envDir, ".env")
		b.WriteString(fmt.Sprintf("watch_file %s\n", envPath))
		b.WriteString(fmt.Sprintf("dotenv_if_exists %s\n", envPath))
	}
	if err := os.WriteFile(dest, []byte(b.String()), 0o644); err != nil {
		return "", fmt.Errorf("failed to write .envrc file: %w", err)
	}
	return dest, nil
}

func (cfg *Config) makeEnv(group Group, base map[string]string) map[string]string {
	e := make(map[string]string, cfg.size)
	for k, v := range base {
		if strings.HasPrefix(k, group.Prefix+"_") {
			e[k] = v
		}
		for _, prefix := range group.Replaceable {
			if strings.HasPrefix(k, prefix+"_") {
				u := strings.Replace(k, prefix, group.Prefix, 1)
				e[u] = v
			}
		}
	}
	return e
}

func readEnv(path string, size int) (map[string]string, error) {
	env := make(map[string]string, size)
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		kv := strings.SplitN(line, "=", 2)
		if len(kv) == 2 {
			k := strings.TrimSpace(kv[0])
			v := strings.TrimSpace(kv[1])
			env[k] = v
		}
	}
	return env, scanner.Err()
}

func writeEnv(path string, env map[string]string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("failed to create dir: %w", err)
	}

	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create env file: %w", err)
	}
	defer f.Close()

	w := bufio.NewWriter(f)
	keys := make([]string, 0, len(env))
	for k := range env {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	for _, k := range keys {
		v := env[k]
		fmt.Fprintf(w, "%s=%s\n", k, v)
	}
	if err := w.Flush(); err != nil {
		return fmt.Errorf("failed to flush env file: %w", err)
	}
	return nil
}

func resolveEnvPath(parent, path string) (string, bool, error) {
	// Provide directory traversal protection
	if !filepath.IsAbs(path) {
		path = filepath.Join(parent, path)
	}
	parent = filepath.Clean(parent)
	path = filepath.Clean(path)
	relPath, err := filepath.Rel(parent, path)
	if err != nil {
		return "", false, fmt.Errorf("failed to resolve path: %w", err)
	}
	if strings.HasPrefix(relPath, "..") {
		return "", false, fmt.Errorf("failed to resolve path: outside of the configuration directory: %s", path)
	}
	info, err := os.Stat(path)
	if err != nil {
		return "", false, fmt.Errorf("failed to stat resolved path: %w", err)
	}
	return path, info.IsDir(), nil
}

func sanitizeConfigPath(path string) (string, bool, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", false, fmt.Errorf("failed to get abs path: %w", err)
	}
	info, err := os.Stat(absPath)
	if err != nil {
		return "", false, fmt.Errorf("failed to stat configuration file: %w", err)
	}
	return absPath, info.IsDir(), nil
}
