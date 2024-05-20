// SPDX-FileCopyrightText: 2024 OOMOL, Inc. <https://www.oomol.com>
// SPDX-License-Identifier: MPL-2.0

package cli

import (
	"flag"
	"fmt"
)

var (
	name    string
	logPath string
)

func parse() error {
	flag.StringVar(&name, "name", "", "Name of the virtual machine")
	flag.StringVar(&logPath, "log-path", "", "Path to the log file")

	flag.Parse()

	return validate()
}

func validate() error {
	if name == "" {
		return fmt.Errorf("name is required in cli")
	}

	if logPath == "" {
		return fmt.Errorf("log-path is required in cli")
	}

	return nil
}
