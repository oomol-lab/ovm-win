// SPDX-FileCopyrightText: 2024-2025 OOMOL, Inc. <https://www.oomol.com>
// SPDX-License-Identifier: MPL-2.0

package wsl

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hashicorp/go-version"
	"github.com/oomol-lab/ovm-win/pkg/channel"
	"github.com/oomol-lab/ovm-win/pkg/ipc/event"
	"github.com/oomol-lab/ovm-win/pkg/logger"
	"github.com/oomol-lab/ovm-win/pkg/types"
	"github.com/oomol-lab/ovm-win/pkg/util"
	"github.com/oomol-lab/ovm-win/pkg/winapi/sys"
)

func Check(ctx context.Context, opt *types.InitOpt) {
	if ok := checkVersion(ctx, opt); !ok {
		return
	}

	if ok := checkFeature(ctx, opt); !ok {
		return
	}

	if ok := checkBIOS(ctx, opt); !ok {
		return
	}

	// Must be placed last, as this could potentially cause WSL to shut down
	if ok := checkWSLConfig(ctx, opt); !ok {
		return
	}

	return
}

func checkFeature(ctx context.Context, opt *types.InitOpt) bool {
	log := opt.Logger

	if isEnabled := isFeatureEnabled(log); isEnabled {
		log.Info("WSL2 feature is already enabled")
		return true
	}

	log.Info("WSL2 feature is not enabled")
	event.NotifyInit(event.NeedEnableFeature)
	opt.CanEnableFeature = true

	<-ctx.Done()
	return false
}

func checkVersion(ctx context.Context, opt *types.InitOpt) bool {
	log := opt.Logger

	if !shouldUpdateWSL(log) {
		log.Info("WSL2 is up to date")
		return true
	}

	event.NotifyInit(event.NeedUpdateWSL)
	opt.CanUpdateWSL = true

	select {
	case <-ctx.Done():
		log.Warnf("Cancel waiting wsl update, ctx is done: %v", context.Cause(ctx))
		return false
	case <-channel.ReceiveWSLUpdated():
		log.Info("WSL updated")
		return true
	}
}

func checkBIOS(ctx context.Context, opt *types.InitOpt) bool {
	log := opt.Logger

	if list, err := getAllWSLDistros(log, false); err == nil && len(list) != 0 {
		log.Info("Exist WSL distros, BIOS may support virtualization")
		return true
	}

	if isSupportedVirtualization(log) {
		log.Info("Virtualization is supported")
		return true
	}

	if isWillReportExpectedErrorInMountVHDX(log, opt) {
		log.Info("Expected error in mount vhdx, BIOS may support virtualization")
		return true
	}

	log.Info("Virtualization is not supported")
	event.NotifyInit(event.NotSupportVirtualization)

	<-ctx.Done()
	return false
}

const (
	FIX_WSLCONFIG_AUTO = iota
	FIX_WSLCONFIG_OPEN
	FIX_WSLCONFIG_SKIP
)

const (
	skipWslconfigCheckFileSuffix = "_check-wslconfig.skip"
)

func checkWSLConfig(ctx context.Context, opt *types.InitOpt) bool {
	log := opt.Logger

	if configPath, ok := util.ConfigPath(); ok {
		skipPath := filepath.Join(configPath, fmt.Sprintf("%s%s", opt.Name, skipWslconfigCheckFileSuffix))

		if ok && util.Exists(skipPath) == nil {
			log.Info("WSL config check skipped")
			return true
		}
	} else {
		log.Warn("Failed to get OVM config path")
	}

	if exist := NewConfig(log).ExistIncompatible(); !exist {
		log.Info("WSL2 config is compatible")
		return true
	}

	event.NotifyInit(event.WSLConfigMaybeIncompatible)
	opt.CanFixWSLConfig = true

	select {
	case <-ctx.Done():
		log.Warnf("cancel waiting fix wsl config, ctx is done: %v", context.Cause(ctx))
		return false
	case flag := <-channel.ReceiveWSLConfigUpdated():
		log.Info("WSL config updated")

		if flag == FIX_WSLCONFIG_OPEN {
			<-channel.ReceiveWSLShutdown()
		}

		return true
	}
}

func SkipConfigCheck(opt *types.InitOpt) {
	configPath, ok := util.ConfigPath()
	if !ok {
		opt.Logger.Warn("Failed to get OVM config path")
		return
	}

	skipPath := filepath.Join(configPath, fmt.Sprintf("%s%s", opt.Name, skipWslconfigCheckFileSuffix))

	if err := util.Touch(skipPath); err != nil {
		opt.Logger.Warnf("Failed to touch skip file: %v", err)
	}
}

func isSupportedVirtualization(log *logger.Context) bool {
	vf, slat := sys.IsSupportedVirtualization()
	if !slat {
		log.Warn("SLAT is not supported")
	}

	if !vf {
		log.Warn("VT-x is not supported")
	}

	// If the CPU does not support SLAT, WSL2 cannot be started (but WSL1 can be started).
	// In modern CPUs, almost all CPUs support SLAT.
	// It is not possible to strictly determine this through `vf && slat`, because in VMware, SLAT is always false (even if "Virtualize Intel VT-x/EPT or AMD-V/RVI" is checked).
	// See:
	// 		https://github.com/microsoft/WSL/issues/4709
	// 		https://www.reddit.com/r/bashonubuntuonwindows/comments/izf4qp/cpus_without_slat_capability_cant_run_wsl_2/
	return vf
}

