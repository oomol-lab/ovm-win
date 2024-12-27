// SPDX-FileCopyrightText: 2024 OOMOL, Inc. <https://www.oomol.com>
// SPDX-License-Identifier: MPL-2.0

package cli

import (
	"context"
	"errors"
	"fmt"

	"github.com/oomol-lab/ovm-win/pkg/channel"
	"github.com/oomol-lab/ovm-win/pkg/ipc/event"
	"github.com/oomol-lab/ovm-win/pkg/ipc/restful"
	"github.com/oomol-lab/ovm-win/pkg/logger"
	"github.com/oomol-lab/ovm-win/pkg/types"
	"github.com/oomol-lab/ovm-win/pkg/util"
	"github.com/oomol-lab/ovm-win/pkg/winapi/sys"
	"github.com/oomol-lab/ovm-win/pkg/wsl"
	"golang.org/x/sync/errgroup"
	"golang.org/x/sys/windows"
)

type PrepareContext struct {
	types.PrepareOpt
}

func PrepareCmd(p *types.PrepareOpt) *PrepareContext {
	c := &PrepareContext{
		*p,
	}
	c.RestfulEndpoint = `\\.\pipe\ovm-prepare-` + c.Name
	return c
}

func (c *PrepareContext) Setup() error {
	isElevated, err := sys.IsElevatedProcess()
	if err != nil {
		return fmt.Errorf("failed to check if the current process is an elevated child process: %w", err)
	}
	c.IsElevatedProcess = isElevated

	// logger
	{
		if err := setupLogPath(&c.BasicOpt); err != nil {
			return fmt.Errorf("failed to setup log path: %w", err)
		}

		if log, err := c.loggerInstance(); err != nil {
			return fmt.Errorf("failed to setup log: %w", err)
		} else {
			c.Logger = log
		}
	}

	c.moveConsoleToParent()

	event.Setup(c.Logger, `\\.\pipe\`+c.EventNpipeName)

	return nil
}

func (c *PrepareContext) Start() error {
	if c.IsElevatedProcess {
		_ = wsl.Install(&c.PrepareOpt)
		util.Exit(0)
	}

	if !sys.SupportWSL2(c.Logger) {
		event.NotifyPrepare(event.SystemNotSupport)
		return fmt.Errorf("WSL2 is not supported on this system, need Windows 10 version 19043 or higher")
	}

	r, err := restful.SetupPrepare(&c.PrepareOpt)
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

	wsl.Check(&c.PrepareOpt)

	select {
	case <-ctx.Done():
		return fmt.Errorf("app unexpectedly exit, because the context is done, ctx err: %v", context.Cause(ctx))
	case <-channel.ReceiveWSLEnvReady():
		wsl.CheckBIOS(&c.PrepareOpt)
		return nil
	}
}

func (c *PrepareContext) moveConsoleToParent() {
	// For debugging purposes, we need to redirect the console of the current process to the parent process
	if c.IsElevatedProcess {
		if err := sys.MoveConsoleToParent(); err != nil {
			if errors.Is(err, windows.ERROR_INVALID_HANDLE) {
				c.Logger.Info("Cannot move console to parent process, because the parent process not have a console")
			} else {
				c.Logger.Warnf("Failed to move console to parent process: %v", err)
			}
		}
	}
}

func (c *PrepareContext) loggerInstance() (*logger.Context, error) {
	if c.IsElevatedProcess {
		return logger.NewWithChildProcess(c.LogPath, "prepare-"+c.Name)
	}
	return logger.New(c.LogPath, "prepare-"+c.Name)
}
