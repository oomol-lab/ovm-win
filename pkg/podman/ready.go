// SPDX-FileCopyrightText: 2024 OOMOL, Inc. <https://www.oomol.com>
// SPDX-License-Identifier: MPL-2.0

package podman

import (
	"context"
	"fmt"
	"time"

	"github.com/oomol-lab/ovm-win/pkg/util/request"
)

const (
	timeout       = 10 * time.Second
	retryInterval = 200 * time.Millisecond
)

func Ready(ctx context.Context, podmanPort int) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// See: https://docs.podman.io/en/latest/_static/api.html?version=v4.0#tag/images/operation/ImageListLibpod
	url := fmt.Sprintf("http://127.0.0.1:%d/libpod/images/json", podmanPort)
	for {
		if _, err := request.Get(ctx, url); err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}

			time.Sleep(retryInterval)
			continue
		}

		return nil
	}
}
