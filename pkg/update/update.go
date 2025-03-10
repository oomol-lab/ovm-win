// SPDX-FileCopyrightText: 2024-2025 OOMOL, Inc. <https://www.oomol.com>
// SPDX-License-Identifier: MPL-2.0

package update

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"github.com/oomol-lab/ovm-win/pkg/ipc/event"
	"github.com/oomol-lab/ovm-win/pkg/util"

	"github.com/oomol-lab/ovm-win/pkg/types"
)

type Context struct {
	jsonPath string

	types.Version
	types.RunOpt
}

func New(opt *types.RunOpt, version types.Version) *Context {
	return &Context{
		jsonPath: filepath.Join(opt.ImageDir, "versions.json"),
		Version:  version,
		RunOpt:   *opt,
	}
}

func (c *Context) save() error {
	data, err := json.Marshal(c.Version)
	if err != nil {
		return fmt.Errorf("failed to marshal versions: %w", err)
	}

	if err := os.WriteFile(c.jsonPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write versions to %s: %w", c.jsonPath, err)
	}

	return nil
}

func (c *Context) needUpdate() (result []types.VersionKey) {
	log := c.Logger
	jsonVersion := &types.Version{}
	data, err := os.ReadFile(c.jsonPath)
	if err != nil {
		log.Warnf("Failed to read versions.json file: %v", err)
		return []types.VersionKey{types.VersionRootFS, types.VersionData}
	}

	if err := json.Unmarshal(data, jsonVersion); err != nil {
		log.Warnf("Failed to unmarshal versions.json file, json content: %s, %v", data, err)
		_ = os.RemoveAll(c.jsonPath)
		return []types.VersionKey{types.VersionRootFS, types.VersionData}
	}

	rootfsPath := filepath.Join(c.ImageDir, "ext4.vhdx")
	if jsonVersion.RootFS != c.RootFS || util.Exists(rootfsPath) != nil {
		if jsonVersion.RootFS != c.RootFS {
			log.Infof("Need update rootfs, because version changed: %s -> %s", jsonVersion.RootFS, c.RootFS)
		} else {
			log.Infof("Need update rootfs, because rootfs not exists: %s", rootfsPath)
		}

		result = append(result, types.VersionRootFS)
	}

	dataPath := filepath.Join(c.ImageDir, "data.vhdx")
	if jsonVersion.Data != c.Data || util.Exists(dataPath) != nil {
		if jsonVersion.Data != c.Data {
			log.Infof("Need update data, because version changed: %s -> %s", jsonVersion.Data, c.Data)
		} else {
			log.Infof("Need update data, because data not exists: %s", dataPath)
		}
		result = append(result, types.VersionData)
	}

	return
}

func (c *Context) CheckAndReplace() error {
	log := c.Logger
	list := c.needUpdate()
	if len(list) == 0 {
		log.Info("No need to update versions")
		return nil
	}

	if slices.Contains(list, types.VersionData) {
		event.NotifyRun(event.UpdatingData)
		if err := c.updateData(); err != nil {
			event.NotifyRun(event.UpdateDataFailed)
			return fmt.Errorf("failed to update data: %w", err)
		}
		event.NotifyRun(event.UpdateDataSuccess)
		log.Info("Update data success")
	}

	if slices.Contains(list, types.VersionRootFS) {
		event.NotifyRun(event.UpdatingRootFS)
		if err := c.updateRootfs(); err != nil {
			event.NotifyRun(event.UpdateRootFSFailed)
			return fmt.Errorf("failed to update rootfs: %w", err)
		}
		event.NotifyRun(event.UpdateRootFSSuccess)
		log.Info("Update rootfs success")
	}

	if err := c.save(); err != nil {
		return fmt.Errorf("failed to save versions: %w", err)
	}

	return nil
}
