// SPDX-FileCopyrightText: 2025 OOMOL, Inc. <https://www.oomol.com>
// SPDX-License-Identifier: MPL-2.0

package wsl

import (
	"fmt"
	"strings"
	"sync"

	"github.com/oomol-lab/ovm-win/pkg/logger"
)

var (
	hostMux       sync.Mutex
	_hostEndpoint string
)

func HostEndpoint(log *logger.Context, name string) (string, error) {
	hostMux.Lock()
	defer hostMux.Unlock()

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

func getHostEndpoint(log *logger.Context, name string) (string, error) {
	var out string
	if err := Exec(log).SetDistro(name).SetStdout(&out).Run("ip", "route"); err != nil {
		return "", fmt.Errorf("failed to get host endpoint: %w", err)
	}

	log.Infof("get host endpoint output: %s", out)

	ip := ""
	lines := strings.Split(strings.TrimSpace(out), "\n")
	for _, l := range lines {
		if strings.HasPrefix(strings.TrimSpace(l), "default") {
			arr := strings.Fields(l)
			if len(arr) >= 3 {
				ip = arr[2]
				break
			}
		}
	}

	if ip == "" {
		return "", fmt.Errorf("failed to parse host endpoint from output: %s", out)
	}

	return ip, nil
}
