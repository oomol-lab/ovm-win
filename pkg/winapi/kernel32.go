// SPDX-FileCopyrightText: 2024-2025 OOMOL, Inc. <https://www.oomol.com>
// SPDX-License-Identifier: MPL-2.0

package winapi

// FreeConsole detaches the calling process from its console.
//
// Ref: https://learn.microsoft.com/en-us/windows/console/freeconsole
func FreeConsole() error {
	if ret, _, lastErr := freeConsole.Call(); ret == 0 {
		return lastErr
	}

	return nil
}

// AttachConsole attaches the calling process to the console of the specified process.
//
// ^uintptr(0) can be used to attach to the parent process.
// Ref: https://learn.microsoft.com/en-us/windows/console/attachconsole
func AttachConsole(pid uintptr) error {
	if ret, _, lastErr := attachConsole.Call(pid); ret == 0 {
		return lastErr
	}

	return nil
}

func IsProcessorFeaturePresent(feature int) bool {
	ret, _, _ := isProcessorFeaturePresent.Call(uintptr(feature))
	return ret != 0
}

func ProcCopyFile(lpExistingFileName, lpNewFileName string, bFailIfExists uint32) error {
	if ret, _, lastErr := procCopyFileW.Call(CStr(lpExistingFileName), CStr(lpNewFileName), uintptr(bFailIfExists)); ret == 0 {
		return lastErr
	}

	return nil
}
