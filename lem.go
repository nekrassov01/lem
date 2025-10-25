package lem

import (
	"bufio"
	_ "embed"
	"encoding/json"
	"errors"
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

// initConfigPath is the default path to the configuration file.
const initConfigPath = "lem.toml"

var (
	//go:embed lem.toml
	initConfig []byte

	// gitDir is the directory name for the git repository.
	gitDir = ".git"

	// statePathFunc returns the path to the state file.
	statePathFunc = defaultStatePath

	// gray is a function that returns a gray color for printing messages.
	gray = color.New(color.FgHiBlack).SprintFunc()

	// cyan is a function that returns a cyan color for printing messages.
	cyan = color.New(color.FgHiCyan).SprintFunc()

	// green is a function that returns a green color for printing messages.
	green = color.New(color.FgHiGreen).SprintFunc()
)

// defaultStatePath returns the default path to the state file.
func defaultStatePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "lem", "state"), nil
}

// Config holds settings such as where the central env is located,
// how it is divided, and to which groups it is delivered.
// It is read from a configuration file in TOML format.
type Config struct {
	Stage map[string]string `toml:"stage"` // Stage holds the path to the central environment file.
	Group map[string]Group  `toml:"group"` // Group holds the configuration for each group of environment variables.

	path string    // path is the absolute path to the configuration file
	dir  string    // dir is the configuration file directory
	root string    // root is the project root directory with .git
	size int       // size is the size of the map to be allocated when reading the central env
	w    io.Writer // w is the writer to which the output is written
}

// Group groups environment variables using several parameters.
type Group struct {
	Prefix        string   `toml:"prefix"`  // Prefix for the environment variable names
	Dir           string   `toml:"dir"`     // Directory to which the environment variables are delivered
	Replaceable   []string `toml:"replace"` // List of prefixes to be delivered by replacing group prefixes
	Plain         []string `toml:"plain"`   // List of environment variables delivered without prefixes
	DirenvSupport []string `toml:"direnv"`  // Groups for which .envrc is generated
	IsCheck       bool     `toml:"check"`   // Whether to check for empty values
}

// Entry represents an environment variable entry.
type Entry struct {
	Group  string // Group is the group name of the environment variable
	Prefix string // Prefix is the prefix for the environment variable names of its group
	Type   string // Type indicates whether the env entry is indirect
	Name   string // Name is the key of the env entry, used for identification
	Value  string // Value is the value of the env entry
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
	if err := os.WriteFile(initConfigPath, initConfig, 0o600); err != nil {
		return fmt.Errorf("failed to initialize: %w", err)
	}
	fmt.Printf("%s %s\n", cyan("created:"), initConfigPath)
	return nil
}

// Load loads and instantiates the specified configuration file path.
func Load(path string, opts ...Option) (*Config, error) {
	absPath, isDir, err := sanitizePath(path)
	if err != nil {
		return nil, fmt.Errorf("failed to validate config path: %w", err)
	}
	if isDir {
		return nil, fmt.Errorf("failed to validate config path: %s: is a directory", path)
	}
	cfg := &Config{}
	if _, err := toml.DecodeFile(absPath, cfg); err != nil {
		return nil, fmt.Errorf("failed to decode config file: %w", err)
	}
	cfg.path = absPath
	cfg.dir = filepath.Dir(absPath)
	cfg.root = projectRoot(cfg.dir)
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
	if err := cfg.validateStageTable(); err != nil {
		return err
	}
	if err := cfg.validateGroupTable(); err != nil {
		return err
	}
	for stage := range cfg.Stage {
		if _, err := cfg.validateStagePair(stage); err != nil {
			return err
		}
	}
	for id, group := range cfg.Group {
		if _, err := cfg.validateGroupPair(id, group); err != nil {
			return err
		}
	}
	_, _ = fmt.Fprintln(cfg.w, green("all checks passed!"))
	return nil
}

// Current shows the current stage context.
func (cfg *Config) Current() error {
	if err := cfg.validateStageTable(); err != nil {
		return err
	}
	stage, err := cfg.loadStage()
	if err != nil {
		return err
	}
	if _, err := cfg.validateStagePair(stage); err != nil {
		return err
	}
	_, _ = fmt.Fprintln(cfg.w, cyan("current: ", stage))
	return nil
}

