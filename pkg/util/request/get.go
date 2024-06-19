// SPDX-FileCopyrightText: 2024 OOMOL, Inc. <https://www.oomol.com>
// SPDX-License-Identifier: MPL-2.0

package request

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

const timeout = 200 * time.Millisecond

func Get(ctx context.Context, url string) error {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create %s request: %w", url, err)
	}

	c := &http.Client{
		Timeout: timeout,
	}

	resp, err := c.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send %s request: %w", url, err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code %d", resp.StatusCode)
	}

	return nil
}
