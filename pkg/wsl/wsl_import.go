package wsl

import (
	"fmt"
	"github.com/oomol-lab/ovm-win/pkg/util"
)

var RootFS string

func ImportRootFS() error {

	if RootFS == "" {
		return fmt.Errorf("RootFS not set")
	}

	if err := util.Silent("wsl", "--import", RootFS); err != nil {
		return fmt.Errorf("WSL Import failed: %v", err)
	}

	return nil
}
