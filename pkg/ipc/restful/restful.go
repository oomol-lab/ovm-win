// SPDX-FileCopyrightText: 2024 OOMOL, Inc. <https://www.oomol.com>
// SPDX-License-Identifier: MPL-2.0

package restful

import (
	"context"
	"fmt"
	"net/http"

	"github.com/oomol-lab/ovm-win/pkg/cli"
	"github.com/oomol-lab/ovm-win/pkg/logger"
	"github.com/oomol-lab/ovm-win/pkg/winapi/npipe"
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

	r.log.Infof("restful server is ready to run on %s", r.opt.RestfulEndpoint)

	go func() {
		<-ctx.Done()
		_ = nl.Close()
		r.log.Info("restful server is shutting down, because the context is done")
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
	mux.Handle("/ping", middlewareLog(r.log, ping))
	return mux
}

func ping(w http.ResponseWriter, _ *http.Request) {
	_, _ = w.Write([]byte("pong"))
}

func middlewareLog(log *logger.Context, next http.HandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Infof("restful server: received request: %s", r.URL.Path)
		next.ServeHTTP(w, r)
		log.Infof("restful server: finished request: %s", r.URL.Path)
	})
}
