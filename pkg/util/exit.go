// SPDX-FileCopyrightText: 2024 OOMOL, Inc. <https://www.oomol.com>
// SPDX-License-Identifier: MPL-2.0

package util

import (
	"os"

	"github.com/oomol-lab/ovm-win/pkg/channel"
	"github.com/oomol-lab/ovm-win/pkg/logger"
)

var list []func()

func RegisteredExitFuncs(f func()) {
	list = append(list, f)
}

func Exit(exitCode int) {
	for _, f := range list {
		f()
	}
	channel.Close()
	logger.CloseAll()
	os.Exit(exitCode)
}