func isWillReportExpectedErrorInMountVHDX(log *logger.Context, opt *types.InitOpt) bool {
	tempVhdx := filepath.Join(os.TempDir(), fmt.Sprintf("ovm-win-%s-%s.vhdx", opt.Name, util.RandomString(5)))
	defer func() {
		os.RemoveAll(tempVhdx)
	}()

	_, err := wslExec(log, "--mount", "--bare", "--vhd", tempVhdx)
	if err == nil {
		_, _ = wslExec(log, "--unmount", tempVhdx)
		log.Warnf("Unexpected loading succeeded; WSL may have modified the mechanism. In this case, we believe there is no issue")
		return true
	}

	if strings.Contains(err.Error(), "WSL_E_WSL2_NEEDED") {
		log.Warn("Mount vhdx failed, BIOS may not support virtualization")
		return false
	}

	log.Infof("Mounting vhdx results in an expected error: %v", err)

	return true
}

func existsKernel() bool {
	// from `MSI` or `Windows Update`
	if system32, ok := util.System32Root(); ok {
		kernel := filepath.Join(system32, "lxss", "tools", "kernel")
		if err := util.Exists(kernel); err == nil {
			return true
		}
	}

	// from `Microsoft Store` or `Github`
	if programFiles, ok := util.ProgramFiles(); ok {
		kernel := filepath.Join(programFiles, "WSL", "tools", "kernel")
		if err := util.Exists(kernel); err == nil {
			return true
		}
	}

	return false
}

// isInstalled Checks if the WSL2 is installed.
func isInstalled(log *logger.Context) bool {
	// If the kernel file does not exist,
	// it means that the current system has only enabled the Features without running wsl --update.
	if !existsKernel() {
		return false
	}

	if err := util.Silent(log, Find(), "--status"); err != nil {
		return false
	}

	return true
}

// isFeatureEnabled Check `Microsoft-Windows-Subsystem-Linux` and `VirtualMachinePlatform` are enabled
func isFeatureEnabled(log *logger.Context) bool {
	// we cannot use the following methods for checking because these commands require administrative privileges.
	// 	1.Get-WindowsOptionalFeature -Online -FeatureName Microsoft-Windows-Subsystem-Linux
	// 	2.Get-WindowsOptionalFeature -Online -FeatureName VirtualMachinePlatform

	// In the old version of WSL, we could check if the feature was enabled by calling the --set-default-version command,
	// and if there was an error, it indicated that the feature was not enabled. However
	// in the new version, this behavior has changed;even if the feature is not enabled, there will be no error.
	//
	// This command will also have a side effect: it will change the default version of WSL to 2. However, this side effect is expected.
	if util.Silent(log, Find(), "--set-default-version", "2") != nil {
		return false
	}

	{

		out, err := wslExec(log, "--status")
		if err != nil {
			// --status The failure may be caused by issues such as the kernel file not existing,
			// and we should not assume that this error indicates that the feature is not enabled.
			return true
		}

		log.Infof("WSL --status result: %s", out)

		lines := strings.Split(string(out), "\n")

		// Delete the line below to avoid inaccuracies in the results.
		// Default Distribution: Ubuntu
		// Default Version: 2
		hasUselessHeader := len(lines) >= 2 && strings.Contains(lines[0], ":") && strings.Contains(lines[1], ":")
		if hasUselessHeader {
			log.Info("Exist useless header")
			lines = lines[2:]
		}
		lineStr := strings.Join(lines, "\n")

		log.Infof("Cleaned wsl --status line: %s", lineStr)

		keywords := []string{"Windows Subsystem for Linux", "BIOS", "wsl.exe", "enablevirtualization", "WSL1"}

		for _, key := range keywords {
			if strings.Contains(lineStr, key) {
				log.Warnf("Find keyword: %s in status result", key)
				return false
			}
		}
	}

	return true
}

func wslVersion(log *logger.Context) (string, error) {
	br, err := wslExec(log, "--version")
	if err != nil {
		return "", fmt.Errorf("failed to get WSL2 version: %w", err)
	}

	r := string(br)
	wslLine := strings.Split(r, "\n")[0]
	wslLine = strings.TrimSpace(wslLine)
	offset := strings.LastIndex(wslLine, " ")
	if offset == -1 {
		return r, fmt.Errorf("failed to parse WSL2 version: %s", r)
	}

	return strings.TrimSpace(wslLine[offset+1:]), nil
}

const minVersion = "2.1.5"

func shouldUpdateWSL(log *logger.Context) bool {
	if isInstalled := isInstalled(log); !isInstalled {
		log.Info("WSL2 is not updated, should update")
		return true
	}

	v, err := wslVersion(log)
	if err != nil {
		log.Warnf("Failed to get WSL2 version: %v", err)
		return true
	}

	log.Infof("Current WSL2 version: %s", v)
	currentVersion, err := version.NewVersion(v)
	if err != nil {
		log.Warnf("Failed to parse current WSL2 version: %v", err)
		return true
	}

	minVersion, err := version.NewVersion(minVersion)
	if err != nil {
		log.Warnf("Failed to parse min WSL2 version: %v", err)
		return true
	}

	if currentVersion.LessThan(minVersion) {
		log.Infof("Current WSL2 version is less than min version: %s < %s", currentVersion, minVersion)
		return true
	}

	return false
}
