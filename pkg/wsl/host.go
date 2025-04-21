// SPDX-FileCopyrightText: 2025 OOMOL, Inc. <https://www.oomol.com>
// SPDX-License-Identifier: MPL-2.0

package wsl

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/oomol-lab/ovm-win/pkg/logger"
	"github.com/oomol-lab/ovm-win/pkg/util"
)

var (
	_hostEndpoint string
)

func HostEndpoint(log *logger.Context, name string) (string, error) {
	// TODO(@BlackHole1): getHostEndpoint may be slow, so it needs to be thread-safe here
	if _hostEndpoint != "" {
		return _hostEndpoint, nil
	}

	if he, err := getHostEndpoint(log, name); err != nil {
		return "", fmt.Errorf("failed to get host endpoint: %w", err)
	} else {
		_hostEndpoint = he
	}

	log.Infof("Host endpoint is: %s", _hostEndpoint)

	return _hostEndpoint, nil
}

func isMirroredNetwork(log *logger.Context) bool {
	val, _ := NewConfig(log).GetValue("wsl2", "networkingMode")
	return val == "mirrored"
}

func getHostEndpoint(log *logger.Context, name string) (string, error) {
	if isMirroredNetwork(log) {
		return "localhost", nil
	}

	// TODO(@BlackHole1): improve wslInvoke
	newArgs := []string{"-d", name, "/bin/sh", "-c", "ip route  | grep '^default' | awk '{print $3}'"}
	cmd := util.SilentCmd(Find(), newArgs...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	cmd.Env = []string{"WSL_UTF8=1"}

	cmdStr := fmt.Sprintf("%s %s", Find(), strings.Join(newArgs, " "))

	log.Infof("Running command in distro: %s", cmdStr)
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to run command in distro: %w, %s", err, stderr.String())
	}

	return strings.TrimSpace(stdout.String()), nil
}
