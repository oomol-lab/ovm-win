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

type key string

const (
	kSys   key = "sys"
	kApp   key = "app"
	kError key = "error"
)

type sys string

const (
	SystemNotSupport     sys = "SystemNotSupport"
	EnableFeaturing      sys = "EnableFeaturing"
	EnableFeatureFailed  sys = "EnableFeatureFailed"
	EnableFeatureSuccess sys = "EnableFeatureSuccess"
	NeedReboot           sys = "NeedReboot"
	UpdatingWSL          sys = "UpdatingWSL"
	UpdateWSLFailed      sys = "UpdateWSLFailed"
	UpdateWSLSuccess     sys = "UpdateWSLSuccess"
)

type app string

const (
	UpdatingRootFS      app = "UpdatingRootFS"
	UpdateRootFSFailed  app = "UpdateRootFSFailed"
	UpdateRootFSSuccess app = "UpdateRootFSSuccess"
	UpdatingData        app = "UpdatingData"
	UpdateDataFailed    app = "UpdateDataFailed"
	UpdateDataSuccess   app = "UpdateDataSuccess"

	StartingVM app = "StartingVM"
	Ready      app = "Ready"
	Exit       app = "Exit"
)

type datum struct {
	name    string
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

			if datum.message == string(Exit) || datum.message == string(NeedReboot) {
				waitDone <- struct{}{}
				return
			}
		}
	}()
}

func notify(k string, v string) {
	if e == nil {
		return
	}

	e.channel.In() <- &datum{
		name:    k,
		message: v,
	}

	// wait for the event to be processed
	// Exit event indicates the main process exit
	// NeedReboot event indicates the child process exit
	if v == string(Exit) || v == string(NeedReboot) {
		e.channel.Close()
		<-waitDone
		close(waitDone)
		e = nil
	}
}

func NotifySys(v sys) {
	notify(string(kSys), string(v))
}

func NotifyApp(v app) {
	notify(string(kApp), string(v))
}

func NotifyError(err error) {
	notify(string(kError), err.Error())
}
