// SPDX-FileCopyrightText: 2024 OOMOL, Inc. <https://www.oomol.com>
// SPDX-License-Identifier: MPL-2.0

package channel

type _context struct {
	wslEnvReady chan bool
}

var c *_context

func init() {
	c = &_context{
		wslEnvReady: make(chan bool, 1),
	}
}

func Close() {
	close(c.wslEnvReady)
}

func NotifyWSLEnvReady() {
	c.wslEnvReady <- true
}

func ReceiveWSLEnvReady() <-chan bool {
	return c.wslEnvReady
}
