// SPDX-FileCopyrightText: 2024 OOMOL, Inc. <https://www.oomol.com>
// SPDX-License-Identifier: MPL-2.0

package update

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/oomol-lab/ovm-win/pkg/cli"
	"github.com/oomol-lab/ovm-win/pkg/logger"
	"github.com/oomol-lab/ovm-win/pkg/util"
	"github.com/oomol-lab/ovm-win/pkg/util/archiver"
	"github.com/oomol-lab/ovm-win/pkg/winapi/vhdx"
	"github.com/oomol-lab/ovm-win/pkg/wsl"
)

func updateRootfs(opt *cli.Context, log *logger.Context) error {
	// Remove the old distro
	{
		if ok, err := wsl.IsRegister(log, opt.DistroName); err != nil {
			return fmt.Errorf("failed to check if distro is registered in update rootfs: %w", err)
		} else if ok {
			// To prevent data from not being written to the disk(data.vhdx), we perform a shutdown before deleting the rootfs.
			_ = wsl.Terminate(log, opt.DistroName)
			if err := wsl.Unregister(log, opt.DistroName); err != nil {
				return fmt.Errorf("cannot remove old distro %s: %w", opt.DistroName, err)
			}
		}
	}

	t, err := os.MkdirTemp(os.TempDir(), "ovm-")
	if err != nil {
		return fmt.Errorf("failed to create temp dir, %s: %w", t, err)
	}
	defer func() {
		_ = os.RemoveAll(t)
	}()

	tar := filepath.Join(t, "ovm.tar")
	if err := archiver.Zstd(opt.RootfsPath, tar, true); err != nil {
		return fmt.Errorf("failed to decompress rootfs: %w", err)
	}

	if err := wsl.ImportDistro(log, opt.DistroName, opt.ImageDir, tar); err != nil {
		return fmt.Errorf("failed to import distro: %w", err)
	}

	return nil
}

func updateData(opt *cli.Context, log *logger.Context) error {
	// Shutdown the distro
	{
		// We do not care whether the distro is running; as long as it exists, we will execute the shutdown.
		if ok, err := wsl.IsRegister(log, opt.DistroName); err != nil {
			return fmt.Errorf("failed to check if distro is running in update data: %w", err)
		} else if ok {
			if err := wsl.Terminate(log, opt.DistroName); err != nil {
				return fmt.Errorf("cannot terminate distro %s: %w", opt.DistroName, err)
			}
		}
	}

	dataPath := filepath.Join(opt.ImageDir, "data.vhdx")

	if err := os.RemoveAll(dataPath); err != nil {
		return fmt.Errorf("failed to remove old data: %w", err)
	}

	if err := vhdx.Create(dataPath, util.DataSize(opt.Name)); err != nil {
		return fmt.Errorf("failed to create new data: %w", err)
	}

	return nil
}
