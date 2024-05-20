// SPDX-FileCopyrightText: 2024 OOMOL, Inc. <https://www.oomol.com>
// SPDX-License-Identifier: MPL-2.0

package winapi

import (
	"strings"
	"syscall"
	"unicode/utf16"
	"unsafe"
)

const universalNameInfoLevel = 1

type universalNameInfo struct {
	universalName [syscall.MAX_LONG_PATH]uint16
}

// WNetGetUniversalName retrieves the Universal Naming Convention (UNC) path for a mapped drive.
func WNetGetUniversalName(lpLocalPath string) (result string, err error) {
	var uni universalNameInfo
	var bufferSize = uint32(unsafe.Sizeof(uni))
	if r1, _, lastErr := wNetGetUniversalName.Call(CStr(lpLocalPath), uintptr(universalNameInfoLevel), uintptr(unsafe.Pointer(&uni)),
		uintptr(unsafe.Pointer(&bufferSize))); r1 != 0 {
		return "", lastErr
	}

	bufferStrings := splitStringBuffer(uni.universalName[:])
	//  There is some junk returned at the beginning of the structure. The actual UNC path starts after the first null terminator.
	return bufferStrings[1], nil
}

// From https://github.com/BishopFox/sliver/blob/2a1453ee37dcb505b212769e08fbb59c961d2a69/implant/sliver/mount/mount_windows.go#L112-L115
func splitStringBuffer(buffer []uint16) []string {
	bufferString := string(utf16.Decode(buffer))
	return strings.Split(strings.TrimRight(bufferString, "\x00"), "\x00")
}
