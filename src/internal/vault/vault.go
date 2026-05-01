// Package vault manages named task databases (vaults). Each vault is a
// separate SQLite file. The active vault is persisted in vaults.json so
// every CLI call and TUI session opens the same database.
package vault

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

func isWindows() bool { return runtime.GOOS == "windows" }

// Vault represents a single named database.
type Vault struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

// Config is the full vault registry, persisted to vaults.json.
type Config struct {
	Active string  `json:"active"`
	Vaults []Vault `json:"vaults"`

	dataDir string // not serialised; set by Load
}

// DataDir returns the directory where vaults.json and default db files live.
func DataDir() (string, error) {
	dir, err := platformDataDir()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("failed to create data directory: %w", err)
	}
	return dir, nil
}

func platformDataDir() (string, error) {
	// mirror the logic from storage.DataDir
	if isWindows() {
		appdata := os.Getenv("APPDATA")
		if appdata == "" {
			profile := os.Getenv("USERPROFILE")
			if profile == "" {
				return "", fmt.Errorf("could not determine APPDATA directory")
			}
			appdata = filepath.Join(profile, "AppData", "Roaming")
		}
		return filepath.Join(appdata, "clist"), nil
	}
	home := os.Getenv("HOME")
	if home == "" {
		home = os.Getenv("USERPROFILE")
	}
	if home == "" {
		return "", fmt.Errorf("could not determine home directory (set HOME or USERPROFILE)")
	}
	return filepath.Join(home, ".local", "share", "clist"), nil
}

// Load reads vaults.json from dataDir. If the file does not exist a default
// vault is created, migrating the legacy clist.db if present.
func Load(dataDir string) (*Config, error) {
	cfg := &Config{dataDir: dataDir}
	path := cfg.configPath()

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return cfg.bootstrap()
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", path, err)
	}
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", path, err)
	}
	cfg.dataDir = dataDir

	// Ensure Active points to a real vault.
	if cfg.Get(cfg.Active) == nil && len(cfg.Vaults) > 0 {
		cfg.Active = cfg.Vaults[0].Name
		_ = cfg.save()
	}
	return cfg, nil
}

// bootstrap creates the initial config, migrating clist.db if it exists.
func (c *Config) bootstrap() (*Config, error) {
	defaultPath := filepath.Join(c.dataDir, "clist.db")
	if _, err := os.Stat(defaultPath); os.IsNotExist(err) {
		defaultPath = filepath.Join(c.dataDir, "default.db")
	}
	c.Active = "default"
	c.Vaults = []Vault{{Name: "default", Path: defaultPath}}
	if err := c.save(); err != nil {
		return nil, err
	}
	return c, nil
}

func (c *Config) configPath() string {
	return filepath.Join(c.dataDir, "vaults.json")
}

func (c *Config) save() error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal vault config: %w", err)
	}
	if err := os.WriteFile(c.configPath(), data, 0o644); err != nil {
		return fmt.Errorf("failed to write vault config: %w", err)
	}
	return nil
}

// ActiveVault returns the currently active vault, or nil if config is empty.
func (c *Config) ActiveVault() *Vault {
	v := c.Get(c.Active)
	if v != nil {
		return v
	}
	if len(c.Vaults) > 0 {
		return &c.Vaults[0]
	}
	return nil
}

// Get returns the vault with the given name, or nil.
func (c *Config) Get(name string) *Vault {
	for i := range c.Vaults {
		if c.Vaults[i].Name == name {
			return &c.Vaults[i]
		}
	}
	return nil
}

// Add creates a new vault. The db file will be created on first open.
func (c *Config) Add(name string) error {
	name = canonicalize(name)
	if name == "" {
		return fmt.Errorf("vault name cannot be empty")
	}
	if !isValidName(name) {
		return fmt.Errorf("vault name %q may only contain letters, digits, hyphens and underscores", name)
	}
	if c.Get(name) != nil {
		return fmt.Errorf("vault %q already exists", name)
	}
	c.Vaults = append(c.Vaults, Vault{
		Name: name,
		Path: filepath.Join(c.dataDir, name+".db"),
	})
	return c.save()
}

// Switch sets the active vault.
func (c *Config) Switch(name string) error {
	name = canonicalize(name)
	if c.Get(name) == nil {
		return fmt.Errorf("vault %q not found", name)
	}
	c.Active = name
	return c.save()
}

// Remove deletes a vault from the registry. The db file is NOT deleted.
// If the removed vault was active, Active is updated to the first remaining
// vault. If it was the last vault, a fresh "default" vault is recreated.
// Returns true when the active vault changed (caller must reopen the DB).
func (c *Config) Remove(name string) (activeChanged bool, err error) {
	name = canonicalize(name)
	idx := -1
	for i, v := range c.Vaults {
		if v.Name == name {
			idx = i
			break
		}
	}
	if idx == -1 {
		return false, fmt.Errorf("vault %q not found", name)
	}

	wasActive := c.Active == name
	c.Vaults = append(c.Vaults[:idx], c.Vaults[idx+1:]...)

	if len(c.Vaults) == 0 {
		// Recreate a blank default vault.
		c.Vaults = []Vault{{Name: "default", Path: filepath.Join(c.dataDir, "default.db")}}
		c.Active = "default"
		return true, c.save()
	}

	if wasActive {
		c.Active = c.Vaults[0].Name
		return true, c.save()
	}
	return false, c.save()
}

func canonicalize(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

func isValidName(name string) bool {
	for _, r := range name {
		if !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_') {
			return false
		}
	}
	return len(name) > 0
}
