// SPDX-FileCopyrightText: 2024 OOMOL, Inc. <https://www.oomol.com>
// SPDX-License-Identifier: MPL-2.0

package wsl

import (
	"fmt"
	"path/filepath"
	"strings"
	"sync"

	"github.com/hashicorp/go-version"
	"github.com/oomol-lab/ovm-win/pkg/channel"
	"github.com/oomol-lab/ovm-win/pkg/cli"
	"github.com/oomol-lab/ovm-win/pkg/ipc/event"
	"github.com/oomol-lab/ovm-win/pkg/logger"
	"github.com/oomol-lab/ovm-win/pkg/util"
)

var (
	_onceIsFeatureEnabled sync.Once
	_isFeatureEnabled     bool
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

func existsKernel() bool {
	// from `MSI` or `Windows Update`
	if system32, ok := util.System32Root(); ok {
		kernel := filepath.Join(system32, "lxss", "tools", "kernel")
		if err := util.Exists(kernel); err == nil {
			return true
		}
	}

	// from `Microsoft Store` or `Github`
	if programFiles, ok := util.ProgramFiles(); ok {
		kernel := filepath.Join(programFiles, "WSL", "tools", "kernel")
		if err := util.Exists(kernel); err == nil {
			return true
		}
	}

	return false
}

// isInstalled Checks if the WSL2 is installed.
func isInstalled(log *logger.Context) bool {
	// If the kernel file does not exist,
	// it means that the current system has only enabled the Features without running wsl --update.
	if !existsKernel() {
		return false
	}

	if err := util.Silent(log, Find(), "--status"); err != nil {
		return false
	}

	return true
}

// isFeatureEnabled Checks if the WSL feature is enabled.
// At the same time, `set-default-version 2` will also be configured.
// The following two features need to be enabled:
//  1. `Microsoft-Windows-Subsystem-Linux`
//  2. `VirtualMachinePlatform`
func isFeatureEnabled(log *logger.Context) bool {
	_onceIsFeatureEnabled.Do(func() {
		// we cannot use the following methods for checking because these commands require administrative privileges.
		// 	1.Get-WindowsOptionalFeature -Online -FeatureName Microsoft-Windows-Subsystem-Linux
		// 	2.Get-WindowsOptionalFeature -Online -FeatureName VirtualMachinePlatform
		_isFeatureEnabled = util.Silent(log, Find(), "--set-default-version", "2") == nil
	})

	return _isFeatureEnabled
}

func wslVersion(log *logger.Context) (string, error) {
	br, err := wslExec(log, "--version")
	if err != nil {
		return "", fmt.Errorf("failed to get WSL2 version: %w", err)
	}

	r := string(br)
	wslLine := strings.Split(r, "\n")[0]
	wslLine = strings.TrimSpace(wslLine)
	offset := strings.LastIndex(wslLine, " ")
	if offset == -1 {
		return r, fmt.Errorf("failed to parse WSL2 version: %s", r)
	}

	return strings.TrimSpace(wslLine[offset+1:]), nil
}

const minVersion = "2.1.5"

func shouldUpdateWSL(log *logger.Context) (bool, error) {
	if isInstalled := isInstalled(log); !isInstalled {
		log.Info("WSL2 is not updated, ready to update")
		return true, nil
	}

	v, err := wslVersion(log)
	if err != nil {
		return false, fmt.Errorf("failed to get WSL2 version: %w", err)
	}

	log.Infof("Current WSL2 version: %s", v)
	currentVersion, err := version.NewVersion(v)
	if err != nil {
		return false, fmt.Errorf("failed to parse current WSL2 version: %w", err)
	}

	minVersion, err := version.NewVersion(minVersion)
	if err != nil {
		return false, fmt.Errorf("failed to parse min WSL2 version: %w", err)
	}

	if currentVersion.LessThan(minVersion) {
		return true, nil
	}

	return false, nil
}
