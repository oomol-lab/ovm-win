// SPDX-FileCopyrightText: 2024 OOMOL, Inc. <https://www.oomol.com>
// SPDX-License-Identifier: MPL-2.0

package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/oomol-lab/ovm-win/pkg/ipc/event"
	"github.com/oomol-lab/ovm-win/pkg/ipc/restful"
	"github.com/oomol-lab/ovm-win/pkg/logger"
	"github.com/oomol-lab/ovm-win/pkg/types"
	"github.com/oomol-lab/ovm-win/pkg/update"
	"github.com/oomol-lab/ovm-win/pkg/util"
	"github.com/oomol-lab/ovm-win/pkg/winapi/vhdx"
	"github.com/oomol-lab/ovm-win/pkg/wsl"
	"golang.org/x/sync/errgroup"
)

type RunContext struct {
	types.RunOpt
}

func RunCmd(p *types.RunOpt) *RunContext {
	r := &RunContext{
		*p,
	}
	r.RestfulEndpoint = `\\.\pipe\ovm-` + p.Name
	r.DistroName = "ovm-" + r.Name
	return r
}

func (c *RunContext) Setup() error {
	// logger
	{
		if err := setupLogPath(&c.BasicOpt); err != nil {
			return fmt.Errorf("failed to setup log path: %w", err)
		}

		if log, err := logger.New(c.LogPath, c.Name); err != nil {
			return fmt.Errorf("failed to setup log: %w", err)
		} else {
			c.Logger = log
		}
	}

	event.Setup(c.Logger, `\\.\pipe\`+c.EventNpipeName)

	if err := c.update(); err != nil {
		return fmt.Errorf("failed to update: %w", err)
	}

	if err := c.setupPort(); err != nil {
		return fmt.Errorf("failed to get port: %w", err)
	}

	if err := c.SetupSourceCodeDisk(); err != nil {
		return fmt.Errorf("failed to setup source code disk: %w", err)
	}

	return nil
}

func (c *RunContext) Start() error {
	r, err := restful.SetupRun(&c.RunOpt)
	if err != nil {
		return fmt.Errorf("failed to setup RESTful server: %w", err)
	}

	g, ctx := errgroup.WithContext(context.Background())

	g.Go(func() error {
		context.AfterFunc(ctx, func() {
			_ = r.Close()
		})

		return r.Run()
	})

	g.Go(func() error {
		return util.WaitBindPID(ctx, c.Logger, c.BindPID)
	})

	g.Go(func() error {
		return wsl.Launch(ctx, c.Logger, &c.RunOpt)
	})

	err = g.Wait()
	if c.StoppedWithAPI {
		return nil
	}
	return err
}

func (c *RunContext) update() error {
	p, err := filepath.Abs(c.ImageDir)
	if err != nil {
		return fmt.Errorf("failed to get imageDir absolute path from %s: %v", c.ImageDir, err)
	}
	if err := os.MkdirAll(p, 0755); err != nil {
		return fmt.Errorf("failed to create imageDir folder %s: %v", p, err)
	}
	c.ImageDir = p

	p, err = filepath.Abs(c.RootFSPath)
	if err != nil {
		return fmt.Errorf("failed to get rootfsPath absolute path from %s: %v", c.RootFSPath, err)
	}
	c.RootFSPath = p

	version := types.Version{}
	s := strings.Split(c.RunOpt.Version, ",")

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

	if err := update.New(&c.RunOpt, version).CheckAndReplace(); err != nil {
		return fmt.Errorf("failed to update: %w", err)
	}

	return nil
}

// just a random port
const podmanStartPort = 7591

func (c *RunContext) setupPort() error {
	p, err := util.FindUsablePort(podmanStartPort)
	if err != nil {
		return fmt.Errorf("failed to find a usable port: %v", err)
	}

	c.PodmanPort = p

	return nil
}

func (c *RunContext) SetupSourceCodeDisk() error {
	p := filepath.Join(c.ImageDir, "source_code.vhdx")
	// if source_code.vhdx already exists, skip
	_, err := os.Stat(p)
	if err == nil {
		c.Logger.Info("source code disk already exists")
		return nil
	}

	c.Logger.Info("setup source code disk")
	return vhdx.ExtractSourceCode(p)
}
