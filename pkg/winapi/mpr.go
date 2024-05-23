// SPDX-FileCopyrightText: 2024 OOMOL, Inc. <https://www.oomol.com>
// SPDX-License-Identifier: MPL-2.0

package winapi

import (
	"errors"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

// UNIVERSAL_NAME_INFO_LEVEL
//
// Ref: https://learn.microsoft.com/en-us/windows/win32/api/winnetwk/nf-winnetwk-wnetgetuniversalnamea#parameters
const UNIVERSAL_NAME_INFO_LEVEL = 1

type universalNameInfo struct {
	lpUniversalName *uint16
}

// WNetGetUniversalName retrieves the Universal Naming Convention (UNC) path for a mapped drive.
//
// Ref: https://learn.microsoft.com/en-us/windows/win32/api/winnetwk/nf-winnetwk-wnetgetuniversalnamew
func WNetGetUniversalName(lpLocalPath string) (result string, err error) {
	size := uint32(1024)

	for {
		info := &universalNameInfo{}
		r1, _, _ := wNetGetUniversalName.Call(CStr(lpLocalPath), uintptr(UNIVERSAL_NAME_INFO_LEVEL), uintptr(unsafe.Pointer(info)), uintptr(unsafe.Pointer(&size)))
		if r1 != 0 {
			err = syscall.Errno(r1)
			if !errors.Is(err, windows.ERROR_MORE_DATA) {
				return "", err
			}
			continue
		}

		return windows.UTF16PtrToString(info.lpUniversalName), nil
	}
}