// Switch switches the current stage to the specified one.
func (cfg *Config) Switch(stage string) error {
	if err := cfg.validateStageTable(); err != nil {
		return err
	}
	if _, err := cfg.validateStagePair(stage); err != nil {
		return err
	}
	if err := cfg.storeStage(stage); err != nil {
		return err
	}
	_, _ = fmt.Fprintln(cfg.w, cyan("switched: ", stage))
	return nil
}

// List returns a slice of Entry for all env entries of all groups for the given stage.
// If stage is empty, returns an error.
func (cfg *Config) List() ([]Entry, error) {
	if err := cfg.validateStageTable(); err != nil {
		return nil, err
	}
	stage, err := cfg.loadStage()
	if err != nil {
		return nil, fmt.Errorf("failed to load stage: %w", err)
	}
	path, err := cfg.validateStagePair(stage)
	if err != nil {
		return nil, err
	}
	if err := cfg.validateGroupTable(); err != nil {
		return nil, err
	}
	e, n, err := readEnv(path, cfg.size)
	if err != nil {
		return nil, fmt.Errorf("failed to read central env: %w", err)
	}
	entries := make([]Entry, 0, n)
	for name, group := range cfg.Group {
		for k, v := range e {
			if after, ok := strings.CutPrefix(k, group.Prefix+"_"); ok {
				entries = append(entries, Entry{
					Group:  name,
					Prefix: group.Prefix,
					Type:   "direct",
					Name:   after,
					Value:  v,
				})
			}
		}
		for _, prefix := range group.Replaceable {
			for k, v := range e {
				if after, ok := strings.CutPrefix(k, prefix+"_"); ok {
					entries = append(entries, Entry{
						Group:  name,
						Prefix: group.Prefix,
						Type:   "indirect",
						Name:   after,
						Value:  v,
					})
				}
			}
		}
		for _, key := range group.Plain {
			if v, ok := e[key]; ok {
				entries = append(entries, Entry{
					Group:  name,
					Prefix: group.Prefix,
					Type:   "plain",
					Name:   key,
					Value:  v,
				})
			}
		}
	}
	slices.SortFunc(entries, func(a, b Entry) int {
		if a.Group != b.Group {
			return strings.Compare(a.Group, b.Group)
		}
		if a.Type != b.Type {
			return strings.Compare(a.Type, b.Type)
		}
		return strings.Compare(a.Name, b.Name)
	})
	return entries, nil
}

