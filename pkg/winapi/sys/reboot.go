// SPDX-FileCopyrightText: 2024 OOMOL, Inc. <https://www.oomol.com>
// SPDX-License-Identifier: MPL-2.0

package sys

import (
	"fmt"

	"github.com/Microsoft/go-winio"
	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

// fast restart
// ref: https://learn.microsoft.com/en-us/windows/win32/api/winuser/nf-winuser-exitwindowsex#parameters
const flags = windows.EWX_HYBRID_SHUTDOWN | windows.EWX_REBOOT | windows.EWX_RESTARTAPPS | windows.EWX_FORCEIFHUNG

// "Application: Installation (Planned)" A planned restart or shutdown to perform application installation.
// ref: https://learn.microsoft.com/en-us/windows/win32/shutdown/system-shutdown-reason-codes
const reason = windows.SHTDN_REASON_MAJOR_APPLICATION | windows.SHTDN_REASON_MINOR_INSTALLATION | windows.SHTDN_REASON_FLAG_PLANNED

// ref: https://learn.microsoft.com/en-us/windows/win32/secauthz/privilege-constants#constants
const privilege = "SeShutdownPrivilege"

// Reboot reboots the system
//
// Ref: https://learn.microsoft.com/en-us/windows/win32/api/winuser/nf-winuser-exitwindowsex
func Reboot() error {
	err := winio.RunWithPrivilege(privilege, func() error {
		if err := windows.ExitWindowsEx(flags, reason); err != nil {
			return fmt.Errorf("execute ExitWindowsEx to reboot system failed: %w", err)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("cannot reboot system: %w", err)
	}

	return nil
}

const registryRunOncePath = `Software\Microsoft\Windows\CurrentVersion\RunOnce`

// RunOnce commands to run after the next system startup
func RunOnce(launchPath string) error {
	// no administrator privileges required to modify HKCU
	key, _, err := registry.CreateKey(registry.CURRENT_USER, registryRunOncePath, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("failed to create/open registry key: %w", err)
	}

	defer func() {
		_ = key.Close()
	}()

	if err := key.SetExpandStringValue("ovm", launchPath); err != nil {
		return fmt.Errorf("failed to set registry value: %w", err)
	}

	return nil
}
