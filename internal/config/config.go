package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

type Profile struct {
	WinKeyDisabled   bool   `json:"winKeyDisabled"`
	InputEnglish     bool   `json:"inputEnglish"`
	AltShiftDisabled bool   `json:"altShiftDisabled"`
	AudioDeviceID    string `json:"audioDeviceId"`
}

type Paths struct {
	Accelerator string `json:"accelerator"`
	Voice       string `json:"voice"`
	Steam       string `json:"steam"`
}

type Profiles struct {
	Optimize Profile `json:"optimize"`
	Restore  Profile `json:"restore"`
}

type Config struct {
	Paths                 Paths    `json:"paths"`
	Shortcut              string   `json:"shortcut"`
	StartOneClickOptimize bool     `json:"startOneClickOptimize"`
	CloseOneClickRestore  bool     `json:"closeOneClickRestore"`
	Profiles              Profiles `json:"profiles"`
}

func Default() Config {
	return Config{
		Shortcut: "Ctrl+Alt+K",
		Profiles: Profiles{
			Optimize: Profile{WinKeyDisabled: true, InputEnglish: true, AltShiftDisabled: true},
			Restore:  Profile{},
		},
	}
}

type Store struct{ Path string }

func DefaultPath() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "Apex Runner", "config.json"), nil
}

func (s Store) Load() (Config, error) {
	c := Default()
	b, err := os.ReadFile(s.Path)
	if errors.Is(err, os.ErrNotExist) {
		return c, nil
	}
	if err != nil {
		return c, err
	}
	if err := json.Unmarshal(b, &c); err != nil {
		return Default(), err
	}
	if c.Shortcut == "" {
		c.Shortcut = Default().Shortcut
	}
	return c, nil
}

func (s Store) Save(c Config) error {
	if err := os.MkdirAll(filepath.Dir(s.Path), 0700); err != nil {
		return err
	}
	b, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	tmp, err := os.CreateTemp(filepath.Dir(s.Path), "config-*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err = tmp.Write(b); err != nil {
		tmp.Close()
		return err
	}
	if err = tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), s.Path)
}
