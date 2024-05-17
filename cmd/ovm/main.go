package main

import (
	"fmt"
	"os"
	"time"

	"github.com/oomol-lab/ovm-win/pkg/winapi/sys"
)

func init() {
	isElevated, err := sys.IsElevatedProcess()
	if err != nil {
		fmt.Println("Failed to check if the current process is an elevated child process", err)
		os.Exit(1)
		return
	}

	if isElevated {
		if err := sys.MoveConsoleToParent(); err != nil {
			fmt.Println("Failed to move console to parent process", err)
			os.Exit(1)
		}
	} else {
		fmt.Println("Running as a normal process")
		return
	}
}

func main() {
	if sys.IsAdmin() {
		fmt.Println("Running as admin")
	} else {
		fmt.Println("Running as non-admin")
		if err := sys.RunAsAdminWait(); err != nil {
			fmt.Println("Failed to run as admin: ", err)
		} else {
			fmt.Println("yes. child process exited")
		}
	}

	time.Sleep(10 * time.Second)
}
