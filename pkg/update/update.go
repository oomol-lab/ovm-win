// SPDX-FileCopyrightText: 2024 OOMOL, Inc. <https://www.oomol.com>
// SPDX-License-Identifier: MPL-2.0

package update

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"github.com/oomol-lab/ovm-win/pkg/logger"
	"github.com/oomol-lab/ovm-win/pkg/util"

	"github.com/oomol-lab/ovm-win/pkg/cli"
	"github.com/oomol-lab/ovm-win/pkg/types"
)

type Updater interface {
	CheckAndReplace(log *logger.Context) error
}

type context struct {
	types.Version

	opt      *cli.Context
	jsonPath string
}

func New(opt *cli.Context) (updater Updater) {
	return &context{
		Version:  opt.Version,
		opt:      opt,
		jsonPath: filepath.Join(opt.ImageDir, "versions.json"),
	}
}

func (c *context) save() error {
	data, err := json.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to marshal versions: %w", err)
	}

	if err := os.WriteFile(c.jsonPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write versions to %s: %w", c.jsonPath, err)
	}

	return nil
}

func (c *context) needUpdate() (result []types.VersionKey) {
	jsonVersion := &context{}
	data, err := os.ReadFile(c.jsonPath)
	if err != nil {
		return []types.VersionKey{types.VersionRootFS, types.VersionData}
	}

	if err := json.Unmarshal(data, jsonVersion); err != nil {
		_ = os.RemoveAll(c.jsonPath)
		return []types.VersionKey{types.VersionRootFS, types.VersionData}
	}

	if jsonVersion.RootFS != c.RootFS ||
		util.Exists(filepath.Join(c.opt.ImageDir, "ext4.vhdx")) != nil {
		result = append(result, types.VersionRootFS)
	}

	if jsonVersion.Data != c.Data ||
		util.Exists(filepath.Join(c.opt.ImageDir, "data.vhdx")) != nil {
		result = append(result, types.VersionData)
	}

	return
}

func (c *context) CheckAndReplace(log *logger.Context) error {
	list := c.needUpdate()
	if len(list) == 0 {
		return nil
	}

	if slices.Contains(list, types.VersionData) {
		if err := updateData(c.opt, log); err != nil {
			return fmt.Errorf("failed to update data: %w", err)
		}
	}

	if slices.Contains(list, types.VersionRootFS) {
		if err := updateRootfs(c.opt, log); err != nil {
			return fmt.Errorf("failed to update rootfs: %w", err)
		}
	}

	if err := c.save(); err != nil {
		return fmt.Errorf("failed to save versions: %w", err)
	}

	return nil
}
