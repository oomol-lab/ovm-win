// SPDX-FileCopyrightText: 2024 OOMOL, Inc. <https://www.oomol.com>
// SPDX-License-Identifier: MPL-2.0

package util

import (
	"os"
	"path/filepath"

	"golang.org/x/sys/windows"
)

func LocalAppData() (string, bool) {
	if p := os.Getenv("LOCALAPPDATA"); p != "" {
		return p, true
	}

	if user := os.Getenv("USERPROFILE"); user != "" {
		return filepath.Join(user, "AppData", "Local"), true
	}

	return "", false
}

func System32Root() (string, bool) {
	if p := os.Getenv("SystemRoot"); p != "" {
		return filepath.Join(p, "System32"), true
	}

	if p, err := windows.GetSystemDirectory(); err == nil {
		return p, true
	}

	return `C:\Windows\System32`, false
}

func ProgramFiles() (string, bool) {
	if p := os.Getenv("ProgramFiles"); p != "" {
		return p, true
	}

	return `C:\Program Files`, false
}

func CachePath() (string, bool) {
	if p := os.Getenv("LOCALAPPDATA"); p != "" {
		return filepath.Join(p, "ovm", "Cache"), true
	}

	if p, err := windows.KnownFolderPath(windows.FOLDERID_LocalAppData, windows.KF_FLAG_DEFAULT); err == nil {
		return filepath.Join(p, "ovm", "Cache"), true
	}

	return "", false
}
