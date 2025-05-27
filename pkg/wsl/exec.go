// SPDX-FileCopyrightText: 2025 OOMOL, Inc. <https://www.oomol.com>
// SPDX-License-Identifier: MPL-2.0

package wsl

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/oomol-lab/ovm-win/pkg/logger"
	"github.com/oomol-lab/ovm-win/pkg/util"
)

type ExecContext struct {
	log *logger.Context

	distro string
	stdout *string
	stderr *string
	allOut *string
}

func Exec(log *logger.Context) *ExecContext {
	return &ExecContext{
		log: log,
	}
}

func (c *ExecContext) SetDistro(name string) *ExecContext {
	c.distro = name
	return c
}

func (c *ExecContext) SetStdout(stdout *string) *ExecContext {
	if stdout != nil {
		c.stdout = stdout
	}
	return c
}

func (c *ExecContext) SetStderr(stderr *string) *ExecContext {
	if stderr != nil {
		c.stderr = stderr
	}
	return c
}

func (c *ExecContext) SetAllOut(allOut *string) *ExecContext {
	if allOut != nil {
		c.allOut = allOut
	}
	return c
}

func (c *ExecContext) Run(args ...string) error {
	var newArgs []string
	if c.distro != "" {
		newArgs = append(newArgs, "-d", c.distro)
	}
	newArgs = append(newArgs, args...)

	cmd := util.SilentCmd(Find(), newArgs...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	cmd.Env = []string{"WSL_UTF8=1"}

	cmdStr := fmt.Sprintf("%s %s", Find(), strings.Join(newArgs, " "))

	c.log.Infof("Running wsl command: %s", cmdStr)

	err := cmd.Run()

	if c.stdout != nil {
		*c.stdout = stdout.String()
	}
	if c.stderr != nil {
		*c.stderr = stderr.String()
	}
	if c.allOut != nil {
		*c.allOut = stdout.String() + stderr.String()
	}

	if err != nil {
		return fmt.Errorf("failed to run command `%s`: %s %s (%w)", cmdStr, stdout.String(), stderr.String(), err)
	}

	return nil
}
