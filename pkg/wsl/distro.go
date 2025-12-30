// SPDX-FileCopyrightText: 2024-2025 OOMOL, Inc. <https://www.oomol.com>
// SPDX-License-Identifier: MPL-2.0

package wsl

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/oomol-lab/ovm-win/pkg/ipc/event"
	"github.com/oomol-lab/ovm-win/pkg/logger"
	"github.com/oomol-lab/ovm-win/pkg/podman"
	"github.com/oomol-lab/ovm-win/pkg/types"
	"github.com/oomol-lab/ovm-win/pkg/util"
	"golang.org/x/sync/errgroup"
)

var ErrDistroNotExist = errors.New("distro does not exist")
var ErrDistroNotRunning = errors.New("distro is not running")
var ErrSharingViolation = errors.New("sharing violation")

func Shutdown(log *logger.Context) error {
	if _, err := wslExec(log, "--shutdown"); err != nil {
		return fmt.Errorf("could not shutdown WSL: %w", err)
	}

	return nil
}

func Terminate(log *logger.Context, distroName string) error {
	if _, err := wslExec(log, "--terminate", distroName); err != nil {
		return fmt.Errorf("could not terminate distro %q: %w", distroName, err)
	}

	return nil
}

func ImportDistro(log *logger.Context, distroName, installPath, rootfs string) error {
	if _, err := wslExec(log, "--import", distroName, installPath, rootfs, "--version", "2"); err != nil {
		return fmt.Errorf("import distro %s failed: %w", rootfs, err)
	}

	return nil
}

func Unregister(log *logger.Context, distroName string) error {
	if _, err := wslExec(log, "--unregister", distroName); err != nil {
		return fmt.Errorf("unregister %s failed: %w", distroName, err)
	}

	return nil
}

func IsRegister(log *logger.Context, distroName string) (ok bool, err error) {
	distros, err := getAllWSLDistros(log, false)
	if err != nil {
		return false, err
	}

	_, exists := distros[distroName]
	return exists, nil
}

func IsRunning(log *logger.Context, distroName string) (ok bool, err error) {
	distros, err := getAllWSLDistros(log, true)
	if err != nil {
		return false, err
	}

	_, exists := distros[distroName]
	return exists, nil
}

func SyncDisk(log *logger.Context, distroName string) error {
	if err := wslInvoke(log, distroName, "sync"); err != nil {
		return fmt.Errorf("sync disk failed: %w", err)
	}

	return nil
}

func SafeSyncDisk(log *logger.Context, distroName string) error {
	if ok, err := IsRegister(log, distroName); err != nil {
		return fmt.Errorf("cannot safe terminate distro %s, because failed to check if distro is registered: %w", distroName, err)
	} else if !ok {
		return ErrDistroNotExist
	}

	if ok, err := IsRunning(log, distroName); err != nil {
		return fmt.Errorf("cannot safe terminate distro %s, because failed to check if distro is running: %w", distroName, err)
	} else if !ok {
		return ErrDistroNotRunning
	}

	_ = SyncDisk(log, distroName)

	return nil
}

func MountVHDX(log *logger.Context, paths ...string) error {
	for _, path := range paths {
		if _, err := wslExec(log, "--mount", "--bare", "--vhd", path); err != nil {
			if strings.Contains(err.Error(), "WSL_E_USER_VHD_ALREADY_ATTACHED") {
				log.Infof("VHDX already mounted: %s", path)
				continue
			}
			return fmt.Errorf("wsl mount %s failed: %w", path, err)
		}
	}

	return nil
}

func UmountVHDX(log *logger.Context, paths ...string) error {
	for _, path := range paths {
		if err := util.Exists(path); err != nil && os.IsNotExist(err) {
			continue
		}

		if _, err := wslExec(log, "--unmount", path); err != nil {
			if strings.Contains(err.Error(), "ERROR_FILE_NOT_FOUND") {
				log.Infof("VHDX already unmounted: %s", path)
				continue
			}
			return fmt.Errorf("wsl umount %s failed: %w", path, err)
		}
	}

	return nil
}

func MoveDistro(log *logger.Context, distroName, newPath string) error {
	if _, err := wslExec(log, "--manage", distroName, "--move", newPath); err != nil {
		if strings.Contains(err.Error(), "ERROR_SHARING_VIOLATION") {
			return ErrSharingViolation
		}
		if strings.Contains(err.Error(), "WSL_E_DISTRO_NOT_STOPPED") {
			return ErrSharingViolation
		}

		return fmt.Errorf("wsl move %s failed: %w", newPath, err)
	}

	return nil
}

func RequestStop(log *logger.Context, name string) error {
	_ = SyncDisk(log, name)

	if err := wslInvoke(log, name, "/opt/ovmd", "--killall"); err != nil {
		return fmt.Errorf("failed to request stop: %w", err)
	}

	if err := Terminate(log, name); err != nil {
		return fmt.Errorf("failed to terminate in request stop: %w", err)
	}

	return nil
}

