// SPDX-FileCopyrightText: 2024 OOMOL, Inc. <https://www.oomol.com>
// SPDX-License-Identifier: MPL-2.0

package wsl

import (
	"github.com/oomol-lab/ovm-win/pkg/channel"
	"github.com/oomol-lab/ovm-win/pkg/cli"
	"github.com/oomol-lab/ovm-win/pkg/ipc/event"
	"github.com/oomol-lab/ovm-win/pkg/logger"
)

func Check(opt *cli.Context, log *logger.Context) {
	if isEnabled := isFeatureEnabled(log); !isEnabled {
		log.Info("WSL2 feature is not enabled")
		event.NotifySys(event.NeedEnableFeature)
		opt.CanEnableFeature = true
		return
	}

	log.Info("WSL2 feature is already enabled")

	shouldUpdate, err := shouldUpdateWSL(log)
	if err == nil && !shouldUpdate {
		log.Info("WSL2 is up to date")
		channel.NotifyWSLEnvReady()
		return
	}

	if err != nil {
		log.Warnf("Failed to check if WSL2 needs to be updated: %v", err)
	} else {
		log.Info("WSL2 needs to be updated")
	}

	event.NotifySys(event.NeedUpdateWSL)
	opt.CanUpdateWSL = true
	return
}
