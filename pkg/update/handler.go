// SPDX-FileCopyrightText: 2024 OOMOL, Inc. <https://www.oomol.com>
// SPDX-License-Identifier: MPL-2.0

package update

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/oomol-lab/ovm-win/pkg/cli"
	"github.com/oomol-lab/ovm-win/pkg/logger"
	"github.com/oomol-lab/ovm-win/pkg/wsl"
)

func updateRootfs(opt *cli.Context, log *logger.Context) error {
	// Remove the old distro
	{
		// TODO: we need to call wsl.Terminate?
		if ok, err := wsl.IsRegister(log, opt.DistroName); err != nil {
			return fmt.Errorf("failed to check if distro is registered: %w", err)
		} else if ok {
			if err := wsl.Unregister(log, opt.DistroName); err != nil {
				return err
			}
		}

		_ = os.RemoveAll(filepath.Join(opt.ImageDir, "ext4.vhdx"))
	}

	if err := wsl.ImportDistro(log, opt.DistroName, opt.ImageDir, opt.RootfsPath); err != nil {
		return fmt.Errorf("failed to import distro: %w", err)
	}

	return nil
}

// TODO
func updateDate() error {
	fmt.Println("updateDate")
	return nil
}
