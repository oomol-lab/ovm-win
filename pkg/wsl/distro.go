package wsl

import (
	"bufio"
	"bytes"
	"fmt"
	"github.com/oomol-lab/ovm-win/pkg/logger"
	"github.com/oomol-lab/ovm-win/pkg/util"
	"path/filepath"
	"strings"
)

func wslExec(log *logger.Context, args ...string) ([]byte, error) {
	cmd := util.SilentCmd(Find(), args...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	cmd.Env = []string{"WSL_UTF8=1"}

	log.Infof("Running command: %s %s", Find(), strings.Join(args, " "))
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("`%v %v` failed: %v %v (%v)", Find(), strings.Join(args, " "), stderr.String(), stdout.String(), err)
	}

	return stdout.Bytes(), nil
}

// Shutdown the wsl2 entirely
func Shutdown(log *logger.Context) error {
	if _, err := wslExec(log, "--shutdown"); err != nil {
		return fmt.Errorf("could not shut WSL down: %w", err)
	}
	return nil
}

func Terminate(log *logger.Context, distroName string) error {
	if _, err := wslExec(log, "--terminate", distroName); err != nil {
		return fmt.Errorf("could not terminate distro %q: %w", distroName, err)
	}
	return nil
}

func getAllWSLDistros(log *logger.Context, running bool) (all map[string]struct{}, err error) {
	args := []string{"--list", "--all", "--quiet"}
	if running {
		args = append(args, "--running")
	}

	out, err := wslExec(log, args...)
	if err != nil {
		return nil, fmt.Errorf("could not get distros: %w", err)
	}

	scanner := bufio.NewScanner(bytes.NewReader(out))
	scanner.Split(bufio.ScanLines)
	for scanner.Scan() {
		line := scanner.Text()

		fields := strings.Fields(line)
		if len(fields) > 0 {
			all[fields[0]] = struct{}{}
		}
	}
	return
}

func ImportDistro(log *logger.Context, overwrite bool, distroName, installPath, rootfs string) error {
	if overwrite {
		if b, _ := IsRegister(log, distroName); b == true {
			if err := Unregister(log, distroName); err != nil {
				return err
			}
		}
	}

	if err := util.Exists(filepath.Join(installPath, "ext4.vhdx")); err == nil {
		return fmt.Errorf("%s exist, You need delete it first", filepath.Join(installPath, "ext4.vhdx"))
	}

	if _, err := wslExec(log, "--import", distroName, installPath, rootfs); err != nil {
		return fmt.Errorf("import %s failed: %v", rootfs, err)
	}

	return nil
}

func Unregister(log *logger.Context, distroName string) error {
	if _, err := wslExec(log, "--unregister", distroName); err != nil {
		return fmt.Errorf("unregister %s failed: %v", distroName, err)
	}
	return nil
}

func IsRegister(log *logger.Context, distroName string) (registered bool, err error) {
	distros, err := getAllWSLDistros(log, false)
	if err != nil {
		return false, err
	}

	_, exists := distros[distroName]
	return exists, nil
}
