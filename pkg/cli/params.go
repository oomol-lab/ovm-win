// SPDX-FileCopyrightText: 2024 OOMOL, Inc. <https://www.oomol.com>
// SPDX-License-Identifier: MPL-2.0

package cli

import (
	"flag"
	"fmt"
)

var (
	name           string
	logPath        string
	imageDir       string
	rootfsPath     string
	versions       string
	eventNpipeName string
)

func parse() error {
	flag.StringVar(&name, "name", "", "Name of the virtual machine")
	flag.StringVar(&logPath, "log-path", "", "Path to the log file")
	flag.StringVar(&imageDir, "image-dir", "", "Store disk images")
	flag.StringVar(&rootfsPath, "rootfs-path", "", "Path to rootfs image")
	flag.StringVar(&versions, "versions", "", "Set versions")
	flag.StringVar(&eventNpipeName, "event-npipe-name", "", "HTTP server established in the named pipe (such as the foo in //./pipe/foo) must implement the GET /notify?event=&message= route")

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

	if imageDir == "" {
		return fmt.Errorf("images-path is required in cli")
	}

	if rootfsPath == "" {
		return fmt.Errorf("rootfs-path is required in cli")
	}

	if versions == "" {
		return fmt.Errorf("versions is required in cli")
	}

	if eventNpipeName == "" {
		return fmt.Errorf("event-npipe-name is required in cli")
	}

	return nil
}
