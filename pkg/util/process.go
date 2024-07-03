// SPDX-FileCopyrightText: 2024 OOMOL, Inc. <https://www.oomol.com>
// SPDX-License-Identifier: MPL-2.0

package util

import (
	"context"
	"fmt"
	"time"

	"github.com/oomol-lab/ovm-win/pkg/logger"
	"github.com/shirou/gopsutil/v4/process"
)

func WaitBindPID(ctx context.Context, log *logger.Context, pid int) error {
	if pid == 0 {
		log.Info("pid is 0, no need to wait")
		<-ctx.Done()
		return nil
	}

	log.Infof("wait bind pid: %d exit", pid)

	for {
		select {
		case <-ctx.Done():
			log.Info("cancel wait bind pid, because context done")
			return nil
		default:
			exists, err := process.PidExistsWithContext(ctx, int32(pid))
			if err != nil {
				return fmt.Errorf("check bind pid %d error: %w", pid, err)
			}

			if !exists {
				return fmt.Errorf("bind pid %d exited", pid)
			}

			time.Sleep(1 * time.Second)
			continue
		}
	}
}
