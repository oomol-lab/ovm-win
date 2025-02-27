// SPDX-FileCopyrightText: 2024-2025 OOMOL, Inc. <https://www.oomol.com>
// SPDX-License-Identifier: MPL-2.0

package util

import "os"

func Exists(path string) error {
	_, err := os.Stat(path)
	return err
}

func Touch(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}

	defer f.Close()
	return nil
}
