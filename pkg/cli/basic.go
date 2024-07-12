// SPDX-FileCopyrightText: 2024 OOMOL, Inc. <https://www.oomol.com>
// SPDX-License-Identifier: MPL-2.0

package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/oomol-lab/ovm-win/pkg/types"
)

func setupLogPath(c *types.BasicOpt) error {
	p, err := filepath.Abs(c.LogPath)
	if err != nil {
		return fmt.Errorf("failed to get absolute path from %s: %v", p, err)
	}

	if err := os.MkdirAll(p, 0755); err != nil {
		return fmt.Errorf("failed to create log folder %s: %v", p, err)
	}

	c.LogPath = p

	return nil
}
