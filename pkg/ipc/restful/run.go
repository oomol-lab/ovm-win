// SPDX-FileCopyrightText: 2024 OOMOL, Inc. <https://www.oomol.com>
// SPDX-License-Identifier: MPL-2.0

package restful

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/oomol-lab/ovm-win/pkg/logger"
	"github.com/oomol-lab/ovm-win/pkg/types"
	"github.com/oomol-lab/ovm-win/pkg/winapi/npipe"
	"github.com/oomol-lab/ovm-win/pkg/wsl"
)

type runPrepare struct {
	opt *types.RunOpt
	log *logger.Context

	needWaitClose bool
	waitClose     chan struct{}
}

func SetupRun(opt *types.RunOpt) (s Server, err error) {
	rp := &runPrepare{
		opt:       opt,
		log:       opt.Logger,
		waitClose: make(chan struct{}, 1),
	}

	nl, err := npipe.Create(opt.RestfulEndpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to create npipe listener: %w", err)
	}

	return &server{
		log:    opt.Logger,
		router: rp,
		nl:     nl,
	}, nil
}

func (r *runPrepare) Close() error {
	if r.needWaitClose {
		select {
		case <-r.waitClose:
		// Avoid being unable to exit due to unexpected situations.
		case <-time.After(5 * time.Second):
		}
	}

	close(r.waitClose)

	return nil
}

func (r *runPrepare) mux() http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/info", mustGet(r.log, middlewareLog(r.log, r.info)))
	mux.Handle("/request-stop", mustPost(r.log, middlewareLog(r.log, r.needWait(r.requestStop))))
	mux.Handle("/stop", mustPost(r.log, middlewareLog(r.log, r.needWait(r.stop))))

	return mux
}

type infoResponse struct {
	PodmanHost string `json:"podmanHost"`
	PodmanPort int    `json:"podmanPort"`
}

func (r *runPrepare) info(w http.ResponseWriter, req *http.Request) {
	_ = json.NewEncoder(w).Encode(&infoResponse{
		PodmanHost: "127.0.0.1",
		PodmanPort: r.opt.PodmanPort,
	})
}

func (r *runPrepare) requestStop(w http.ResponseWriter, req *http.Request) {
	if err := wsl.RequestStop(r.log, r.opt.DistroName); err != nil {
		r.log.Warnf("Failed to request stop: %v", err)
		http.Error(w, "failed to request stop", http.StatusInternalServerError)
		return
	}

	r.opt.StoppedWithAPI = true
}

func (r *runPrepare) stop(w http.ResponseWriter, req *http.Request) {
	if err := wsl.Stop(r.log, r.opt.DistroName); err != nil {
		r.log.Warnf("Failed to stop: %v", err)
		http.Error(w, "failed to stop", http.StatusInternalServerError)
		return
	}

	r.opt.StoppedWithAPI = true
}

func (r *runPrepare) needWait(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		r.needWaitClose = true

		next.ServeHTTP(w, req)

		r.waitClose <- struct{}{}
		r.needWaitClose = false
		r.waitClose = make(chan struct{})
	}
}