// Run reads the central environment and divides and distributes it
// to each group based on the configuration file. If necessary,
// it also checks if the environment variable values are empty.
func (cfg *Config) Run() (string, error) {
	if err := cfg.validateStageTable(); err != nil {
		return "", err
	}
	stage, err := cfg.loadStage()
	if err != nil {
		return "", fmt.Errorf("failed to load stage: %w", err)
	}
	path, err := cfg.validateStagePair(stage)
	if err != nil {
		return "", err
	}
	if err := cfg.validateGroupTable(); err != nil {
		return "", err
	}
	e, _, err := readEnv(path, cfg.size)
	if err != nil {
		return "", fmt.Errorf("failed to read central env: %w", err)
	}
	msgs := make([]string, len(cfg.Group))
	i := 0
	_, _ = fmt.Fprintf(cfg.w, "%s %s %s %s\n", gray("staged:"), stage, gray("->"), path)
	for id, group := range cfg.Group {
		dir, err := cfg.validateGroupPair(id, group)
		if err != nil {
			return "", err
		}
		// Collect prefix matching entries from the central env to the group
		// Some entries are added with group prefixes based on configuration
		o := makeEnv(group, e, cfg.size)
		// Check for empty values if specified
		if group.IsCheck {
			for k, v := range o {
				if v == "" || v == "''" || v == `""` || v == "``" {
					return "", fmt.Errorf("failed to validate: empty value: %s", k)
				}
			}
		}
		// Create .envrc file if specified
		if len(group.DirenvSupport) != 0 {
			_, err = cfg.createEnvrc(group, dir)
			if err != nil {
				return "", fmt.Errorf("failed to create .envrc for group.%s: %w", id, err)
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
		_, _ = fmt.Fprintln(cfg.w, msg)
	}
	return path, nil
}

// Watch watches for changes in the env file for the specified
// stage and executes the run command when a change is detected.
// Monitoring continues as long as it is not interrupted.
func (cfg *Config) Watch() (string, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return "", fmt.Errorf("failed to create watcher: %w", err)
	}
	defer func() {
		if closeErr := watcher.Close(); closeErr != nil {
			err = errors.Join(err, fmt.Errorf("failed to close watcher: %w", closeErr))
		}
	}()
	stagePath, err := cfg.Run()
	if err != nil {
		return "", err
	}
	dir := filepath.Dir(stagePath)
	if err := watcher.Add(dir); err != nil {
		return "", fmt.Errorf("failed to add dir to watcher: %w", err)
	}
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
					_, _ = fmt.Fprintln(cfg.w, cyan("rerun..."))
					if _, err := cfg.Run(); err != nil {
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
		return "", err
	}
	return stagePath, err
}

// validateStageTable checks if the stage table is set in the configuration.
func (cfg *Config) validateStageTable() error {
	if len(cfg.Stage) == 0 {
		return fmt.Errorf("failed to validate stage: stage not set in %s", cfg.path)
	}
	return nil
}

// validateStagePair checks if the stage is set in the configuration and returns its absolute path.
func (cfg *Config) validateStagePair(stage string) (string, error) {
	path, ok := cfg.Stage[stage]
	if !ok {
		return "", fmt.Errorf("failed to validate stage: %s: not set in %s", stage, cfg.path)
	}
	absPath, isDir, err := cfg.resolvePath(path)
	if err != nil {
		return "", fmt.Errorf("failed to validate stage path: %s: %w", stage, err)
	}
	if isDir {
		return "", fmt.Errorf("failed to validate stage path: %s: is a directory", stage)
	}
	return absPath, nil
}

// validateGroupTable checks if the group table is set in the configuration.
func (cfg *Config) validateGroupTable() error {
	if len(cfg.Group) == 0 {
		return fmt.Errorf("failed to validate group: group not set in %s", cfg.path)
	}
	return nil
}

// validateGroupPair checks if the group is set in the configuration and returns its absolute path.
func (cfg *Config) validateGroupPair(id string, group Group) (string, error) {
	if group.Prefix == "" {
		return "", fmt.Errorf("failed to validate group.%s: prefix not set in %s", id, cfg.path)
	}
	if group.Dir == "" {
		return "", fmt.Errorf("failed to validate group.%s: dir not set in %s", id, cfg.path)
	}
	absPath, isDir, err := cfg.resolvePath(group.Dir)
	if err != nil {
		return "", fmt.Errorf("failed to validate group.%s: %w", id, err)
	}
	if !isDir {
		return "", fmt.Errorf("failed to validate group.%s: is not a directory", id)
	}
	if slices.Contains(group.Replaceable, "") {
		return "", fmt.Errorf("failed to validate: group.%s: `replace` contains empty", id)
	}
	if slices.Contains(group.Plain, "") {
		return "", fmt.Errorf("failed to validate: group.%s: `plain` contains empty", id)
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

// createEnvrc creates a .envrc file for direnv support in the specified group directory.
func (cfg *Config) createEnvrc(group Group, dir string) (string, error) {
	dest := filepath.Join(dir, ".envrc")
	b := strings.Builder{}
	b.Grow(2048)
	for _, target := range group.DirenvSupport {
		g := cfg.Group[target]
		envDir, isDir, err := cfg.resolvePath(g.Dir)
		if err != nil {
			return "", fmt.Errorf("%s: %w", target, err)
		}
		if !isDir {
			return "", fmt.Errorf("%s: is not a directory", target)
		}
		relPath, err := filepath.Rel(dir, envDir)
		if err != nil {
			return "", fmt.Errorf("%s: %w", target, err)
		}
		b.WriteString(fmt.Sprintf("watch_file %s/.env\n", relPath))
		b.WriteString(fmt.Sprintf("dotenv_if_exists %s/.env\n", relPath))
	}
	if err := os.WriteFile(dest, []byte(b.String()), 0o600); err != nil {
		return "", fmt.Errorf("failed to write .envrc file: %w", err)
	}
	return dest, nil
}

// resolvePath resolves the given path relative to the configuration directory.
func (cfg *Config) resolvePath(path string) (string, bool, error) {
	var absPath string
	if filepath.IsAbs(path) {
		absPath = filepath.Clean(path)
	} else {
		absPath = filepath.Clean(filepath.Join(cfg.dir, path))
	}
	relPath, err := filepath.Rel(cfg.root, absPath)
	if err != nil {
		return "", false, fmt.Errorf("failed to resolve path: %w", err)
	}
	if strings.HasPrefix(relPath, "..") {
		return "", false, fmt.Errorf("failed to resolve path: outside of the project root: %s", absPath)
	}
	info, err := os.Stat(absPath)
	if err != nil {
		return "", false, fmt.Errorf("failed to stat resolved path: %w", err)
	}
	return absPath, info.IsDir(), nil
}

// storeStage stores the current stage in the state file.
func (cfg *Config) storeStage(stage string) error {
	path, err := statePathFunc()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	state := map[string]map[string]string{}
	if data, err := os.ReadFile(filepath.Clean(path)); err == nil && len(data) > 0 {
		if err := json.Unmarshal(data, &state); err != nil {
			return err
		}
	}
	state[cfg.path] = map[string]string{"stage": stage}
	b, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o600)
}

// loadStage loads the current stage from the state file.
func (cfg *Config) loadStage() (string, error) {
	path, err := statePathFunc()
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return "", err
	}
	m := map[string]map[string]string{}
	if err := json.Unmarshal(data, &m); err != nil {
		return "", err
	}
	v, ok := m[cfg.path]
	if !ok {
		return "", fmt.Errorf("no stage stored for config: %s", cfg.path)
	}
	stage, ok := v["stage"]
	if !ok {
		return "", fmt.Errorf("no stage value for config: %s", cfg.path)
	}
	return stage, nil
}

// projectRoot finds the project root directory by looking for the .git directory.
// It traverses up the directory tree until it finds the .git directory or reaches the root.
func projectRoot(baseDir string) string {
	current := filepath.Clean(baseDir)
	for {
		root := filepath.Join(current, gitDir)
		info, err := os.Stat(root)
		if err == nil && info.IsDir() {
			return current
		}
		parent := filepath.Dir(current)
		if parent == current {
			break
		}
		current = parent
	}
	return baseDir
}

// readEnv reads the environment variables from the specified path and returns them as a map.
func readEnv(path string, size int) (map[string]string, int, error) {
	env := make(map[string]string, size)
	f, err := os.Open(filepath.Clean(path))
	if err != nil {
		return nil, 0, err
	}
	defer func() {
		if closeErr := f.Close(); closeErr != nil {
			err = errors.Join(err, fmt.Errorf("failed to close file: %w", closeErr))
		}
	}()
	i := 0
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
			i++
		}
	}
	if scanErr := scanner.Err(); scanErr != nil {
		err = scanErr
		return nil, 0, err
	}
	return env, i, err
}

