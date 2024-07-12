// SPDX-FileCopyrightText: 2024 OOMOL, Inc. <https://www.oomol.com>
// SPDX-License-Identifier: MPL-2.0

package wsl

import (
	"errors"
	"fmt"
	"os/exec"
	"time"

	"github.com/oomol-lab/ovm-win/pkg/ipc/event"
	"github.com/oomol-lab/ovm-win/pkg/logger"
	"github.com/oomol-lab/ovm-win/pkg/types"
	"github.com/oomol-lab/ovm-win/pkg/util"
	"github.com/oomol-lab/ovm-win/pkg/winapi/sys"
)

// Install installs WSL2 feature
//
// Enable feature need admin privileges and reboot
func Install(opt *types.PrepareOpt) error {
	log := opt.Logger
	if !opt.IsElevatedProcess {
		event.NotifyPrepare(event.EnableFeaturing)
	}

	if !sys.IsAdmin() {
		log.Info("Current process is not running with admin privileges, will open a new process with admin privileges")
		if err := sys.RunAsAdminWait(); err != nil {
			event.NotifyPrepare(event.EnableFeatureFailed)
			return fmt.Errorf("failed to run as admin: %w", err)
		}

		log.Info("Admin process already successfully executed and exited")
		opt.CanEnableFeature = false
		opt.CanReboot = true
		event.NotifyPrepare(event.EnableFeatureSuccess)
		event.NotifyPrepare(event.NeedReboot)
		return nil
	}

	log.Info("Ready to enable WSL2 feature")
	if err := doEnableFeature(opt); err != nil {
		wrapperErr := fmt.Errorf("failed to enable WSL2 feature: %w", err)

		if opt.IsElevatedProcess {
			_ = log.Errorf(wrapperErr.Error())
			util.Exit(1)
		}

		event.NotifyPrepare(event.EnableFeatureFailed)
		return wrapperErr
	}

	log.Info("WSL2 feature enabled successfully, need reboot system")

	if opt.IsElevatedProcess {
		util.Exit(0)
	}

	opt.CanEnableFeature = false
	opt.CanReboot = true
	event.NotifyPrepare(event.EnableFeatureSuccess)
	event.NotifyPrepare(event.NeedReboot)
	return nil
}

func doEnableFeature(opt *types.PrepareOpt) error {
	log := opt.Logger
	logPath, err := logger.NewOnlyCreate(opt.LogPath, opt.Name+"-dism")
	if err != nil {
		return fmt.Errorf("failed to create logger in dism: %w", err)
	}

	logParams := fmt.Sprintf("/logpath:%s", logPath)
	logLevel := "/loglevel:4"

	if err := util.Silent(log, "dism", "/online", "/enable-feature", "/featurename:Microsoft-Windows-Subsystem-Linux", "/all", "/norestart"); isMsiErr(err) {
		return fmt.Errorf("dism enable Microsoft-Windows-Subsystem-Linux feature failed: %w", err)
	}

	if err := util.Silent(log, "dism", "/online", "/enable-feature", "/featurename:VirtualMachinePlatform", "/all", "/norestart", logParams, logLevel); isMsiErr(err) {
		return fmt.Errorf("dism enable VirtualMachinePlatform feature failed: %w", err)
	}

	return nil
}

// Update updates WSL2(include kernel)
func Update(opt *types.PrepareOpt) error {
	log := opt.Logger
	log.Info("Updating WSL2...")

	event.NotifyPrepare(event.UpdatingWSL)

	backoff := 500 * time.Millisecond
	tryCount := 3
	for i := 1; i <= tryCount; i++ {
		err := util.Silent(log, Find(), "--update")
		if err == nil {
			opt.CanUpdateWSL = false
			log.Info("WSL2 has been updated")
			event.NotifyPrepare(event.UpdateWSLSuccess)
			return nil
		}

		var eerr *exec.ExitError
		if errors.As(err, &eerr) {
			log.Warnf("Failed to update WSL2: %v, exit code: %d, retry %d/%d", err, eerr.ExitCode(), i, tryCount)
		} else {
			log.Warnf("Failed to update WSL2: %v, retry %d/%d", err, i, tryCount)
		}

		time.Sleep(backoff)
		backoff *= 2
	}

	event.NotifyPrepare(event.UpdateWSLFailed)
	return fmt.Errorf("failed to update WSL2")
}

const (
	msiErrorSuccess                = 0
	msiErrorSuccessRebootInitiated = 1641
	msiErrorSuccessRebootRequired  = 3010
)

// isMsiErr checks if the error is an MSI error.
//
// Need skip 1641 and 3010, reason see: https://learn.microsoft.com/en-us/windows/win32/msi/error-codes
func isMsiErr(err error) bool {
	if err == nil {
		return false
	}

	var eerr *exec.ExitError
	if errors.As(err, &eerr) {
		switch eerr.ExitCode() {
		case msiErrorSuccess:
		case msiErrorSuccessRebootInitiated:
		case msiErrorSuccessRebootRequired:
			return false
		}
	}

	return true
}
