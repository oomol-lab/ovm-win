// SPDX-FileCopyrightText: 2024 OOMOL, Inc. <https://www.oomol.com>
// SPDX-License-Identifier: MPL-2.0

package update

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/oomol-lab/ovm-win/pkg/cli"
	"github.com/oomol-lab/ovm-win/pkg/logger"
	"github.com/oomol-lab/ovm-win/pkg/util"
	"github.com/oomol-lab/ovm-win/pkg/winapi/vhdx"
	"github.com/oomol-lab/ovm-win/pkg/wsl"
)

func updateRootfs(opt *cli.Context, log *logger.Context) error {
	// Remove the old distro
	{
		err := wsl.SafeSyncDisk(log, opt.DistroName)
		switch {
		case errors.Is(err, wsl.ErrDistroNotRunning), err == nil:
			log.Infof("removing old distro: %s", opt.DistroName)
			if err := wsl.Unregister(log, opt.DistroName); err != nil {
				return fmt.Errorf("cannot remove old distro %s: %w", opt.DistroName, err)
			}
		case errors.Is(err, wsl.ErrDistroNotExist):
			break
		default:
			return fmt.Errorf("cannot remove old distro %s in sync disk step: %w", opt.DistroName, err)
		}
	}

	log.Infof("importing distro %s from %s", opt.DistroName, opt.RootfsPath)
	if err := wsl.ImportDistro(log, opt.DistroName, opt.ImageDir, opt.RootfsPath); err != nil {
		return fmt.Errorf("failed to import distro: %w", err)
	}

	return nil
}

func updateData(opt *cli.Context, log *logger.Context) error {
	// Shutdown the distro
	{
		err := wsl.SafeSyncDisk(log, opt.DistroName)
		switch {
		case err == nil:
			log.Infof("shutting down distro: %s", opt.DistroName)
			if err := wsl.Terminate(log, opt.DistroName); err != nil {
				return fmt.Errorf("cannot terminate distro %s: %w", opt.DistroName, err)
			}
		case errors.Is(err, wsl.ErrDistroNotExist), errors.Is(err, wsl.ErrDistroNotRunning):
			break
		default:
			return fmt.Errorf("cannot terminate distro %s in sync disk step: %w", opt.DistroName, err)
		}
	}

	dataPath := filepath.Join(opt.ImageDir, "data.vhdx")

	log.Infof("umounting data: %s", dataPath)
	if err := wsl.UmountVHDX(log, dataPath); err != nil {
		return fmt.Errorf("failed to unmount data: %w", err)
	}

	log.Infof("removing old data: %s", dataPath)
	if err := os.RemoveAll(dataPath); err != nil {
		return fmt.Errorf("failed to remove old data: %w", err)
	}

	dataSize := util.DataSize(opt.Name + opt.ImageDir)
	log.Infof("creating new data: %s, size: %d", dataPath, dataSize)
	if err := vhdx.Create(dataPath, dataSize); err != nil {
		return fmt.Errorf("failed to create new data: %w", err)
	}

	return nil
}
