// SPDX-FileCopyrightText: 2024 OOMOL, Inc. <https://www.oomol.com>
// SPDX-License-Identifier: MPL-2.0

package event

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/Code-Hex/go-infinity-channel"
	"github.com/Microsoft/go-winio"
	"github.com/oomol-lab/ovm-win/pkg/logger"
)

type Name string

const (
	NeedReboot       Name = "NeedReboot"
	SystemNotSupport Name = "SystemNotSupport"
	UpdatingWSL      Name = "UpdatingWSL"
	UpdatingRootFS   Name = "UpdatingRootFS"
	UpdatingData     Name = "UpdatingData"
	StartingVM       Name = "StartingVM"
	Exit             Name = "Exit"
	Error            Name = "Error"
)

type datum struct {
	name    Name
	message string
}

type event struct {
	client  *http.Client
	log     *logger.Context
	channel *infinity.Channel[*datum]
}

var e *event

// see: https://github.com/Code-Hex/go-infinity-channel/issues/1
var waitDone = make(chan struct{})

func Setup(log *logger.Context, socketPath string) {
	c := &http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, _, addr string) (net.Conn, error) {
				return winio.DialPipeContext(ctx, socketPath)
			},
		},
		Timeout: 200 * time.Millisecond,
	}

	e = &event{
		client:  c,
		log:     log,
		channel: infinity.NewChannel[*datum](),
	}

	go func() {
		for datum := range e.channel.Out() {
			uri := fmt.Sprintf("http://ovm/notify?event=%s&message=%s", datum.name, url.QueryEscape(datum.message))
			e.log.Infof("Notify %s event to %s", datum.name, uri)

			if resp, err := e.client.Get(uri); err != nil {
				e.log.Warnf("Notify %+v event failed: %v", *datum, err)
			} else {
				_ = resp.Body.Close()
				if resp.StatusCode != http.StatusOK {
					e.log.Warnf("Notify %+v event failed, status code is: %d", *datum, resp.StatusCode)
				}
			}

			if datum.name == Exit || datum.name == NeedReboot {
				waitDone <- struct{}{}
				return
			}
		}
	}()
}

func Notify(name Name) {
	if e == nil {
		return
	}

	e.channel.In() <- &datum{
		name: name,
	}

	// wait for the event to be processed
	// Exit event indicates the main process exit
	// NeedReboot event indicates the child process exit
	if name == Exit || name == NeedReboot {
		e.channel.Close()
		<-waitDone
		close(waitDone)
		e = nil
	}
}

func NotifyError(err error) {
	if e == nil {
		return
	}

	e.channel.In() <- &datum{
		name:    Error,
		message: err.Error(),
	}
}
