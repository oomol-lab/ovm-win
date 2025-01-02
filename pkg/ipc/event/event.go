// SPDX-FileCopyrightText: 2024-2025 OOMOL, Inc. <https://www.oomol.com>
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

type stage string

const (
	kInit stage = "init"
	kRun  stage = "run"
)

type nameInit string

const (
	SystemNotSupport nameInit = "SystemNotSupport"

	NotSupportVirtualization nameInit = "NotSupportVirtualization"
	NeedEnableFeature        nameInit = "NeedEnableFeature"
	EnableFeaturing          nameInit = "EnableFeaturing"
	EnableFeatureFailed      nameInit = "EnableFeatureFailed"
	EnableFeatureSuccess     nameInit = "EnableFeatureSuccess"
	NeedReboot               nameInit = "NeedReboot"

	NeedUpdateWSL    nameInit = "NeedUpdateWSL"
	UpdatingWSL      nameInit = "UpdatingWSL"
	UpdateWSLFailed  nameInit = "UpdateWSLFailed"
	UpdateWSLSuccess nameInit = "UpdateWSLSuccess"
	InitExit         nameInit = "Exit"
	InitError        nameInit = "Error"
)

type nameRun string

const (
	UpdatingRootFS      nameRun = "UpdatingRootFS"
	UpdateRootFSFailed  nameRun = "UpdateRootFSFailed"
	UpdateRootFSSuccess nameRun = "UpdateRootFSSuccess"

	UpdatingData      nameRun = "UpdatingData"
	UpdateDataFailed  nameRun = "UpdateDataFailed"
	UpdateDataSuccess nameRun = "UpdateDataSuccess"

	Starting nameRun = "Starting"
	Ready    nameRun = "Ready"
	RunExit  nameRun = "Exit"
	RunError nameRun = "Error"
)

type datum struct {
	stage stage
	name  string
	value string
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
			uri := fmt.Sprintf("http://ovm/notify?stage=%s&name=%s&value=%s", datum.stage, url.QueryEscape(datum.name), url.QueryEscape(datum.value))
			e.log.Infof("Notify %s event to %s", datum.name, uri)

			if resp, err := e.client.Get(uri); err != nil {
				e.log.Warnf("Notify %+v event failed: %v", *datum, err)
			} else {
				_ = resp.Body.Close()
				if resp.StatusCode != http.StatusOK {
					e.log.Warnf("Notify %+v event failed, status code is: %d", *datum, resp.StatusCode)
				}
			}

			if datum.name == "Exit" || datum.name == string(NeedReboot) {
				waitDone <- struct{}{}
				return
			}
		}
	}()
}

func notify(c stage, name string, value ...string) {
	if e == nil {
		return
	}

	v := ""
	if len(value) == 0 {
		v = ""
	} else {
		v = value[0]
	}

	e.channel.In() <- &datum{
		stage: c,
		name:  name,
		value: v,
	}

	// wait for the event to be processed
	// Exit event indicates the main process exit
	// NeedReboot event indicates the child process exit
	if name == "Exit" || name == string(NeedReboot) {
		<-waitDone
		close(waitDone)
		e.channel.Close()
	}
}

func NotifyInit(name nameInit, value ...string) {
	notify(kInit, string(name), value...)
}

func NotifyRun(name nameRun, value ...string) {
	notify(kRun, string(name), value...)
}
