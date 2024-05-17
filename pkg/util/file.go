// SPDX-FileCopyrightText: 2024 OOMOL, Inc. <https://www.oomol.com>
// SPDX-License-Identifier: MPL-2.0

package util

import "os"

func Exists(path string) error {
	_, err := os.Stat(path)
	return err
}
