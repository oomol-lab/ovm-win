// SPDX-FileCopyrightText: 2024 OOMOL, Inc. <https://www.oomol.com>
// SPDX-License-Identifier: MPL-2.0

package util

import (
	"os/exec"
	"strings"
	"syscall"
)

const (
	// https://learn.microsoft.com/en-us/windows/win32/procthread/process-creation-flags
	flagsCreateNoWindow = 0x08000000
)

func Silent(command string, args ...string) error {
	cmd := exec.Command(command, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: flagsCreateNoWindow}
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Run()
}

func SilentCmd(command string, args ...string) *exec.Cmd {
	cmd := exec.Command(command, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: flagsCreateNoWindow}
	return cmd
}

func EscapeArg(args []string) string {
	var newArgs []string
	for _, arg := range args {
		newArgs = append(newArgs, syscall.EscapeArg(arg))
	}

	return strings.Join(newArgs, " ")
}
