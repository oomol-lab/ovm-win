// SPDX-FileCopyrightText: 2024 OOMOL, Inc. <https://www.oomol.com>
// SPDX-License-Identifier: MPL-2.0

package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/oomol-lab/ovm-win/pkg/winapi/sys"
	"golang.org/x/sync/errgroup"
)

type Context struct {
	Name              string
	LogPath           string
	IsElevatedProcess bool
	IsAdmin           bool
	RestfulEndpoint   string
	EventSocketPath   string
	CanReboot         bool
}

func Setup() (*Context, error) {
	if err := parse(); err != nil {
		fmt.Printf("validate flags error: %v\n", err)
		return nil, fmt.Errorf("validate flags error: %v\n", err)
	}

	ctx := &Context{}

	g := errgroup.Group{}
	g.Go(ctx.basic)
	g.Go(ctx.logPath)
	g.Go(ctx.process)

	return ctx, g.Wait()
}

func (c *Context) basic() error {
	c.Name = name
	c.RestfulEndpoint = `\\.\pipe\ovm-` + c.Name
	c.EventSocketPath = `\\.\pipe\` + eventNpipeName
	c.CanReboot = false
	return nil
}

func (c *Context) logPath() error {
	p, err := filepath.Abs(logPath)
	if err != nil {
		return fmt.Errorf("failed to get absolute path from %s: %v", logPath, err)
	}

	if err := os.MkdirAll(p, 0755); err != nil {
		return fmt.Errorf("failed to create log folder %s: %v", p, err)
	}

	c.LogPath = p
	return nil
}

func (c *Context) process() error {
	c.IsAdmin = sys.IsAdmin()

	return nil
}
