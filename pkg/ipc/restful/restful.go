// SPDX-FileCopyrightText: 2024 OOMOL, Inc. <https://www.oomol.com>
// SPDX-License-Identifier: MPL-2.0

package restful

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"

	"github.com/oomol-lab/ovm-win/pkg/channel"
	"github.com/oomol-lab/ovm-win/pkg/cli"
	"github.com/oomol-lab/ovm-win/pkg/logger"
	"github.com/oomol-lab/ovm-win/pkg/winapi/npipe"
	"github.com/oomol-lab/ovm-win/pkg/winapi/sys"
	"github.com/oomol-lab/ovm-win/pkg/wsl"
)

type restful struct {
	log *logger.Context
	opt *cli.Context
	ctx context.Context
	nl  net.Listener
}

type Run interface {
	Run() error
}

func Setup(ctx context.Context, opt *cli.Context, log *logger.Context) (r Run, err error) {
	nl, err := npipe.Create(opt.RestfulEndpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to create npipe listener: %w", err)
	}

	r = &restful{
		log: log,
		opt: opt,
		ctx: ctx,
		nl:  nl,
	}

	return r, nil
}

func (r *restful) Run() error {
	r.log.Infof("RESTful server is ready to run on %s", r.opt.RestfulEndpoint)

	stop := context.AfterFunc(r.ctx, func() {
		_ = r.nl.Close()
		r.log.Info("RESTful server is shutting down, because the context is done")
	})
	defer stop()

	server := &http.Server{
		Handler: r.mux(),
	}

	if err := server.Serve(r.nl); err != nil {
		if r.ctx.Err() != nil {
			return r.ctx.Err()
		}

		return fmt.Errorf("failed to serve restful server: %w", err)
	}

	return fmt.Errorf("restful server is closed")
}

func (r *restful) mux() http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/info", mustGet(r.log, middlewareLog(r.log, r.info)))
	mux.Handle("/reboot", mustPost(r.log, middlewareLog(r.log, r.reboot)))
	mux.Handle("/enable-feature", mustPost(r.log, middlewareLog(r.log, r.enableFeature)))
	mux.Handle("/update-wsl", mustPut(r.log, middlewareLog(r.log, r.updateWSL)))
	mux.Handle("/request-stop", mustPost(r.log, middlewareLog(r.log, r.requestStop)))
	mux.Handle("/stop", mustPost(r.log, middlewareLog(r.log, r.stop)))

	return mux
}

type infoResponse struct {
	PodmanHost string `json:"podmanHost"`
	PodmanPort int    `json:"podmanPort"`
}

func (r *restful) info(w http.ResponseWriter, req *http.Request) {
	_ = json.NewEncoder(w).Encode(&infoResponse{
		PodmanHost: "127.0.0.1",
		PodmanPort: r.opt.PodmanPort,
	})
}

type rebootBody struct {
	// RunOnce is the command to run after the next system startup
	RunOnce string `json:"runOnce"`
}

func (r *restful) reboot(w http.ResponseWriter, req *http.Request) {
	if !r.opt.CanReboot {
		r.log.Warn("Reboot is not allowed")
		http.Error(w, "reboot is not allowed", http.StatusForbidden)
		return
	}

	var body rebootBody
	if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
		r.log.Warnf("Failed to decode request body: %v", err)
		http.Error(w, "failed to decode request body", http.StatusBadRequest)
		return
	}

	if body.RunOnce == "" {
		http.Error(w, "runOnce is required", http.StatusBadRequest)
		return
	}

	if err := sys.RunOnce(body.RunOnce); err != nil {
		r.log.Warnf("Failed to set %s to runOnce: %v", body.RunOnce, err)
		http.Error(w, "failed to set runOnce", http.StatusInternalServerError)
		return
	}

	if err := sys.Reboot(); err != nil {
		r.log.Warnf("Failed to reboot system: %v", err)
		http.Error(w, "failed to reboot system", http.StatusInternalServerError)
	}
}

func (r *restful) enableFeature(w http.ResponseWriter, req *http.Request) {
	if !r.opt.CanEnableFeature {
		r.log.Warn("Enable feature is not allowed")
		http.Error(w, "enable feature is not allowed", http.StatusForbidden)
		return
	}

	if err := wsl.Install(r.opt, r.log); err != nil {
		r.log.Warnf("Failed to enable feature: %v", err)
		http.Error(w, "failed to enable feature", http.StatusInternalServerError)
		return
	}
}

func (r *restful) updateWSL(w http.ResponseWriter, req *http.Request) {
	if !r.opt.CanUpdateWSL {
		r.log.Warn("Update WSL is not allowed")
		http.Error(w, "update WSL is not allowed", http.StatusForbidden)
		return
	}

	if err := wsl.Update(r.opt, r.log); err != nil {
		r.log.Warnf("Failed to update WSL: %v", err)
		http.Error(w, "failed to update WSL", http.StatusInternalServerError)
		return
	}

	channel.NotifyWSLEnvReady()
}

func (r *restful) requestStop(w http.ResponseWriter, req *http.Request) {
	if err := wsl.RequestStop(r.log, r.opt.DistroName); err != nil {
		r.log.Warnf("Failed to request stop: %v", err)
		http.Error(w, "failed to request stop", http.StatusInternalServerError)
		return
	}
}

func (r *restful) stop(w http.ResponseWriter, req *http.Request) {
	if err := wsl.Stop(r.log, r.opt.DistroName); err != nil {
		r.log.Warnf("Failed to stop: %v", err)
		http.Error(w, "failed to stop", http.StatusInternalServerError)
		return
	}
}

func middlewareLog(log *logger.Context, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		log.Infof("RESTful server: received request: %s", req.URL.Path)
		next.ServeHTTP(w, req)
		log.Infof("RESTful server: finished request: %s", req.URL.Path)
	}
}

func mustGet(log *logger.Context, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodGet {
			log.Warnf("RESTful server: %s is not allowed in %s", req.Method, req.URL.Path)
			http.Error(w, "get only", http.StatusBadRequest)
		} else {
			next.ServeHTTP(w, req)
		}
	}
}

func mustPost(log *logger.Context, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodPost {
			log.Warnf("RESTful server: %s is not allowed in %s", req.Method, req.URL.Path)
			http.Error(w, "post only", http.StatusBadRequest)
		} else {
			next.ServeHTTP(w, req)
		}
	}
}

func mustPut(log *logger.Context, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodPut {
			log.Warnf("RESTful server: %s is not allowed in %s", req.Method, req.URL.Path)
			http.Error(w, "put only", http.StatusBadRequest)
		} else {
			next.ServeHTTP(w, req)
		}
	}
}
