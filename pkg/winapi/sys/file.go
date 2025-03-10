// SPDX-FileCopyrightText: 2025 OOMOL, Inc. <https://www.oomol.com>
// SPDX-License-Identifier: MPL-2.0

package sys

import "github.com/oomol-lab/ovm-win/pkg/winapi"

func CopyFile(src, dist string, overwrite bool) error {
	var bFailIfExists uint32

	if overwrite {
		bFailIfExists = 0
	} else {
		bFailIfExists = 1
	}

	return winapi.ProcCopyFile(src, dist, bFailIfExists)
}
