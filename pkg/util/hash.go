// SPDX-FileCopyrightText: 2024 OOMOL, Inc. <https://www.oomol.com>
// SPDX-License-Identifier: MPL-2.0

package util

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
)

func Sha256File(p string) (r string, ok bool) {
	f, err := os.Open(p)
	if err != nil {
		return "", false
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", false
	}

	return fmt.Sprintf("%x", h.Sum(nil)), true
}
