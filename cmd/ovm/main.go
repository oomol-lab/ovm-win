package main

import (
	"fmt"
	"os"

	"github.com/oomol-lab/ovm-win/pkg/cli"
	"github.com/oomol-lab/ovm-win/pkg/logger"
	"github.com/oomol-lab/ovm-win/pkg/winapi/sys"
	"github.com/oomol-lab/ovm-win/pkg/wsl"
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

	if err := wsl.Install(opt, log); err != nil {
		if wsl.IsNeedReboot(err) {
			log.Info("Need reboot system")
			exit(0)
		}

		log.Error(fmt.Sprintf("Failed to install WSL2: %v", err))
		exit(1)
	} else {
		// If it is currently a child process, then its task has been completed, and we need to exit.
		if opt.IsElevatedProcess {
			exit(0)
		}
	}

	log.Info("Done")
	exit(0)

}

func exit(exitCode int) {
	logger.CloseAll()
	for _, clean := range cleans {
		clean()
	}
	os.Exit(exitCode)
}
