// SPDX-FileCopyrightText: 2024 OOMOL, Inc. <https://www.oomol.com>
// SPDX-License-Identifier: MPL-2.0

package wsl

import (
	"path/filepath"

	"github.com/oomol-lab/ovm-win/pkg/util"
)

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

// IsInstalled Checks if the WSL2 is installed.
func IsInstalled() bool {
	// If the kernel file does not exist,
	// it means that the current system has only enabled the Features without running wsl --update.
	if !existsKernel() {
		return false
	}

	if err := util.Silent(Find(), "--status"); err != nil {
		return false
	}

	return true
}

// IsFeatureEnabled Checks if the WSL feature is enabled.
// At the same time, `set-default-version 2` will also be configured.
// The following two features need to be enabled:
//  1. `Microsoft-Windows-Subsystem-Linux`
//  2. `VirtualMachinePlatform`
func IsFeatureEnabled() bool {
	// we cannot use the following methods for checking because these commands require administrative privileges.
	// 	1.Get-WindowsOptionalFeature -Online -FeatureName Microsoft-Windows-Subsystem-Linux
	// 	2.Get-WindowsOptionalFeature -Online -FeatureName VirtualMachinePlatform
	return util.Silent(Find(), "--set-default-version", "2") == nil
}
