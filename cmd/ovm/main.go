// SPDX-FileCopyrightText: 2024 OOMOL, Inc. <https://www.oomol.com>
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/oomol-lab/ovm-win/pkg/cli"
	"github.com/oomol-lab/ovm-win/pkg/ipc/event"
	"github.com/oomol-lab/ovm-win/pkg/ipc/restful"
	"github.com/oomol-lab/ovm-win/pkg/logger"
	"github.com/oomol-lab/ovm-win/pkg/winapi/sys"
	"github.com/oomol-lab/ovm-win/pkg/wsl"
	"golang.org/x/sync/errgroup"
)

var (
	opt    *cli.Context
	cleans []func()
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

	ctx, cancel := context.WithCancel(context.Background())
	g, ctx := errgroup.WithContext(ctx)

	if !opt.IsElevatedProcess {
		g.Go(func() error {
			return restful.Run(ctx, opt, log)
		})
	}

	event.Setup(log, opt.EventSocketPath)

	// WSL2 Check / Install / Update
	{
		if !sys.SupportWSL2(log) {
			log.Error("WSL2 is not supported on this system, need Windows 10 version 19043 or higher")
			exit(1)
		}

		if err := wsl.Install(opt, log); err != nil {
			if wsl.IsNeedReboot(err) {
				log.Info("Need reboot system")
				event.Notify(event.NeedReboot)
				exit(0)
			}

			log.Error(fmt.Sprintf("Failed to install WSL2: %v", err))
			exit(1)
		}

		shouldUpdate, err := wsl.ShouldUpdate(log)
		if err != nil {
			log.Error(fmt.Sprintf("Failed to check if WSL2 needs to be updated: %v", err))
			exit(1)
		}

		if shouldUpdate {
			if err := wsl.Update(log); err != nil {
				log.Error(fmt.Sprintf("Failed to update WSL2: %v", err))
				exit(1)
			}
			log.Info("WSL2 has been updated")
		} else {
			log.Info("WSL2 is up to date")
		}
	}

	go func() {
		time.Sleep(2 * time.Minute)
		cancel()
	}()

	if err := g.Wait(); err != nil {
		log.Errorf("Main error: %v", err)
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
	for _, clean := range cleans {
		clean()
	}
	os.Exit(exitCode)
}
