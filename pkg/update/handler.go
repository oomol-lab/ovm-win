// SPDX-FileCopyrightText: 2024 OOMOL, Inc. <https://www.oomol.com>
// SPDX-License-Identifier: MPL-2.0

package update

import (
	"fmt"
	"github.com/oomol-lab/ovm-win/pkg/cli"
	"github.com/oomol-lab/ovm-win/pkg/logger"
	"github.com/oomol-lab/ovm-win/pkg/wsl"
)

// TODO
func updateRootfs(opt *cli.Context, log *logger.Context) error {
	fmt.Println("updateRootfs")

	err := wsl.ImportDistro(nil, true, opt.DistroName, opt.ImageDir, opt.RootfsPath)
	if err != nil {
		return fmt.Errorf(err.Error())
	}
	return nil
}

// TODO
func updateDate() error {
	fmt.Println("updateDate")
	return nil
}
