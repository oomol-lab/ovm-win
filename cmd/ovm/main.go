// SPDX-FileCopyrightText: 2024 OOMOL, Inc. <https://www.oomol.com>
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"context"
	"fmt"
	"os"

	ocli "github.com/oomol-lab/ovm-win/pkg/cli"
	"github.com/oomol-lab/ovm-win/pkg/ipc/event"
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
)

var (
	prepareCtx *ocli.PrepareContext
	runCtx     *ocli.RunContext
)

func cmd() error {
	command := &cli.Command{
		HideHelpCommand: true,
		Commands: []*cli.Command{
			{
				Name:  "prepare",
				Usage: "Check the System Requirements",
				Before: func(ctx context.Context, command *cli.Command) error {
					prepareCtx = ocli.PrepareCmd(&types.PrepareOpt{
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

					return prepareCtx.Setup()
				},
				Action: func(ctx context.Context, command *cli.Command) (err error) {
					if err = prepareCtx.Start(); err != nil {
						event.NotifyError(err)
					}
					return
				},
			},
			{
				Name:  "run",
				Usage: "Run the Virtual Machine",
				Before: func(ctx context.Context, command *cli.Command) error {
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
						event.NotifyError(err)
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
				Required:    true,
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
	err := cmd()
	event.NotifyExit()

	if err != nil {
		fmt.Println(err)
		util.Exit(1)
	} else {
		util.Exit(0)
	}
}
