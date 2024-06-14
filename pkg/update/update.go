// SPDX-FileCopyrightText: 2024 OOMOL, Inc. <https://www.oomol.com>
// SPDX-License-Identifier: MPL-2.0

package update

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"github.com/oomol-lab/ovm-win/pkg/ipc/event"
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

func (c *context) needUpdate(log *logger.Context) (result []types.VersionKey) {
	jsonVersion := &context{}
	data, err := os.ReadFile(c.jsonPath)
	if err != nil {
		log.Warnf("failed to read versions.json file: %v", err)
		return []types.VersionKey{types.VersionRootFS, types.VersionData}
	}

	if err := json.Unmarshal(data, jsonVersion); err != nil {
		log.Warnf("failed to unmarshal versions.json file, json content: %s, %v", data, err)
		_ = os.RemoveAll(c.jsonPath)
		return []types.VersionKey{types.VersionRootFS, types.VersionData}
	}

	rootfsPath := filepath.Join(c.opt.ImageDir, "ext4.vhdx")
	if jsonVersion.RootFS != c.RootFS || util.Exists(rootfsPath) != nil {
		if jsonVersion.RootFS != c.RootFS {
			log.Infof("need update rootfs, because version changed: %s -> %s", jsonVersion.RootFS, c.RootFS)
		} else {
			log.Infof("need update rootfs, because rootfs not exists: %s", rootfsPath)
		}

		result = append(result, types.VersionRootFS)
	}

	dataPath := filepath.Join(c.opt.ImageDir, "data.vhdx")
	if jsonVersion.Data != c.Data || util.Exists(dataPath) != nil {
		if jsonVersion.Data != c.Data {
			log.Infof("need update data, because version changed: %s -> %s", jsonVersion.Data, c.Data)
		} else {
			log.Infof("need update data, because data not exists: %s", dataPath)
		}
		result = append(result, types.VersionData)
	}

	return
}

func (c *context) CheckAndReplace(log *logger.Context) error {
	list := c.needUpdate(log)
	if len(list) == 0 {
		log.Info("no need to update versions")
		return nil
	}

	if slices.Contains(list, types.VersionData) {
		event.Notify(event.UpdatingData)
		if err := updateData(c.opt, log); err != nil {
			return fmt.Errorf("failed to update data: %w", err)
		}
		log.Info("update data success")
	}

	if slices.Contains(list, types.VersionRootFS) {
		event.Notify(event.UpdatingRootFS)
		if err := updateRootfs(c.opt, log); err != nil {
			return fmt.Errorf("failed to update rootfs: %w", err)
		}
		log.Info("update rootfs success")
	}

	if err := c.save(); err != nil {
		return fmt.Errorf("failed to save versions: %w", err)
	}

	return nil
}
