// SPDX-FileCopyrightText: 2024-2025 OOMOL, Inc. <https://www.oomol.com>
// SPDX-License-Identifier: MPL-2.0

package types

import "github.com/oomol-lab/ovm-win/pkg/logger"

type BasicOpt struct {
	Name            string
	LogPath         string
	EventNpipeName  string
	RestfulEndpoint string
	BindPID         int
	Logger          *logger.Context
}

type InitOpt struct {
	IsElevatedProcess bool
	CanReboot         bool
	CanEnableFeature  bool
	CanUpdateWSL      bool

	BasicOpt
}

type RunOpt struct {
	DistroName     string
	ImageDir       string
	RootFSPath     string
	Version        string
	PodmanPort     int
	StoppedWithAPI bool

	BasicOpt
}
