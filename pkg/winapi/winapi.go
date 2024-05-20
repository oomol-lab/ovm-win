// SPDX-FileCopyrightText: 2024 OOMOL, Inc. <https://www.oomol.com>
// SPDX-License-Identifier: MPL-2.0

package winapi

import (
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	shell32  *windows.LazyDLL
	kernel32 *windows.LazyDLL
	mpr      *windows.LazyDLL

	shellExecuteEx       *windows.LazyProc
	freeConsole          *windows.LazyProc
	attachConsole        *windows.LazyProc
	wNetGetUniversalName *windows.LazyProc
)

func init() {
	// lib
	shell32 = windows.NewLazySystemDLL("shell32.dll")
	kernel32 = windows.NewLazySystemDLL("kernel32.dll")
	mpr := windows.NewLazyDLL("mpr.dll")

	// function
	shellExecuteEx = shell32.NewProc("ShellExecuteExW")
	freeConsole = kernel32.NewProc("FreeConsole")
	attachConsole = kernel32.NewProc("AttachConsole")
	wNetGetUniversalName = mpr.NewProc("WNetGetUniversalNameW")
}

func CStr(str string) uintptr {
	s, _ := syscall.UTF16PtrFromString(str)
	return uintptr(unsafe.Pointer(s))
}
