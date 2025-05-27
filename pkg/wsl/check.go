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

	if ok := checkBIOS(opt); !ok {
		opt.Logger.Info("Virtualization is not supported")
		event.NotifyInit(event.NotSupportVirtualization)

		<-ctx.Done()
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

func checkBIOS(opt *types.InitOpt) bool {
	log := opt.Logger

	if list, err := getAllWSLDistros(log, false); err == nil && len(list) != 0 {
		var first string
		for key := range list {
			first = key
			break
		}

		flag := "TEST_PASS"

		var out string
		_ = Exec(log).SetAllOut(&out).SetDistro(first).Run("echo", flag)
		if strings.Contains(out, flag) {
			log.Info("Exist WSL distros and succeeded invoke, BIOS support virtualization")
			return true
		}

		if strings.Contains(out, "HCS_E_HYPERV_NOT_INSTALLED") {
			log.Info("execute wsl command failed, BIOS not support virtualization")
			return false
		}
	}

	recordCPUFeature(log)

	if tryImportTestDistro(log) {
		log.Info("Test distro imported successfully, maybe BIOS support virtualization")
		return true
	}

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

	incompatibleKeys := NewConfig(log).ExistIncompatible()
	if len(incompatibleKeys) == 0 {
		log.Info("WSL2 config is compatible")
		return true
	}

	event.NotifyInit(event.WSLConfigMaybeIncompatible, strings.Join(incompatibleKeys, ","))
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

func recordCPUFeature(log *logger.Context) {
	vf, slat := sys.IsSupportedVirtualization()
	if !slat {
		log.Warn("SLAT is not supported")
	}

	if !vf {
		log.Warn("VT-x is not supported")
	}
}

func tryImportTestDistro(log *logger.Context) bool {
	random := fmt.Sprintf("ovm-test-distro-%s", util.RandomString(5))
	tempDir := filepath.Join(os.TempDir(), random)
	defer func() {
		_ = os.RemoveAll(tempDir)
	}()

	target := filepath.Join(tempDir, "target")
	emptyTar := filepath.Join(tempDir, "empty.tar")

	if err := os.MkdirAll(target, 0755); err != nil {
		log.Warnf("Failed to create target directory: %v", err)
		return true
	}

	if err := os.WriteFile(emptyTar, []byte{}, 0644); err != nil {
		log.Warnf("Failed to create empty tar file: %v", err)
		return true
	}

	var out string
	_ = Exec(log).SetAllOut(&out).Run("--import", random, target, emptyTar, "--version", "2")
	if strings.Contains(out, "HCS_E_HYPERV_NOT_INSTALLED") {
		_ = log.Errorf("Import test distro failed, BIOS not support virtualization")
		return false
	}

	if err := Exec(log).Run("--unregister", random); err != nil {
		log.Warnf("Failed to unregister test distro: %v", err)
	}

	return true
}

// isInstalled Checks if the WSL2 is installed.
func isInstalled(log *logger.Context) bool {
	result := ""
	out, err := wslExec(log, "--help")
	result += string(out) + "\n"
	if err != nil {
		result += err.Error()
	}

	log.Infof("WSL --help result: %s", result)

	if strings.Contains(result, "--version, -v") {
		return true
	}

	return false
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
			// In Windows 10, if features are not enabled, the status will report an error and display information like `wsl.exe –install –no-distribution`
			// In Windows 11, if features are not enabled, the status will not report an error.
			if strings.Contains(err.Error(), "--install --no-distribution") {
				return false
			}

			// The failure may be caused by issues such as the kernel file not existing,
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

const minVersion = "2.3.24"

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
