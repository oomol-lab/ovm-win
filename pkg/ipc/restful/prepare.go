// SPDX-FileCopyrightText: 2024 OOMOL, Inc. <https://www.oomol.com>
// SPDX-License-Identifier: MPL-2.0

package restful

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/oomol-lab/ovm-win/pkg/channel"
	"github.com/oomol-lab/ovm-win/pkg/logger"
	"github.com/oomol-lab/ovm-win/pkg/types"
	"github.com/oomol-lab/ovm-win/pkg/winapi/npipe"
	"github.com/oomol-lab/ovm-win/pkg/winapi/sys"
	"github.com/oomol-lab/ovm-win/pkg/wsl"
)

type routerPrepare struct {
	opt *types.PrepareOpt
	log *logger.Context
}

func SetupPrepare(opt *types.PrepareOpt) (s Server, err error) {
	rp := &routerPrepare{
		opt: opt,
		log: opt.Logger,
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

func (r *routerPrepare) Close() error {
	return nil
}

func (r *routerPrepare) mux() http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/reboot", mustPost(r.log, middlewareLog(r.log, r.reboot)))
	mux.Handle("/enable-feature", mustPost(r.log, middlewareLog(r.log, r.enableFeature)))
	mux.Handle("/update-wsl", mustPut(r.log, middlewareLog(r.log, r.updateWSL)))

	return mux
}

type rebootBody struct {
	// RunOnce is the command to run after the next system startup
	RunOnce string `json:"runOnce"`
}

func (r *routerPrepare) reboot(w http.ResponseWriter, req *http.Request) {
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

func (r *routerPrepare) enableFeature(w http.ResponseWriter, req *http.Request) {
	if !r.opt.CanEnableFeature {
		r.log.Warn("Enable feature is not allowed")
		http.Error(w, "enable feature is not allowed", http.StatusForbidden)
		return
	}

	if err := wsl.Install(r.opt); err != nil {
		r.log.Warnf("Failed to enable feature: %v", err)
		http.Error(w, "failed to enable feature", http.StatusInternalServerError)
		return
	}
}

func (r *routerPrepare) updateWSL(w http.ResponseWriter, req *http.Request) {
	if !r.opt.CanUpdateWSL {
		r.log.Warn("Update WSL is not allowed")
		http.Error(w, "update WSL is not allowed", http.StatusForbidden)
		return
	}

	if err := wsl.Update(r.opt); err != nil {
		r.log.Warnf("Failed to update WSL: %v", err)
		http.Error(w, "failed to update WSL", http.StatusInternalServerError)
		return
	}

	channel.NotifyWSLEnvReady()
}
