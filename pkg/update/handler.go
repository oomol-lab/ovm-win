// SPDX-FileCopyrightText: 2024-2025 OOMOL, Inc. <https://www.oomol.com>
// SPDX-License-Identifier: MPL-2.0

package update

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/oomol-lab/ovm-win/pkg/util"
	"github.com/oomol-lab/ovm-win/pkg/winapi/vhdx"
	"github.com/oomol-lab/ovm-win/pkg/wsl"
)

func (c *Context) updateRootfs() error {
	log := c.Logger

	// Remove the old distro
	{
		err := wsl.SafeSyncDisk(log, c.DistroName)
		switch {
		case errors.Is(err, wsl.ErrDistroNotRunning), err == nil:
			log.Infof("Removing old distro: %s", c.DistroName)
			if err := wsl.Unregister(log, c.DistroName); err != nil {
				return fmt.Errorf("cannot remove old distro %s: %w", c.DistroName, err)
			}
		case errors.Is(err, wsl.ErrDistroNotExist):
			break
		default:
			return fmt.Errorf("cannot remove old distro %s in sync disk step: %w", c.DistroName, err)
		}
	}

	log.Infof("Importing distro %s from %s", c.DistroName, c.RootFSPath)
	if err := wsl.ImportDistro(log, c.DistroName, c.ImageDir, c.RootFSPath); err != nil {
		return fmt.Errorf("failed to import distro: %w", err)
	}

	return nil
}

func (c *Context) updateData() error {
	log := c.Logger

	// Shutdown the distro
	{
		err := wsl.SafeSyncDisk(log, c.DistroName)
		switch {
		case err == nil:
			log.Infof("Shutting down distro: %s", c.DistroName)
			if err := wsl.Terminate(log, c.DistroName); err != nil {
				return fmt.Errorf("cannot terminate distro %s: %w", c.DistroName, err)
			}
		case errors.Is(err, wsl.ErrDistroNotExist), errors.Is(err, wsl.ErrDistroNotRunning):
			break
		default:
			return fmt.Errorf("cannot terminate distro %s in sync disk step: %w", c.DistroName, err)
		}
	}

	dataPath := filepath.Join(c.ImageDir, "data.vhdx")
	sourceCodeDiskPath := filepath.Join(c.ImageDir, "sourcecode.vhdx")

	log.Infof("Umounting data: %s, source code disk: %s", dataPath, sourceCodeDiskPath)
	if err := wsl.UmountVHDX(log, dataPath, sourceCodeDiskPath); err != nil {
		return fmt.Errorf("failed to unmount data: %w", err)
	}

	log.Infof("Removing old data: %s", dataPath)
	if err := os.RemoveAll(dataPath); err != nil {
		return fmt.Errorf("failed to remove old data: %w", err)
	}

	dataSize := util.DataSize(c.Name)
	log.Infof("Creating new data: %s, size: %d", dataPath, dataSize)
	if err := vhdx.Create(dataPath, dataSize); err != nil {
		return fmt.Errorf("failed to create new data: %w", err)
	}

	return nil
}
