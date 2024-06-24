// SPDX-FileCopyrightText: 2024 OOMOL, Inc. <https://www.oomol.com>
// SPDX-License-Identifier: MPL-2.0

package wsl

import (
	"fmt"

	"github.com/oomol-lab/ovm-win/pkg/channel"
	"github.com/oomol-lab/ovm-win/pkg/cli"
	"github.com/oomol-lab/ovm-win/pkg/logger"
)

// Setup sets up WSL2 environment
func Setup(opt *cli.Context, log *logger.Context) (err error) {
	if isEnabled := isFeatureEnabled(log); !isEnabled {
		log.Info("WSL2 feature is not enabled")

		if err := Install(opt, log); err != nil {
			return fmt.Errorf("failed to install WSL2: %w", err)
		}

		return nil
	}

	log.Info("WSL2 feature is already enabled")

	shouldUpdate, err := shouldUpdateWSL(log)
	if err != nil {
		opt.CanUpdateWSL = true
		return fmt.Errorf("failed to check if WSL2 needs to be updated: %w", err)
	}

	if shouldUpdate {
		if err := Update(opt, log); err != nil {
			return fmt.Errorf("failed to update WSL2: %w", err)
		}
		log.Info("WSL2 has been updated")
	} else {
		log.Info("WSL2 is up to date")
	}

	channel.NotifyWSLEnvReady()
	return nil
}
