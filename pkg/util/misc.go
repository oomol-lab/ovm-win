// SPDX-FileCopyrightText: 2024 OOMOL, Inc. <https://www.oomol.com>
// SPDX-License-Identifier: MPL-2.0

package util

import "hash/fnv"

const initSize uint64 = 301 * 1024 * 1024 * 1024

// DataSize
//
// To determine the size of generating data.vhdx, so that in wsl can determine /dev/sdN based on the size.
// The sizes obtained by different names are different.
func DataSize(name string) uint64 {
	offset := 512 * generateNumberFNV(name)
	return initSize - uint64(offset)
}

// generateNumberFNV Given a string, generate numbers from 1 to 5000.
func generateNumberFNV(s string) uint32 {
	hash := fnv.New32a()
	_, _ = hash.Write([]byte(s))
	return hash.Sum32()%50000 + 1
}
