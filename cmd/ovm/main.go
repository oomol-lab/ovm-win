// SPDX-FileCopyrightText: 2024-2025 OOMOL, Inc. <https://www.oomol.com>
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	ocli "github.com/oomol-lab/ovm-win/pkg/cli"
	"github.com/oomol-lab/ovm-win/pkg/ipc/event"
	"github.com/oomol-lab/ovm-win/pkg/logger"
	"github.com/oomol-lab/ovm-win/pkg/types"
	"github.com/oomol-lab/ovm-win/pkg/util"
	"github.com/urfave/cli/v3"
)

var (
	name           string
	logPath        string
	imageDir       string
	rootFSPath     string
	versions       string
	eventNpipeName string
	bindPID        int64

	oldImageDir string
	newImageDir string
)

var (
	initCtx    *ocli.InitContext
	runCtx     *ocli.RunContext
	migrateCtx *ocli.MigrateContext
)

func cmd() error {
	command := &cli.Command{
		HideHelpCommand: true,
		Commands: []*cli.Command{
			{
				Name:  "init",
				Usage: "Check the System Requirements",
				Before: func(ctx context.Context, command *cli.Command) error {
					if eventNpipeName == "" {
						return errors.New("--event-npipe-name not specified")
					}

					initCtx = ocli.InitCmd(&types.InitOpt{
						IsElevatedProcess: false,
						CanEnableFeature:  false,
						CanReboot:         false,
						CanUpdateWSL:      false,
						BasicOpt: types.BasicOpt{
							Name:           name,
							LogPath:        logPath,
							EventNpipeName: eventNpipeName,
							BindPID:        int(bindPID),
						},
					})

					return initCtx.Setup()
				},
				Action: func(ctx context.Context, command *cli.Command) (err error) {
					if err = initCtx.Start(); err != nil {
						event.NotifyInit(event.InitError, err.Error())
					} else {
						event.NotifyInit(event.InitSuccess)
					}
					return
				},
			},
			{
				Name:  "run",
				Usage: "Run the Virtual Machine",
				Before: func(ctx context.Context, command *cli.Command) error {
					if eventNpipeName == "" {
						return errors.New("--event-npipe-name not specified")
					}

					runCtx = ocli.RunCmd(&types.RunOpt{
						DistroName: name,
						ImageDir:   imageDir,
						RootFSPath: rootFSPath,
						Version:    versions,
						BasicOpt: types.BasicOpt{
							Name:           name,
							LogPath:        logPath,
							EventNpipeName: eventNpipeName,
							BindPID:        int(bindPID),
						},
					})
					return runCtx.Setup()
				},
				Action: func(ctx context.Context, command *cli.Command) (err error) {
					if err = runCtx.Start(); err != nil {
						event.NotifyRun(event.RunError, err.Error())
					}
					return
				},
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:        "image-dir",
						Usage:       "Store disk images",
						Required:    true,
						Destination: &imageDir,
					},
					&cli.StringFlag{
						Name:        "rootfs-path",
						Usage:       "Path to rootfs image",
						Required:    true,
						Destination: &rootFSPath,
					},
					&cli.StringFlag{
						Name:        "versions",
						Usage:       "Set versions",
						Required:    true,
						Destination: &versions,
					},
				},
			},
			{
				Name:  "migrate",
				Usage: "Migrate the ovm image to the specified directory",
				Before: func(ctx context.Context, command *cli.Command) error {
					migrateCtx = ocli.MigrateCmd(&types.MigrateOpt{
						OldImageDir: oldImageDir,
						NewImageDir: newImageDir,
						BasicOpt: types.BasicOpt{
							Name:           name,
							LogPath:        logPath,
							EventNpipeName: "",
							BindPID:        0,
						},
					})
					return migrateCtx.Setup()
				},
				Action: func(ctx context.Context, command *cli.Command) error {
					if err := migrateCtx.Start(); err != nil {
						return fmt.Errorf("failed to migrate: %w", err)
					}

					return nil
				},
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:        "old-image-dir",
						Usage:       "Old image directory",
						Required:    true,
						Destination: &oldImageDir,
					},
					&cli.StringFlag{
						Name:        "new-image-dir",
						Usage:       "new image directory",
						Required:    true,
						Destination: &newImageDir,
					},
				},
			},
		},
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "name",
				Usage:       "Name of the virtual machine",
				Required:    true,
				Persistent:  true,
				Destination: &name,
			},
			&cli.StringFlag{
				Name:        "log-path",
				Usage:       "Path to the log file",
				Required:    true,
				Persistent:  true,
				Destination: &logPath,
			},
			&cli.StringFlag{
				Name:        "event-npipe-name",
				Usage:       "HTTP server established in the named pipe (such as the foo in //./pipe/foo) must implement the GET /notify?event=&message= route",
				Required:    false,
				Persistent:  true,
				Destination: &eventNpipeName,
			},
			&cli.IntFlag{
				Name:        "bind-pid",
				Usage:       "OVM will exit when the bound pid exited",
				Value:       0,
				Required:    false,
				Persistent:  true,
				Destination: &bindPID,
			},
		},
	}
	return command.Run(context.Background(), os.Args)
}

func main() {
	var log *logger.Context
	err := cmd()
	switch {
	case initCtx != nil:
		log = initCtx.Logger
		event.NotifyInit(event.InitExit)
	case runCtx != nil:
		log = runCtx.Logger
		event.NotifyRun(event.RunExit)
	case migrateCtx != nil:
		log = migrateCtx.Logger
	}

	if err != nil {
		fmt.Println(err)
		if log != nil {
			_ = log.Error(err.Error())
		}
		util.Exit(1)
	} else {
		util.Exit(0)
	}
}
