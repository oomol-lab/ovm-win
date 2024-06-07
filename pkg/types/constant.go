// SPDX-FileCopyrightText: 2024 OOMOL, Inc. <https://www.oomol.com>
// SPDX-License-Identifier: MPL-2.0

package types

type VersionKey = string

const (
	VersionRootFS VersionKey = "rootfs"
	VersionData   VersionKey = "data"
)

type Version struct {
	RootFS string `json:"rootfs"`
	Data   string `json:"data"`
}
