// SPDX-FileCopyrightText: 2024 OOMOL, Inc. <https://www.oomol.com>
// SPDX-License-Identifier: MPL-2.0

package request

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	NoCache = "no-cache"
	TimeOut = "timeout"
)

const DefaultTimeout = 200 * time.Millisecond

func Get(ctx context.Context, url string) ([]byte, error) {
	noCache, ok := ctx.Value(NoCache).(bool)
	if !ok {
		noCache = false
	}

	timeout, ok := ctx.Value(TimeOut).(time.Duration)
	if !ok {
		timeout = DefaultTimeout
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create %s request: %w", url, err)
	}

	c := &http.Client{
		Timeout: timeout,
	}

	if noCache {
		req.Header.Set("Cache-Control", "no-cache")
		t := fmt.Sprintf("%d", time.Now().UnixNano())
		req.URL.Query().Set("noCache", t)
	}

	resp, err := c.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send %s request: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	return body, nil
}
