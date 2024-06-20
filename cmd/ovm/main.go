// SPDX-FileCopyrightText: 2024 OOMOL, Inc. <https://www.oomol.com>
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/oomol-lab/ovm-win/pkg/cli"
	"github.com/oomol-lab/ovm-win/pkg/ipc/event"
	"github.com/oomol-lab/ovm-win/pkg/ipc/restful"
	"github.com/oomol-lab/ovm-win/pkg/logger"
	"github.com/oomol-lab/ovm-win/pkg/update"
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
		os.Exit(1)
	}

	// For debugging purposes, we need to redirect the console of the current process to the parent process before cli.Setup.
	if isElevated {
		if err := sys.MoveConsoleToParent(); err != nil {
			fmt.Println("Failed to move console to parent process", err)
			os.Exit(1)
		}
	}

	if ctx, err := cli.Setup(); err != nil {
		fmt.Println("Failed to setup cli", err)
		os.Exit(1)
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
		exit(1)
	}

	ctx, cancel := context.WithCancelCause(context.Background())
	g, ctx := errgroup.WithContext(ctx)

	if !opt.IsElevatedProcess {
		g.Go(func() error {
			return restful.Run(ctx, opt, log)
		})
	}

	event.Setup(log, opt.EventSocketPath)

	if err := wsl.Setup(opt, log); err != nil {
		if opt.IsElevatedProcess {
			if wsl.ErrIsNeedReboot(err) {
				exit(0)
			}

			_ = log.Error(err.Error())
			exit(1)
		}

		if wsl.ErrIsNeedReboot(err) {
			log.Info("Need reboot system")
			event.Notify(event.NeedReboot)

			// Wait for the reboot event to be processed.
			// Before restarting the system,we should not perform any operations because the WSL environment is not ready yet.
			goto WAIT
		}

		cancel(err)
	}

	if ctx.Err() == nil {
		if err := update.New(opt).CheckAndReplace(log); err != nil {
			cancel(log.Errorf("Failed to update: %v", err))
		}
	}

	if ctx.Err() == nil {
		g.Go(func() error {
			return wsl.Launch(ctx, log, opt)
		})
	}

WAIT:
	if err := g.Wait(); err != nil {
		if errors.Is(err, context.Canceled) {
			err = fmt.Errorf("canceled, err: %w, reason: %w", err, context.Cause(ctx))
		}

		err = log.Errorf("Main error: %v", err)
		event.NotifyError(err)
		exit(1)
	} else {
		log.Info("Done")
		exit(0)
	}
}

func exit(exitCode int) {
	if !opt.IsElevatedProcess {
		event.Notify(event.Exit)
	}

	logger.CloseAll()
	os.Exit(exitCode)
}