func Stop(log *logger.Context, name string) error {
	_ = SyncDisk(log, name)

	if err := Terminate(log, name); err != nil {
		return fmt.Errorf("failed to terminate in stop: %w", err)
	}

	return nil
}

func Launch(ctx context.Context, log *logger.Context, opt *types.RunOpt) error {
	event.NotifyRun(event.Starting)

	dataPath := filepath.Join(opt.ImageDir, "data.vhdx")
	sourceCodeDiskPath := filepath.Join(opt.ImageDir, "sourcecode.vhdx")
	if err := MountVHDX(log, dataPath, sourceCodeDiskPath); err != nil {
		return fmt.Errorf("failed to mount vhdx disk: %w", err)
	}

	g, ctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		util.RegisteredExitFuncs(func() {
			log.Info("Stopping distro...")
			if err := RequestStop(log, opt.DistroName); err != nil {
				log.Warnf("Failed to stop distro %s: %v", opt.DistroName, err)
			}
			log.Info("Distro stopped")
		})

		return launchOVMD(ctx, opt)
	})
	g.Go(func() error {
		// TODO: ovmd needs some time to kill the previous podman processes.
		//   This is just a temporary solution, waiting for ovmd to support sending the ready event.
		//   @BlackHole1
		time.Sleep(1 * time.Second)
		if err := podman.Ready(ctx, opt.PodmanPort); err != nil {
			return fmt.Errorf("podman is not ready: %w", err)
		}

		event.NotifyRun(event.Ready)
		return nil
	})

	return g.Wait()
}

func launchOVMD(ctx context.Context, opt *types.RunOpt) error {
	log := opt.Logger
	vmLog, err := log.NewWithAppendName("vm")
	if err != nil {
		return fmt.Errorf("could not create vm logger: %w", err)
	}

	// Backward compatibility
	oldDataSector := util.DataSize(opt.Name+opt.ImageDir) / 512
	dataSector := util.DataSize(opt.Name) / 512

	// See: https://github.com/oomol-lab/ovm-builder/blob/main/layers/wsl2_amd64/opt/ovmd
	cmd := util.SilentCmdContext(ctx, Find(),
		"-d", opt.DistroName,
		"/opt/ovmd",
		"-p", fmt.Sprintf("%d", opt.PodmanPort),
		"-s", fmt.Sprintf("%d,%d", dataSector, oldDataSector),
	)
	cmd.Env = []string{"WSL_UTF8=1"}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("could not get stdout pipe: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("could not get stderr pipe: %w", err)
	}
	defer func() {
		_ = stdout.Close()
		_ = stderr.Close()
	}()

	log.Infof("Launching %s: podman port is: %d, data sector count: %d", opt.DistroName, opt.PodmanPort, dataSector)

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start `%s`: %w", opt.DistroName, err)
	}

	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			vmLog.Raw(scanner.Text())
		}
	}()
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			vmLog.Raw(scanner.Text())
		}
	}()

	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("failed to launch ovmd for `%s`: %s", opt.DistroName, err)
	}

	return fmt.Errorf("ovmd unexpected closed")
}

// GetAllWSLDistros returns all WSL distros
func getAllWSLDistros(log *logger.Context, running bool) (map[string]struct{}, error) {
	args := []string{"--list", "--quiet"}
	if running {
		args = append(args, "--running")
	} else {
		args = append(args, "--all")
	}

	out, err := wslExec(log, args...)
	if err != nil {
		return nil, fmt.Errorf("could not get distros: %w", err)
	}

	all := make(map[string]struct{})
	scanner := bufio.NewScanner(bytes.NewReader(out))
	scanner.Split(bufio.ScanLines)

	// `wsl --list --quiet --all` output:
	//
	//	Ubuntu
	//	Debian
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)

		if len(fields) > 0 {
			all[fields[0]] = struct{}{}
		}
	}

	return all, nil
}

func wslExec(log *logger.Context, args ...string) ([]byte, error) {
	cmd := util.SilentCmd(Find(), args...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	cmd.Env = []string{"WSL_UTF8=1"}

	cmdStr := fmt.Sprintf("%s %s", Find(), strings.Join(args, " "))

	log.Infof("Running command in wsl: %s", cmdStr)

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to run command `%s` failed: %s %s (%w)", cmdStr, stderr.String(), stdout.String(), err)
	}

	return stdout.Bytes(), nil
}

func wslInvoke(log *logger.Context, name string, args ...string) error {
	newArgs := []string{"-d", name}
	newArgs = append(newArgs, args...)
	cmd := util.SilentCmd(Find(), newArgs...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	cmd.Env = []string{"WSL_UTF8=1"}

	cmdStr := fmt.Sprintf("%s %s", Find(), strings.Join(newArgs, " "))

	log.Infof("Running command in distro: %s", cmdStr)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to run command `%s` in distro: %s %s (%w)", cmdStr, stderr.String(), stdout.String(), err)
	}

	return nil
}
