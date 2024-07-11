// SPDX-FileCopyrightText: 2024 OOMOL, Inc. <https://www.oomol.com>
// SPDX-License-Identifier: MPL-2.0

package types

import "github.com/oomol-lab/ovm-win/pkg/logger"

type BasicOpt struct {
	Name            string
	LogPath         string
	EventNpipeName  string
	RestfulEndpoint string
	Logger          *logger.Context
}

type PrepareOpt struct {
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
	BindPID        int
	PodmanPort     int
	StoppedWithAPI bool

	BasicOpt
}
