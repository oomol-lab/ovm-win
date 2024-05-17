// SPDX-FileCopyrightText: 2024 OOMOL, Inc. <https://www.oomol.com>
// SPDX-License-Identifier: MPL-2.0

package sys

import (
	"fmt"
	"os"
	"syscall"

	"github.com/oomol-lab/ovm-win/pkg/winapi"
)

const ATTACH_PARENT_PROCESS = ^uintptr(0)

// MoveConsoleToParent moves the console(stdout / stderr / stdin) to the parent process
func MoveConsoleToParent() error {
	if err := winapi.FreeConsole(); err != nil {
		return fmt.Errorf("failed to free console: %v", err)
	}

	if err := winapi.AttachConsole(ATTACH_PARENT_PROCESS); err != nil {
		return fmt.Errorf("failed to attach console: %v", err)
	}

	//  Update the standard handles in the `syscall` package
	// https://github.com/golang/go/blob/go1.20.5/src/syscall/syscall_windows.go#L493-L495
	syscall.Stdin, _ = syscall.GetStdHandle(syscall.STD_INPUT_HANDLE)
	syscall.Stdout, _ = syscall.GetStdHandle(syscall.STD_OUTPUT_HANDLE)
	syscall.Stderr, _ = syscall.GetStdHandle(syscall.STD_ERROR_HANDLE)

	// Update the corresponding file objects in the `os` package
	// See: https://github.com/golang/go/blob/go1.20.5/src/os/file.go#L65-L67
	os.Stdin = os.NewFile(uintptr(syscall.Stdin), "/dev/stdin")
	os.Stdout = os.NewFile(uintptr(syscall.Stdout), "/dev/stdout")
	os.Stderr = os.NewFile(uintptr(syscall.Stderr), "/dev/stderr")

	return nil
}
