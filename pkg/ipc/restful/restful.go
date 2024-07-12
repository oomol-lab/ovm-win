// SPDX-FileCopyrightText: 2024 OOMOL, Inc. <https://www.oomol.com>
// SPDX-License-Identifier: MPL-2.0

package restful

import (
	"fmt"
	"io"
	"net"
	"net/http"

	"github.com/oomol-lab/ovm-win/pkg/logger"
)

type Server interface {
	Run() error
	io.Closer
}

type router interface {
	mux() http.Handler
	io.Closer
}

type server struct {
	router router
	log    *logger.Context

	nl net.Listener
}

func (s *server) Run() error {
	s.log.Infof("RESTful server is ready to run on %s", s.nl.Addr().String())

	server := &http.Server{
		Handler: s.router.mux(),
	}
	if err := server.Serve(s.nl); err != nil {
		return fmt.Errorf("failed to serve restful server: %w", err)
	}

	return nil
}

func (s *server) Close() error {
	_ = s.router.Close()
	_ = s.nl.Close()

	return nil
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
