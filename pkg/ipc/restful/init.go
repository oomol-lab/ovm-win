// SPDX-FileCopyrightText: 2024-2025 OOMOL, Inc. <https://www.oomol.com>
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

type routerInit struct {
	opt *types.InitOpt
	log *logger.Context

	canShutdownWSL bool
}

func SetupInit(opt *types.InitOpt) (s Server, err error) {
	rp := &routerInit{
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

func (r *routerInit) Close() error {
	return nil
}

func (r *routerInit) mux() http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/reboot", mustPost(r.log, middlewareLog(r.log, r.reboot)))
	mux.Handle("/enable-feature", mustPost(r.log, middlewareLog(r.log, r.enableFeature)))
	mux.Handle("/update-wsl", mustPut(r.log, middlewareLog(r.log, r.updateWSL)))
	mux.Handle("/fix-wsl-config", mustPut(r.log, middlewareLog(r.log, r.fixWSLConfig)))
	mux.Handle("/shutdown-wsl", mustPut(r.log, middlewareLog(r.log, r.shutdownWSL)))

	return mux
}

type rebootBody struct {
	// RunOnce is the command to run after the next system startup
	RunOnce string `json:"runOnce"`
	// Later is whether to reboot later
	Later bool `json:"later"`
}

func (r *routerInit) reboot(w http.ResponseWriter, req *http.Request) {
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

	if err := sys.RunOnce(r.opt.Name, body.RunOnce); err != nil {
		r.log.Warnf("Failed to set %s to runOnce: %v", body.RunOnce, err)
		http.Error(w, "failed to set runOnce", http.StatusInternalServerError)
		return
	}

	if !body.Later {
		if err := sys.Reboot(); err != nil {
			r.log.Warnf("Failed to reboot system: %v", err)
			http.Error(w, "failed to reboot system", http.StatusInternalServerError)
		}
	}
}

func (r *routerInit) enableFeature(w http.ResponseWriter, req *http.Request) {
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

func (r *routerInit) updateWSL(w http.ResponseWriter, req *http.Request) {
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

	channel.NotifyWSLUpdated()
}

type fixWSLConfigBody struct {
	Method string `json:"method"`
}

func (r *routerInit) fixWSLConfig(w http.ResponseWriter, req *http.Request) {
	if !r.opt.CanFixWSLConfig {
		r.log.Warn("Fix WSL config is not allowed")
		http.Error(w, "fix WSL config is not allowed", http.StatusForbidden)
		return
	}

	var body fixWSLConfigBody
	if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
		r.log.Warnf("Failed to decode request body: %v", err)
		http.Error(w, "failed to decode request body", http.StatusBadRequest)
		return
	}

	wslconfig := wsl.NewConfig(r.log)

	r.log.Infof("Fix WSL config with method: %s", body.Method)

	switch body.Method {
	case "auto":
		if err := wslconfig.Fix(); err != nil {
			r.log.Warnf("Failed to fix WSL config: %v", err)
			http.Error(w, "failed to fix WSL config", http.StatusInternalServerError)
			return
		}

		if err := wsl.Shutdown(r.log); err != nil {
			r.log.Warnf("Failed to shutdown WSL: %v", err)
		}

		channel.NotifyWSLConfigUpdated(wsl.FIX_WSLCONFIG_AUTO)
	case "open":
		r.canShutdownWSL = true

		if err := wslconfig.Open(); err != nil {
			r.log.Warnf("Failed to open WSL config: %v", err)
			http.Error(w, "failed to open WSL config", http.StatusInternalServerError)
			return
		}

		channel.NotifyWSLConfigUpdated(wsl.FIX_WSLCONFIG_OPEN)
	case "skip":
		wsl.SkipConfigCheck(r.opt)
		channel.NotifyWSLConfigUpdated(wsl.FIX_WSLCONFIG_SKIP)
	}
}

func (r *routerInit) shutdownWSL(w http.ResponseWriter, req *http.Request) {
	if !r.canShutdownWSL {
		r.log.Warn("Shutdown WSL is not allowed")
		http.Error(w, "shutdown WSL is not allowed", http.StatusForbidden)
		return
	}

	if err := wsl.Shutdown(r.opt.Logger); err != nil {
		r.log.Warnf("Failed to shutdown WSL: %v", err)
		http.Error(w, "failed to shutdown WSL", http.StatusInternalServerError)
		return
	}

	channel.NotifyWSLShutdown()
	r.canShutdownWSL = false
}
