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

	if err := wsl.SafeSyncDisk(log, m.DistroName); err != nil {
		switch {
		case errors.Is(err, wsl.ErrDistroNotExist):
			log.Info("distro is not exist")
			break
		case errors.Is(err, wsl.ErrDistroNotRunning):
			log.Info("distro is not running")
			break
		default:
			if err := wsl.Terminate(log, m.DistroName); err != nil {
				return fmt.Errorf("cannot terminate distro %s: %w", m.DistroName, err)
			}
			log.Info("distro is terminated")
		}
	}

	// move data
	{
		dataPath := filepath.Join(m.OldImageDir, "data.vhdx")

		if err := wsl.UmountVHDX(m.Logger, dataPath); err != nil {
			return fmt.Errorf("failed to umount data: %w", err)
		}

		if err := sys.CopyFile(dataPath, filepath.Join(m.NewImageDir, "data.vhdx"), true); err != nil {
			return fmt.Errorf("failed to copy data: %w", err)
		}

		log.Info("data is copied to new dir")
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

		log.Info("distro is moved")
	}

	{
		versions := filepath.Join(m.OldImageDir, "versions.json")

		if err := sys.CopyFile(versions, filepath.Join(m.NewImageDir, "versions.json"), true); err != nil {
			return fmt.Errorf("failed to copy versions: %w", err)
		}

		log.Info("versions is copied to new dir")
	}

	if err := os.RemoveAll(m.OldImageDir); err != nil {
		log.Warnf("failed to remove old image dir: %v", err)
	}

	return nil
}
