// SPDX-FileCopyrightText: 2024 OOMOL, Inc. <https://www.oomol.com>
// SPDX-License-Identifier: MPL-2.0

package winapi

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

// SEE_MASK Define [FMask] member for [SHELLEXECUTEINFO]
//
// [FMask]: https://learn.microsoft.com/en-us/windows/win32/api/shellapi/ns-shellapi-shellexecuteinfow#members
type SEE_MASK uint32

const (
	SEE_MASK_DEFAULT            SEE_MASK = 0x00000000
	SEE_MASK_CLASSNAME          SEE_MASK = 0x00000001
	SEE_MASK_CLASSKEY           SEE_MASK = 0x00000003
	SEE_MASK_IDLIST             SEE_MASK = 0x00000004
	SEE_MASK_INVOKEIDLIST       SEE_MASK = 0x0000000c
	SEE_MASK_ICON               SEE_MASK = 0x00000010
	SEE_MASK_HOTKEY             SEE_MASK = 0x00000020
	SEE_MASK_NOCLOSEPROCESS     SEE_MASK = 0x00000040
	SEE_MASK_CONNECTNETDRV      SEE_MASK = 0x00000080
	SEE_MASK_NOASYNC            SEE_MASK = 0x00000100
	SEE_MASK_FLAG_DDEWAIT       SEE_MASK = 0x00000100
	SEE_MASK_DOENVSUBST         SEE_MASK = 0x00000200
	SEE_MASK_FLAG_NO_UI         SEE_MASK = 0x00000400
	SEE_MASK_UNICODE            SEE_MASK = 0x00004000
	SEE_MASK_NO_CONSOLE         SEE_MASK = 0x00008000
	SEE_MASK_ASYNCOK            SEE_MASK = 0x00100000
	SEE_MASK_NOQUERYCLASSSTORE  SEE_MASK = 0x01000000
	SEE_MASK_HMONITOR           SEE_MASK = 0x00200000
	SEE_MASK_NOZONECHECKS       SEE_MASK = 0x00800000
	SEE_MASK_WAITFORINPUTIDLE   SEE_MASK = 0x02000000
	SEE_MASK_FLAG_LOG_USAGE     SEE_MASK = 0x04000000
	SEE_MASK_FLAG_HINST_IS_SITE SEE_MASK = 0x08000000
)

// SE_ERR Define [Error Code] for [SHELLEXECUTEINFO] HInstApp member
//
// [Error Code]: https://learn.microsoft.com/en-us/windows/win32/api/shellapi/ns-shellapi-shellexecuteinfow#members
type SE_ERR uint32

const (
	SE_ERR_FNF             SE_ERR = 2  // File not found
	SE_ERR_PNF             SE_ERR = 3  // Path not found
	SE_ERR_ACCESSDENIED    SE_ERR = 5  // Access denied
	SE_ERR_OOM             SE_ERR = 8  // Out of memory
	SE_ERR_DLLNOTFOUND     SE_ERR = 32 // Dynamic-link library not found.
	SE_ERR_SHARE           SE_ERR = 26 // Cannot share an open file
	SE_ERR_ASSOCINCOMPLETE SE_ERR = 27 // File association information not complete
	SE_ERR_DDETIMEOUT      SE_ERR = 28 // DDE operation timed out
	SE_ERR_DDEFAIL         SE_ERR = 29 // DDE operation failed
	SE_ERR_DDEBUSY         SE_ERR = 30 // DDE operation is busy
	SE_ERR_NOASSOC         SE_ERR = 31 // File association not available
)

// SE_ERR_MSG [SE_ERR] to string
var SE_ERR_MSG = map[SE_ERR]string{
	SE_ERR_FNF:             "File not found",
	SE_ERR_PNF:             "Path not found",
	SE_ERR_ACCESSDENIED:    "Access denied",
	SE_ERR_OOM:             "Out of memory",
	SE_ERR_DLLNOTFOUND:     "Dynamic-link library not found.",
	SE_ERR_SHARE:           "Cannot share an open file",
	SE_ERR_ASSOCINCOMPLETE: "File association information not complete",
	SE_ERR_DDETIMEOUT:      "DDE operation timed out",
	SE_ERR_DDEFAIL:         "DDE operation failed",
	SE_ERR_DDEBUSY:         "DDE operation is busy",
	SE_ERR_NOASSOC:         "File association not available",
}

// SHELLEXECUTEINFO Define Window [SHELLEXECUTEINFOW Structure]
//
// Ref: [windows-data-types] and [xaevman/win32/shell32]
//
// [SHELLEXECUTEINFOW Structure]: https://learn.microsoft.com/en-us/windows/win32/api/shellapi/ns-shellapi-shellexecuteinfow
// [windows-data-types]: https://learn.microsoft.com/en-us/windows/win32/winprog/windows-data-types
// [xaevman/win32/shell32]: https://github.com/xaevman/win32/blob/509aee64f623fc6a0578f1ea3dfbe94b3b0293cd/shell32/shell32.go#L51
type SHELLEXECUTEINFO struct {
	CbSize         uint32
	FMask          SEE_MASK
	Hwnd           windows.Handle
	LpVerb         uintptr
	LpFile         uintptr
	LpParams       uintptr
	LpDirectory    uintptr
	NShow          int
	HInstApp       windows.Handle
	LpIDList       unsafe.Pointer
	LpClass        uintptr
	HKeyClass      windows.Handle
	DwHotKey       uint32
	HIconOrMonitor windows.Handle
	HProcess       windows.Handle
}

// ShellExecuteEx Encapsulation of the Windows [ShellExecuteExW] function
//
// [ShellExecuteExW]: https://learn.microsoft.com/en-us/windows/win32/api/shellapi/nf-shellapi-shellexecuteexw
func ShellExecuteEx(info *SHELLEXECUTEINFO) (err error) {
	if ret, _, lastErr := shellExecuteEx.Call(uintptr(unsafe.Pointer(info))); ret == 0 {
		return lastErr
	}

	return nil
}
