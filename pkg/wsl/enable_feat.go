package wsl

import (
	"fmt"
	"github.com/oomol-lab/ovm-win/pkg/util"
)

// EnableFeatures enable feature hypervisor which needed
func EnableFeatures() error {
	if err := util.Silent("dism", "/online", "/enable-feature", "/featurename:Microsoft-Windows-Subsystem-Linux", "/all", "/norestart"); err != nil {
		return fmt.Errorf("dism enable Microsoft-Windows-Subsystem-Linux feature failed")
	}

	if err := util.Silent("dism", "/online", "/enable-feature", "/featurename:VirtualMachinePlatform", "/all", "/norestart"); err != nil {
		return fmt.Errorf("dism enable VirtualMachinePlatform feature failed")
	}

	return nil
}
