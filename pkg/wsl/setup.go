// SPDX-FileCopyrightText: 2024 OOMOL, Inc. <https://www.oomol.com>
// SPDX-License-Identifier: MPL-2.0

package wsl

import (
	"fmt"

	"github.com/oomol-lab/ovm-win/pkg/cli"
	"github.com/oomol-lab/ovm-win/pkg/ipc/event"
	"github.com/oomol-lab/ovm-win/pkg/logger"
	"github.com/oomol-lab/ovm-win/pkg/winapi/sys"
)

func Setup(opt *cli.Context, log *logger.Context) error {
	if !sys.SupportWSL2(log) {
		event.Notify(event.SystemNotSupport)
		return fmt.Errorf("WSL2 is not supported on this system, need Windows 10 version 19043 or higher")
	}

	if err := installWSL(opt, log); err != nil {
		if ErrIsNeedReboot(err) {
			log.Info("Need reboot system")
			event.Notify(event.NeedReboot)
			return err
		}

		return fmt.Errorf("failed to install WSL2: %w", err)
	}

	shouldUpdate, err := wslShouldUpdate(log)
	if err != nil {
		return fmt.Errorf("failed to check if WSL2 needs to be updated: %w", err)
	}

	if shouldUpdate {
		if err := wslUpdate(log); err != nil {
			return fmt.Errorf("failed to update WSL2: %w", err)
		}
		log.Info("WSL2 has been updated")
	} else {
		log.Info("WSL2 is up to date")
	}

	return nil
}
