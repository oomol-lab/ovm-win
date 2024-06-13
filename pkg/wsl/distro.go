package wsl

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/oomol-lab/ovm-win/pkg/logger"
	"github.com/oomol-lab/ovm-win/pkg/util"
)

var ErrDistroNotExist = errors.New("distro does not exist")
var ErrDistroNotRunning = errors.New("distro is not running")

// Shutdown the wsl2 entirely
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

func MountVHDX(log *logger.Context, path string) error {
	if _, err := wslExec(log, "--mount", "--bare", "--vhd", path); err != nil {
		if strings.Contains(err.Error(), "MountVhd/WSL_E_USER_VHD_ALREADY_ATTACHED") {
			log.Infof("vhdx already mounted: %s", path)
			return nil
		}
		return fmt.Errorf("wsl mount %s failed: %w", path, err)
	}

	return nil
}

func UmountVHDX(log *logger.Context, path string) error {
	if err := util.Exists(path); err != nil && os.IsNotExist(err) {
		return nil
	}

	if _, err := wslExec(log, "--unmount", path); err != nil {
		return fmt.Errorf("wsl umount %s failed: %w", path, err)
	}

	return nil
}

// GetAllWSLDistros returns all WSL distros
func getAllWSLDistros(log *logger.Context, running bool) (map[string]struct{}, error) {
	args := []string{"--list", "--quiet"}
	if running {
		args = append(args, "--running")
	}

	out, err := wslExec(log, args...)
	if err != nil {
		return nil, fmt.Errorf("could not get distros: %w", err)
	}

	all := make(map[string]struct{})
	scanner := bufio.NewScanner(bytes.NewReader(out))
	scanner.Split(bufio.ScanLines)

	// `wsl --list --quiet` output:
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
