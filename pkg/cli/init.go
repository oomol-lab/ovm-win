// SPDX-FileCopyrightText: 2024-2025 OOMOL, Inc. <https://www.oomol.com>
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

type InitContext struct {
	types.InitOpt
}

func InitCmd(p *types.InitOpt) *InitContext {
	c := &InitContext{
		*p,
	}
	c.RestfulEndpoint = `\\.\pipe\ovm-init-` + c.Name
	return c
}

func (c *InitContext) Setup() error {
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

func (c *InitContext) Start() error {
	if c.IsElevatedProcess {
		_ = wsl.Install(&c.InitOpt)
		util.Exit(0)
	}

	if !sys.SupportWSL2(c.Logger) {
		event.NotifyInit(event.SystemNotSupport)
		return fmt.Errorf("WSL2 is not supported on this system, need Windows 10 version 19043 or higher")
	}

	r, err := restful.SetupInit(&c.InitOpt)
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

	wsl.Check(&c.InitOpt)

	select {
	case <-ctx.Done():
		return fmt.Errorf("app unexpectedly exit, because the context is done, ctx err: %v", context.Cause(ctx))
	case <-channel.ReceiveWSLEnvReady():
		wsl.CheckBIOS(&c.InitOpt)
		return nil
	}
}

func (c *InitContext) moveConsoleToParent() {
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

func (c *InitContext) loggerInstance() (*logger.Context, error) {
	if c.IsElevatedProcess {
		return logger.NewWithChildProcess(c.LogPath, "init-"+c.Name)
	}
	return logger.New(c.LogPath, "init-"+c.Name)
}
