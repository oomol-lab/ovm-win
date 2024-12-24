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
	kPrepare key = "prepare"
	kRun     key = "run"
	kError   key = "error"
	kExit    key = "exit"
)

type prepare string

const (
	SystemNotSupport prepare = "SystemNotSupport"

	NotSupportVirtualization prepare = "NotSupportVirtualization"
	NeedEnableFeature        prepare = "NeedEnableFeature"
	EnableFeaturing          prepare = "EnableFeaturing"
	EnableFeatureFailed      prepare = "EnableFeatureFailed"
	EnableFeatureSuccess     prepare = "EnableFeatureSuccess"
	NeedReboot               prepare = "NeedReboot"

	NeedUpdateWSL    prepare = "NeedUpdateWSL"
	UpdatingWSL      prepare = "UpdatingWSL"
	UpdateWSLFailed  prepare = "UpdateWSLFailed"
	UpdateWSLSuccess prepare = "UpdateWSLSuccess"
)

type run string

const (
	UpdatingRootFS      run = "UpdatingRootFS"
	UpdateRootFSFailed  run = "UpdateRootFSFailed"
	UpdateRootFSSuccess run = "UpdateRootFSSuccess"

	UpdatingData      run = "UpdatingData"
	UpdateDataFailed  run = "UpdateDataFailed"
	UpdateDataSuccess run = "UpdateDataSuccess"

	Starting run = "Starting"
	Ready    run = "Ready"
)

type datum struct {
	name    key
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

			if datum.name == kExit || datum.message == string(NeedReboot) {
				waitDone <- struct{}{}
				return
			}
		}
	}()
}

func notify(k key, v string) {
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
	if k == kExit || v == string(NeedReboot) {
		<-waitDone
		close(waitDone)
		e.channel.Close()
	}
}

func NotifyPrepare(v prepare) {
	notify(kPrepare, string(v))
}

func NotifyRun(v run) {
	notify(kRun, string(v))
}

func NotifyError(err error) {
	notify(kError, err.Error())
}

func NotifyExit() {
	notify(kExit, "")
}
