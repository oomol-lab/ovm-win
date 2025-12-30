// SPDX-FileCopyrightText: 2024-2025 OOMOL, Inc. <https://www.oomol.com>
// SPDX-License-Identifier: MPL-2.0

package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/oomol-lab/ovm-win/pkg/logger"
	"github.com/oomol-lab/ovm-win/pkg/types"
	"github.com/oomol-lab/ovm-win/pkg/winapi/sys"
	"github.com/oomol-lab/ovm-win/pkg/wsl"
)

type MigrateContext struct {
	types.MigrateOpt
}

func MigrateCmd(p *types.MigrateOpt) *MigrateContext {
	m := &MigrateContext{
		*p,
	}

	m.DistroName = "ovm-" + m.Name
	return m
}

func (m *MigrateContext) Setup() error {
	if err := setupLogPath(&m.BasicOpt); err != nil {
		return fmt.Errorf("failed to setup log path: %w", err)
	}

	if log, err := logger.New(m.LogPath, "migrate"+m.Name); err != nil {
		return fmt.Errorf("failed to setup log: %w", err)
	} else {
		m.Logger = log
	}

	if err := os.MkdirAll(m.NewImageDir, 0755); err != nil {
		return fmt.Errorf("failed to create new image dir: %w", err)
	}

	return nil
}

func (m *MigrateContext) Start() error {
	log := m.Logger

	log.Infof("Ready to migrate, from %s to %s", m.OldImageDir, m.NewImageDir)

	err := wsl.SafeSyncDisk(log, m.DistroName)
	switch {
	case errors.Is(err, wsl.ErrDistroNotExist):
		log.Info("Distro is not exist")
		break
	case errors.Is(err, wsl.ErrDistroNotRunning):
		log.Info("Distro is not running")
		break
	default:
		if err != nil {
			log.Warnf("Failed to sync disk: %w", err)
		}

		if err := wsl.Terminate(log, m.DistroName); err != nil {
			log.Warnf("Failed to terminate: %v", err)

			if err := wsl.Shutdown(log); err != nil {
				return fmt.Errorf("failed to shutdown wsl: %w", err)
			}
		}
		log.Info("Distro is terminated")
	}

	// copy data
	oldDataPath := filepath.Join(m.OldImageDir, "data.vhdx")
	{
		// After unmounting, there is no need to execute mount again.
		// The mount operation will be done automatically during the next startup.
		if err := wsl.UmountVHDX(m.Logger, oldDataPath); err != nil {
			return fmt.Errorf("failed to umount data: %w", err)
		}

		if err := sys.CopyFile(oldDataPath, filepath.Join(m.NewImageDir, "data.vhdx"), true); err != nil {
			return fmt.Errorf("failed to copy data: %w", err)
		}

		log.Info("File data.vhdx is copied to new dir")
	}
	// copy sourcecode.vhdx
	oldSourceCodeDiskPath := filepath.Join(m.OldImageDir, "sourcecode.vhdx")
	{
		if err := wsl.UmountVHDX(m.Logger, oldSourceCodeDiskPath); err != nil {
			return fmt.Errorf("failed to umount sourcecode.vhdx: %w", err)
		}

		if err := sys.CopyFile(oldSourceCodeDiskPath, filepath.Join(m.NewImageDir, "sourcecode.vhdx"), true); err != nil {
			return fmt.Errorf("failed to copy sourcecode.vhdx: %w", err)
		}

		log.Info("File sourcecode.vhdx is copied to new dir")
	}

	// copy versions.json
	oldVersions := filepath.Join(m.OldImageDir, "versions.json")
	newVersions := filepath.Join(m.NewImageDir, "versions.json")
	{
		if err := sys.CopyFile(oldVersions, newVersions, true); err != nil {
			return fmt.Errorf("failed to copy versions: %w", err)
		}

		log.Info("File versions.json is copied to new dir")
	}

	// move distro
	{
		needShutdown := false
		if err := wsl.MoveDistro(log, m.DistroName, m.NewImageDir); err != nil {
			if errors.Is(err, wsl.ErrSharingViolation) {
				needShutdown = true
			} else {
				return fmt.Errorf("failed to move distro: %w", err)
			}
		}

		if needShutdown {
			if err := wsl.Shutdown(log); err != nil {
				return fmt.Errorf("failed to shutdown wsl: %w", err)
			}

			if err := wsl.MoveDistro(log, m.DistroName, m.NewImageDir); err != nil {
				return fmt.Errorf("failed to move distro: %w", err)
			}
		}

		log.Info("Distro is moved")
	}

	if err := os.RemoveAll(oldDataPath); err != nil {
		log.Warnf("Failed to remove old data.vhdx: %v", err)
	}

	if err := os.RemoveAll(oldVersions); err != nil {
		log.Warnf("Failed to remove old versions.json: %v", err)
	}

	log.Infof("Success to migrate, from %s to %s", m.OldImageDir, m.NewImageDir)

	return nil
}
