// SPDX-FileCopyrightText: 2024 OOMOL, Inc. <https://www.oomol.com>
// SPDX-License-Identifier: MPL-2.0

package npipe

import (
	"fmt"
	"net"
	"os/user"

	"github.com/Microsoft/go-winio"
)

// allow built-in admins and system/kernel components
// ref: [Security Descriptor String Format] / [ace strings] / [sid strings]
//
// [Security Descriptor String Format]: https://learn.microsoft.com/en-us/windows/win32/secauthz/security-descriptor-string-format
// [sid strings]: https://learn.microsoft.com/en-us/windows/win32/secauthz/sid-strings
// [ace strings]: https://learn.microsoft.com/en-us/windows/win32/secauthz/ace-strings
const sddlSysAllAdmAll = "D:P(A;;GA;;;SY)(A;;GA;;;BA)"

func Create(socketPath string) (nl net.Listener, err error) {
	uc, _ := user.Current()

	// Also allow current user
	sddl := fmt.Sprintf("%s(A;;GA;;;%s)", sddlSysAllAdmAll, uc.Uid)
	pc := winio.PipeConfig{
		SecurityDescriptor: sddl,
		MessageMode:        true,
		InputBufferSize:    65536,
		OutputBufferSize:   65536,
	}

	nl, err = winio.ListenPipe(socketPath, &pc)
	if err != nil {
		return nil, fmt.Errorf("failed to listen pipe: %w", err)
	}

	return
}
