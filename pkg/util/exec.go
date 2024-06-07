// SPDX-FileCopyrightText: 2024 OOMOL, Inc. <https://www.oomol.com>
// SPDX-License-Identifier: MPL-2.0

package util

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
	"syscall"

	"github.com/oomol-lab/ovm-win/pkg/logger"
)

const (
	// https://learn.microsoft.com/en-us/windows/win32/procthread/process-creation-flags
	flagsCreateNoWindow = 0x08000000
)

func Silent(log *logger.Context, command string, args ...string) error {
	cmd := SilentCmd(command, args...)
	cmd.Stdout = nil
	cmd.Stderr = nil

	log.Infof("Running command: %s %s", command, strings.Join(args, " "))
	return cmd.Run()
}

func SilentCmd(command string, args ...string) *exec.Cmd {
	cmd := exec.Command(command, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: flagsCreateNoWindow}
	return cmd
}

func Exec(log *logger.Context, command string, args ...string) (string, error) {
	cmd := SilentCmd(command, args...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	log.Infof("Running command: %s %s", command, strings.Join(args, " "))
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("`%v %v` failed: %v %v (%v)", command, strings.Join(args, " "), stderr.String(), stdout.String(), err)
	}

	return stdout.String(), nil
}

func EscapeArg(args []string) string {
	var newArgs []string
	for _, arg := range args {
		newArgs = append(newArgs, syscall.EscapeArg(arg))
	}

	return strings.Join(newArgs, " ")
}
