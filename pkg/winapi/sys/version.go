// SPDX-FileCopyrightText: 2024 OOMOL, Inc. <https://www.oomol.com>
// SPDX-License-Identifier: MPL-2.0

package sys

import (
	"github.com/oomol-lab/ovm-win/pkg/logger"
	"golang.org/x/sys/windows"
)

// 19043 == 21H1
// 19044 == 21H2
// Although Microsoft claims that version 19041 supports WSL2, it was actually supported in later updates, not right from the start.
// Version 19043 supports WSL2 from the beginning, but tests show that the latest version of WSL2 has issues when used on 19043.
// Therefore, for convenience, version 19044 is used here.
const minBuildNumber uint32 = 19044

func SupportWSL2(log *logger.Context) bool {
	v := windows.RtlGetVersion()

	log.Infof("Current system build number is %d", v.BuildNumber)

	return v.BuildNumber >= minBuildNumber
}
