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
	"github.com/oomol-lab/ovm-win/pkg/util/archiver"
	"github.com/oomol-lab/ovm-win/pkg/winapi/vhdx"
	"github.com/oomol-lab/ovm-win/pkg/wsl"
)

func updateRootfs(opt *cli.Context, log *logger.Context) error {
	// Remove the old distro
	{
		err := wsl.SafeSyncDisk(log, opt.DistroName)
		switch {
		case errors.Is(err, wsl.ErrDistroNotRunning), err == nil:
			if err := wsl.Unregister(log, opt.DistroName); err != nil {
				return fmt.Errorf("cannot remove old distro %s: %w", opt.DistroName, err)
			}
		case errors.Is(err, wsl.ErrDistroNotExist):
			break
		default:
			return fmt.Errorf("cannot remove old distro %s in sync disk step: %w", opt.DistroName, err)
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
		err := wsl.SafeSyncDisk(log, opt.DistroName)
		switch {
		case err == nil:
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

	// Remove the old data
	if err := wsl.UmountVHDX(log, dataPath); err != nil {
		return fmt.Errorf("failed to unmount data: %w", err)
	}

	if err := os.RemoveAll(dataPath); err != nil {
		return fmt.Errorf("failed to remove old data: %w", err)
	}

	if err := vhdx.Create(dataPath, util.DataSize(opt.Name+opt.ImageDir)); err != nil {
		return fmt.Errorf("failed to create new data: %w", err)
	}

	return nil
}
