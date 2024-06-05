// SPDX-FileCopyrightText: 2024 OOMOL, Inc. <https://www.oomol.com>
// SPDX-License-Identifier: MPL-2.0

package wsl

import (
	"path/filepath"
	"sync"

	"github.com/oomol-lab/ovm-win/pkg/util"
)

var (
	onceFind sync.Once
	wslPath  string
)

func Find() string {
	onceFind.Do(func() {
		var list []string

		if p, ok := util.ProgramFiles(); ok {
			list = append(list, filepath.Join(p, "WSL", "wsl.exe"))
		}

		if p, ok := util.LocalAppData(); ok {
			list = append(list, filepath.Join(p, "Microsoft", "WindowsApps", "wsl.exe"))
		}

		if p, ok := util.System32Root(); ok {
			list = append(list, filepath.Join(p, "wsl.exe"))
		}

		for _, p := range list {
			if err := util.Exists(p); err == nil {
				wslPath = p
				return
			}
		}

		wslPath = "wsl"
	})

	return wslPath
}
