// SPDX-FileCopyrightText: 2025 OOMOL, Inc. <https://www.oomol.com>
// SPDX-License-Identifier: MPL-2.0

package wsl

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/oomol-lab/ovm-win/pkg/logger"
	"github.com/oomol-lab/ovm-win/pkg/util"
)

type Config struct {
	log *logger.Context
}

func NewConfig(log *logger.Context) *Config {
	return &Config{log: log}
}

func (c *Config) ExistIncompatible() bool {
	return c.findKey("wsl2", "kernel")
}

func (c *Config) Fix() error {
	return c.commentKey("kernel")
}

func (c *Config) Open() error {
	wslConfigPath, ok := c.path()
	if !ok {
		c.log.Info("WSL config file not found")
		return nil
	}

	notepadPath, ok := util.NotepadPath()
	if !ok {
		return fmt.Errorf("notepad not found")
	}

	cmd := exec.Command(notepadPath, wslConfigPath)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to open wsl config file: %w", err)
	}

	if err := cmd.Process.Release(); err != nil {
		c.log.Warnf("Failed to release notepad process: %v", err)
	}

	return nil
}

func (c *Config) findKey(expectSection string, expectKey string) bool {
	wslConfigPath, ok := c.path()
	if !ok {
		c.log.Info("WSL config file not found")
		return false
	}

	file, err := os.Open(wslConfigPath)
	if err != nil {
		c.log.Warnf("Failed to open .wslconfig file: %v", err)
		return false
	}
	defer file.Close()

	section := ""
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.ToLower(strings.TrimSpace(scanner.Text()))

		if strings.HasPrefix(line, "#") {
			continue
		}

		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section = line
		}

		if section != fmt.Sprintf("[%s]", expectSection) {
			continue
		}

		if section == line {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		if key != expectKey {
			continue
		}

		val := strings.Trim(strings.TrimSpace(parts[1]), `"`)
		if val == "" {
			continue
		}

		c.log.Infof("Find %s key in .wslconfig: %s", expectKey, line)
		return true
	}

	if err := scanner.Err(); err != nil {
		c.log.Warnf("Failed to scan .wslconfig: %v", err)
		return false
	}

	c.log.Infof("No %s key config found in WSL config file: %s", expectKey, wslConfigPath)
	return false
}

func (c *Config) commentKey(expectKey string) error {
	c.log.Infof("Ready comment %s key in .wslconfig", expectKey)

	wslConfigPath, ok := c.path()
	if !ok {
		c.log.Info("WSL config file not found, skip comment key")
		return nil
	}

	content, err := os.ReadFile(wslConfigPath)
	if err != nil {
		return fmt.Errorf("failed to read wslconfig file: %w", err)
	}

	reg, err := regexp.Compile(fmt.Sprintf(`(?im)^[ \t]*%s\s*=.*$`, expectKey))
	if err != nil {
		return fmt.Errorf("failed to compile wslconfig key regexp: %w", err)
	}

	newContent := reg.ReplaceAll(content, []byte("# $0"))

	if err := os.WriteFile(wslConfigPath, newContent, 0600); err != nil {
		return fmt.Errorf("failed to write wslconfig file: %w", err)
	}

	return nil
}

func (c *Config) path() (string, bool) {
	p, err := os.UserHomeDir()
	if err != nil {
		return "", false
	}

	p = filepath.Join(p, ".wslconfig")

	if util.Exists(p) != nil {
		return "", false
	}

	return p, true
}
