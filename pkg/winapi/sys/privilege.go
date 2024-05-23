// SPDX-FileCopyrightText: 2024 OOMOL, Inc. <https://www.oomol.com>
// SPDX-License-Identifier: MPL-2.0

package sys

import (
	"fmt"
	"os"
	"syscall"
	"unsafe"

	"github.com/oomol-lab/ovm-win/pkg/util"
	"github.com/oomol-lab/ovm-win/pkg/winapi"
	"golang.org/x/sys/windows"
)

// IsAdmin checks if the current process is running with admin privileges
func IsAdmin() bool {
	return windows.GetCurrentProcessToken().IsElevated()
}

// RunAsAdminWait restarts the current process with admin privileges
func RunAsAdminWait() error {
	exe, cwd, err := currentEXEAndCWD()
	if err != nil {
		return fmt.Errorf("could not get current process executable and cwd: %w", err)
	}

	sei := &winapi.SHELLEXECUTEINFO{
		FMask:       winapi.SEE_MASK_NOCLOSEPROCESS,
		Hwnd:        0,
		LpVerb:      winapi.CStr("runas"),
		LpFile:      winapi.CStr(exe),
		LpParams:    winapi.CStr(util.EscapeArg(os.Args[1:])),
		LpDirectory: winapi.CStr(cwd),
		NShow:       syscall.SW_HIDE,
	}
	sei.CbSize = uint32(unsafe.Sizeof(*sei))

	if err := winapi.ShellExecuteEx(sei); err != nil {
		if message, found := winapi.SE_ERR_MSG[winapi.SE_ERR(sei.HInstApp)]; found {
			return fmt.Errorf("failed to run as admin: %s, last error: %v", message, err)
		} else {
			return fmt.Errorf("failed to run as admin, error: %v", err)
		}
	}

	handle := sei.HProcess
	defer func() {
		_ = windows.CloseHandle(handle)
	}()

	if err := waitProcessExit(handle); err != nil {
		return fmt.Errorf("failed to wait process exit: %w", err)
	}

	return nil
}

// IsElevatedProcess checks if the current process is an elevated child process created through [RunAsAdmin]
func IsElevatedProcess() (ok bool, err error) {
	if !IsAdmin() {
		return false, nil
	}

	pe, err := parentExecutable()
	if err != nil {
		return false, fmt.Errorf("could not get parent process executable: %v", err)
	}

	executable, err := os.Executable()
	if err != nil {
		return false, fmt.Errorf("could not get current process executable: %v", err)
	}

	return pe == executable, nil
}

func waitProcessExit(handle windows.Handle) error {
	e, err := windows.WaitForSingleObject(handle, syscall.INFINITE)

	switch e {
	case windows.WAIT_OBJECT_0:
		break
	case windows.WAIT_FAILED:
		return fmt.Errorf("could not wait for process exit: %v", err)
	default:
		return fmt.Errorf("could not wait for process exit: unknown error. event: %X, err: %v", e, err)
	}

	var code uint32
	if err := windows.GetExitCodeProcess(handle, &code); err != nil {
		return fmt.Errorf("could not get process exit code: %v", err)
	}
	if code != 0 {
		return fmt.Errorf("process exited with code %d", code)
	}

	return nil
}

func parentExecutable() (path string, err error) {
	h, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(os.Getppid()))
	if err != nil {
		return "", fmt.Errorf("could not open process: %v", err)
	}
	defer func() {
		_ = windows.CloseHandle(h)
	}()

	buf := make([]uint16, syscall.MAX_LONG_PATH)
	size := uint32(syscall.MAX_LONG_PATH)
	if err := windows.QueryFullProcessImageName(h, 0, &buf[0], &size); err != nil {
		return "", fmt.Errorf("could not query full process image name: %v", err)
	}

	return windows.UTF16ToString(buf[:]), nil
}

// currentEXEAndCWD returns the current process executable and current working directory.
//
// If the currently executing file is located in a UNC directory, when creating a subprocess,
// a UNC path must also be used; otherwise, the params of the subprocess will not take effect.
func currentEXEAndCWD() (exe, cwd string, err error) {
	h, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(os.Getpid()))
	if err != nil {
		return "", "", fmt.Errorf("could not open current process: %v", err)
	}
	defer func() {
		_ = windows.CloseHandle(h)
	}()

	buf := make([]uint16, syscall.MAX_LONG_PATH)
	size := uint32(syscall.MAX_LONG_PATH)
	if err := windows.QueryFullProcessImageName(h, 0, &buf[0], &size); err != nil {
		return "", "", fmt.Errorf("could not query current full process image name: %v", err)
	}

	exeStr := windows.UTF16ToString(buf[:])
	exeGo, _ := os.Executable()
	cwdGo, _ := os.Getwd()

	if !isUNC(exeStr) {
		return exeGo, cwdGo, nil
	}

	uncCWD, err := winapi.WNetGetUniversalName(cwdGo)
	if err != nil {
		return exeStr, cwdGo, nil
	}

	return exeStr, uncCWD, nil
}

func isSlash(c uint8) bool {
	return c == '\\' || c == '/'
}

// isUNC reports whether path is a UNC path.
func isUNC(path string) bool {
	return len(path) > 1 && isSlash(path[0]) && isSlash(path[1])
}
