// SPDX-FileCopyrightText: 2024 OOMOL, Inc. <https://www.oomol.com>
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"context"
	"fmt"

	"github.com/oomol-lab/ovm-win/pkg/channel"
	"github.com/oomol-lab/ovm-win/pkg/cli"
	"github.com/oomol-lab/ovm-win/pkg/ipc/event"
	"github.com/oomol-lab/ovm-win/pkg/ipc/restful"
	"github.com/oomol-lab/ovm-win/pkg/logger"
	"github.com/oomol-lab/ovm-win/pkg/update"
	"github.com/oomol-lab/ovm-win/pkg/util"
	"github.com/oomol-lab/ovm-win/pkg/winapi/sys"
	"github.com/oomol-lab/ovm-win/pkg/wsl"
	"golang.org/x/sync/errgroup"
)

var (
	opt *cli.Context
)

func init() {
	isElevated, err := sys.IsElevatedProcess()
	if err != nil {
		fmt.Println("Failed to check if the current process is an elevated child process", err)
		util.Exit(1)
	}

	// For debugging purposes, we need to redirect the console of the current process to the parent process before cli.Setup.
	if isElevated {
		if err := sys.MoveConsoleToParent(); err != nil {
			fmt.Println("Failed to move console to parent process", err)
			util.Exit(1)
		}
	}

	if ctx, err := cli.Setup(); err != nil {
		fmt.Println("Failed to setup cli", err)
		util.Exit(1)
	} else {
		opt = ctx
		opt.IsElevatedProcess = isElevated
	}
}

func newLogger() (*logger.Context, error) {
	if opt.IsElevatedProcess {
		return logger.NewWithChildProcess(opt.LogPath, opt.Name)
	}
	return logger.New(opt.LogPath, opt.Name)
}

func main() {
	log, err := newLogger()
	if err != nil {
		fmt.Println("Failed to create logger", err)
		util.Exit(1)
	}

	event.Setup(log, opt.EventSocketPath)

	ctx, cancel := context.WithCancelCause(context.Background())
	g, ctx := errgroup.WithContext(ctx)
	defer func() {
		if err := g.Wait(); err != nil {
			err = log.Errorf("Main error: %v, cause: %v", err, context.Cause(ctx))
			event.NotifyError(err)
			event.NotifyApp(event.Exit)
			util.Exit(1)
		} else {
			log.Info("Done")
			event.NotifyApp(event.Exit)
			util.Exit(0)
		}
	}()

	if !sys.SupportWSL2(log) {
		event.NotifySys(event.SystemNotSupport)
		cancel(fmt.Errorf("WSL2 is not supported on this system, need Windows 10 version 19043 or higher"))
		return
	}

	if !opt.IsElevatedProcess {
		r, err := restful.Setup(ctx, opt, log)
		if err != nil {
			cancel(fmt.Errorf("failed to setup RESTful server: %w", err))
			return
		}

		g.Go(r.Run)
	}

	wsl.Check(opt, log)

	select {
	case <-channel.ReceiveWSLEnvReady():
		log.Info("WSL environment is ready")
	case <-ctx.Done():
		return
	}

	if err := update.New(opt).CheckAndReplace(log); err != nil {
		cancel(log.Errorf("Failed to update: %v", err))
		return
	}

	g.Go(func() error {
		return wsl.Launch(ctx, log, opt)
	})

	return
}
