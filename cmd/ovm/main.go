package main

import (
	"fmt"
	"os"

	"github.com/oomol-lab/ovm-win/pkg/cli"
	"github.com/oomol-lab/ovm-win/pkg/logger"
	"github.com/oomol-lab/ovm-win/pkg/winapi/sys"
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

	if !opt.IsAdmin {
		log.Info("Running as non-admin")
		if err := sys.RunAsAdminWait(); err != nil {
			log.Errorf("Failed to run as admin: %v", err)
			exit(1)
		}
		log.Info("admin child process exited successfully")
	} else {
		log.Info("Running as admin")
	}

}

func exit(exitCode int) {
	logger.CloseAll()
	for _, clean := range cleans {
		clean()
	}
	os.Exit(exitCode)
}
