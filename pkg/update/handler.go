// SPDX-FileCopyrightText: 2024 OOMOL, Inc. <https://www.oomol.com>
// SPDX-License-Identifier: MPL-2.0

package update

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/oomol-lab/ovm-win/pkg/cli"
	"github.com/oomol-lab/ovm-win/pkg/logger"
	"github.com/oomol-lab/ovm-win/pkg/util/archiver"
	"github.com/oomol-lab/ovm-win/pkg/wsl"
)

func updateRootfs(opt *cli.Context, log *logger.Context) error {
	// Remove the old distro
	{
		if ok, err := wsl.IsRegister(log, opt.DistroName); err != nil {
			return fmt.Errorf("failed to check if distro is registered: %w", err)
		} else if ok {
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

// TODO
func updateDate() error {
	fmt.Println("updateDate")
	return nil
}
