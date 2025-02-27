// SPDX-FileCopyrightText: 2024-2025 OOMOL, Inc. <https://www.oomol.com>
// SPDX-License-Identifier: MPL-2.0

package channel

type _context struct {
	wslUpdated       chan struct{}
	wslConfigUpdated chan int
	wslShutdown      chan struct{}
}

var c *_context

func init() {
	c = &_context{
		wslUpdated:       make(chan struct{}, 1),
		wslConfigUpdated: make(chan int, 1),
		wslShutdown:      make(chan struct{}, 1),
	}
}

func Close() {
	close(c.wslUpdated)
	close(c.wslConfigUpdated)
	close(c.wslShutdown)
}

func NotifyWSLUpdated() {
	c.wslUpdated <- struct{}{}
}

func ReceiveWSLUpdated() <-chan struct{} {
	return c.wslUpdated
}

func NotifyWSLConfigUpdated(flag int) {
	c.wslConfigUpdated <- flag
}

func ReceiveWSLConfigUpdated() <-chan int {
	return c.wslConfigUpdated
}

func NotifyWSLShutdown() {
	c.wslShutdown <- struct{}{}
}

func ReceiveWSLShutdown() <-chan struct{} {
	return c.wslShutdown
}