// makeEnv creates a map of environment variables for the specified group.
// It filters the base environment variables based on the group's prefix and replaceable prefixes.
func makeEnv(group Group, base map[string]string, size int) map[string]string {
	e := make(map[string]string, size)
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
		for _, key := range group.Plain {
			if k == key {
				e[k] = v
			}
		}
	}
	return e
}

// writeEnv writes the environment variables to the specified path.
func writeEnv(path string, env map[string]string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return fmt.Errorf("failed to create env dir: %w", err)
	}
	f, err := os.Create(filepath.Clean(path))
	if err != nil {
		return fmt.Errorf("failed to create env file: %w", err)
	}
	defer func() {
		if closeErr := f.Close(); closeErr != nil {
			err = errors.Join(err, fmt.Errorf("failed to close file: %w", closeErr))
		}
	}()
	w := bufio.NewWriter(f)
	keys := make([]string, 0, len(env))
	for k := range env {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	for _, k := range keys {
		v := env[k]
		_, _ = fmt.Fprintf(w, "%s=%s\n", k, v)
	}
	if flushErr := w.Flush(); flushErr != nil {
		return fmt.Errorf("failed to flush env file: %w", flushErr)
	}
	return err
}

// sanitizePath sanitizes the given path by resolving it to an absolute path.
func sanitizePath(path string) (string, bool, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", false, fmt.Errorf("failed to get abs path: %w", err)
	}
	info, err := os.Stat(absPath)
	if err != nil {
		return "", false, fmt.Errorf("failed to stat sanitized path: %w", err)
	}
	return absPath, info.IsDir(), nil
}
