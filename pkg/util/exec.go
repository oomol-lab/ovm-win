// SPDX-FileCopyrightText: 2024 OOMOL, Inc. <https://www.oomol.com>
// SPDX-License-Identifier: MPL-2.0

package util

import (
	"bytes"
	"embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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

func ExecCmd(log *logger.Context, command string, args ...string) (string, error) {
	cmd := SilentCmd(command, args...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	cmd.Env = append(cmd.Env, "WSL_UTF8=1")

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

//go:embed "bin/xz.exe" "bin/liblzma-5.dll" "bin/libiconv-2.dll" "bin/libintl-8.dll"
var embeddedFiles embed.FS

func ExecExtTools(log *logger.Context, command string, args ...string) error {

	files := []string{"bin/xz.exe", "bin/liblzma-5.dll", "bin/libiconv-2.dll", "bin/libintl-8.dll"}

	tempDir, err := os.MkdirTemp("", "xz_temp")
	if err != nil {
		return fmt.Errorf("failed to create temporary directory: %v", err)
	}
	binDir := filepath.Join(tempDir, "bin")
	err = os.Mkdir(binDir, 0755)
	if err != nil {
		return fmt.Errorf("failed to create temporary bin directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	for _, file := range files {
		data, err := embeddedFiles.ReadFile(file)
		if err != nil {
			return fmt.Errorf("failed to read embedded file (%s): %v", file, err)
		}

		destPath := filepath.Join(tempDir, file)
		err = os.WriteFile(destPath, data, 0755)
		if err != nil {
			return fmt.Errorf("failed to extract file (%s): %v", destPath, err)
		}
	}

	// 构建命令
	_, _ = ExecCmd(log, (filepath.Join(tempDir, "bin/", command)), args...)

	return nil
}
