// SPDX-FileCopyrightText: 2024 OOMOL, Inc. <https://www.oomol.com>
// SPDX-License-Identifier: MPL-2.0

package restful

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/oomol-lab/ovm-win/pkg/cli"
	"github.com/oomol-lab/ovm-win/pkg/logger"
	"github.com/oomol-lab/ovm-win/pkg/winapi/npipe"
	"github.com/oomol-lab/ovm-win/pkg/winapi/sys"
)

type restful struct {
	log *logger.Context
	opt *cli.Context
}

func Run(ctx context.Context, opt *cli.Context, log *logger.Context) error {
	r := &restful{
		log: log,
		opt: opt,
	}

	return r.start(ctx)
}

func (r *restful) start(ctx context.Context) error {
	nl, err := npipe.Create(r.opt.RestfulEndpoint)
	if err != nil {
		return fmt.Errorf("failed to create npipe listener: %w", err)
	}

	r.log.Infof("RESTful server is ready to run on %s", r.opt.RestfulEndpoint)

	go func() {
		<-ctx.Done()
		_ = nl.Close()
		r.log.Info("RESTful server is shutting down, because the context is done")
	}()

	server := &http.Server{
		Handler: r.mux(),
	}

	if err := server.Serve(nl); err != nil && ctx.Err() == nil {
		return fmt.Errorf("failed to serve restful server: %w", err)
	}

	return nil
}

func (r *restful) mux() http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/reboot", mustPost(r.log, middlewareLog(r.log, r.reboot)))
	return mux
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

func middlewareLog(log *logger.Context, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		log.Infof("RESTful server: received request: %s", req.URL.Path)
		next.ServeHTTP(w, req)
		log.Infof("RESTful server: finished request: %s", req.URL.Path)
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
