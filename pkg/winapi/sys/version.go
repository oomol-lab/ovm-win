// SPDX-FileCopyrightText: 2024 OOMOL, Inc. <https://www.oomol.com>
// SPDX-License-Identifier: MPL-2.0

package sys

import (
	"github.com/oomol-lab/ovm-win/pkg/logger"
	"golang.org/x/sys/windows"
)

// 19043 == 21H1
// Although Microsoft claims that 19041 supports WSL2, it was actually supported in subsequent updates,
// not from the beginning. Version 19043 supports WSL2 from the start. For convenience, 19043 is used here.
const minBuildNumber uint32 = 19043

func SupportWSL2(log *logger.Context) bool {
	v := windows.RtlGetVersion()

	log.Infof("Current system build number is %d", v.BuildNumber)

	return v.BuildNumber >= minBuildNumber
}
