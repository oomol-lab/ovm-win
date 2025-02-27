// SPDX-FileCopyrightText: 2024-2025 OOMOL, Inc. <https://www.oomol.com>
// SPDX-License-Identifier: MPL-2.0

package channel

type _context struct {
	wslUpdated chan struct{}
}

var c *_context

func init() {
	c = &_context{
		wslUpdated: make(chan struct{}, 1),
	}
}

func Close() {
	close(c.wslUpdated)
}

func NotifyWSLUpdated() {
	c.wslUpdated <- struct{}{}
}

func ReceiveWSLUpdated() <-chan struct{} {
	return c.wslUpdated
}
