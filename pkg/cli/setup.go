// SPDX-FileCopyrightText: 2024 OOMOL, Inc. <https://www.oomol.com>
// SPDX-License-Identifier: MPL-2.0

package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/oomol-lab/ovm-win/pkg/types"
	"github.com/oomol-lab/ovm-win/pkg/util"
	"github.com/oomol-lab/ovm-win/pkg/winapi/sys"
	"golang.org/x/sync/errgroup"
)

type Context struct {
	Name              string
	LogPath           string
	ImageDir          string
	RootfsPath        string
	DistroName        string
	Version           types.Version
	IsElevatedProcess bool
	IsAdmin           bool
	RestfulEndpoint   string
	EventSocketPath   string
	CanReboot         bool
	CanEnableFeature  bool
	CanUpdateWSL      bool
	PodmanPort        int
}

func Setup() (*Context, error) {
	if err := parse(); err != nil {
		return nil, fmt.Errorf("validate flags error: %v\n", err)
	}

	ctx := &Context{}

	g := errgroup.Group{}
	g.Go(ctx.basic)
	g.Go(ctx.logPath)
	g.Go(ctx.process)
	g.Go(ctx.update)
	g.Go(ctx.port)

	return ctx, g.Wait()
}

func (c *Context) basic() error {
	c.Name = name
	c.RestfulEndpoint = `\\.\pipe\ovm-` + c.Name
	c.EventSocketPath = `\\.\pipe\` + eventNpipeName
	c.CanReboot = false
	c.DistroName = "ovm-" + c.Name
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

func (c *Context) update() error {
	p, err := filepath.Abs(imageDir)
	if err != nil {
		return fmt.Errorf("failed to get imageDir absolute path from %s: %v", imageDir, err)
	}
	if err := os.MkdirAll(p, 0755); err != nil {
		return fmt.Errorf("failed to create imageDir folder %s: %v", p, err)
	}
	c.ImageDir = p

	p, err = filepath.Abs(rootfsPath)
	if err != nil {
		return fmt.Errorf("failed to get rootfsPath absolute path from %s: %v", rootfsPath, err)
	}
	c.RootfsPath = p

	version := types.Version{}
	s := strings.Split(versions, ",")

	for _, val := range s {
		item := strings.Split(strings.TrimSpace(val), "=")
		if len(item) != 2 {
			continue
		}

		key := strings.TrimSpace(item[0])

		switch key {
		case types.VersionRootFS:
			version.RootFS = strings.TrimSpace(item[1])
		case types.VersionData:
			version.Data = strings.TrimSpace(item[1])
		}
	}

	if version.RootFS == "" {
		return fmt.Errorf("need %s in versions", types.VersionRootFS)
	}

	if version.Data == "" {
		return fmt.Errorf("need %s in versions", types.VersionData)
	}

	c.Version = version
	return nil
}

// just a random port
const podmanStartPort = 7591

func (c *Context) port() error {
	p, err := util.FindUsablePort(podmanStartPort)
	if err != nil {
		return fmt.Errorf("failed to find a usable port: %v", err)
	}

	c.PodmanPort = p

	return nil
}
