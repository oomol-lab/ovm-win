// SPDX-FileCopyrightText: 2024 OOMOL, Inc. <https://www.oomol.com>
// SPDX-License-Identifier: MPL-2.0

package wsl

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/oomol-lab/ovm-win/pkg/ipc/event"
	"github.com/oomol-lab/ovm-win/pkg/logger"
	"github.com/oomol-lab/ovm-win/pkg/types"
	"github.com/oomol-lab/ovm-win/pkg/util"
	"github.com/oomol-lab/ovm-win/pkg/util/request"
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
		if err := sys.ReRunAsAdminWait(); err != nil {
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

type item struct {
	URL    string `json:"url"`
	Sha256 string `json:"sha256"`
	Size   int64  `json:"size"`
}

// See: https://github.com/oomol/wsl-msi-s3-sync
type latest struct {
	Version string `json:"version"`
	X64     item   `json:"x64"`
	Arm64   item   `json:"arm64"`
	Date    string `json:"date"`
}

const latestURL = "https://static.oomol.com/wsl-msi/latest.json"

// Update updates WSL2(include kernel)
func Update(opt *types.PrepareOpt) error {
	log := opt.Logger

	event.NotifyPrepare(event.UpdatingWSL)

	log.Info("Downloading the latest version of WSL2...")

	ctx := context.WithValue(context.Background(), request.NoCache, true)
	ctx = context.WithValue(ctx, request.TimeOut, 6*time.Second)

	log.Info("Checking the latest version of WSL2...")

	body, err := request.Get(ctx, latestURL)
	if err != nil {
		event.NotifyPrepare(event.UpdateWSLFailed)
		return fmt.Errorf("failed to get latest version: %w", err)
	}

	var l latest
	if err := json.Unmarshal(body, &l); err != nil {
		event.NotifyPrepare(event.UpdateWSLFailed)
		return fmt.Errorf("failed to unmarshal latest version: %w", err)
	}

	log.Infof("Latest version: %s", l.Version)

	cachePath, ok := util.CachePath()
	if !ok {
		event.NotifyPrepare(event.UpdateWSLFailed)
		return fmt.Errorf("failed to get cache path")
	}
	if err := os.MkdirAll(cachePath, 0755); err != nil {
		event.NotifyPrepare(event.UpdateWSLFailed)
		return fmt.Errorf("failed to create cache path: %w", err)
	}

	msi := filepath.Join(cachePath, "wsl2.msi")

	if err := request.Download(context.Background(), log, l.X64.URL, msi, l.X64.Sha256); err != nil {
		event.NotifyPrepare(event.UpdateWSLFailed)
		return fmt.Errorf("failed to download WSL2: %w", err)
	}

	logPath, err := logger.NewOnlyCreate(opt.LogPath, opt.Name+"-update-wsl")
	if err != nil {
		return fmt.Errorf("failed to create logger in update wsl: %w", err)
	}

	if err := sys.RunAsAdminWait([]string{"msiexec", "/i", msi, "/passive", "/norestart", "/L*V", logPath}, opt.LogPath); err != nil {
		event.NotifyPrepare(event.UpdateWSLFailed)
		return fmt.Errorf("failed to update WSL2: %w", err)
	}

	opt.CanUpdateWSL = false
	event.NotifyPrepare(event.UpdateWSLSuccess)
	return nil
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
