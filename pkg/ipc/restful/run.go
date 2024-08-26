// SPDX-FileCopyrightText: 2024 OOMOL, Inc. <https://www.oomol.com>
// SPDX-License-Identifier: MPL-2.0

package restful

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/Code-Hex/go-infinity-channel"
	"github.com/oomol-lab/ovm-win/pkg/logger"
	"github.com/oomol-lab/ovm-win/pkg/types"
	"github.com/oomol-lab/ovm-win/pkg/util"
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
	mux.Handle("/exec", mustPost(r.log, middlewareLog(r.log, r.exec)))

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

type execBody struct {
	Command string `json:"command"`
}

func (r *runPrepare) exec(w http.ResponseWriter, req *http.Request) {
	var body execBody
	if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
		r.log.Warnf("Failed to decode request body: %v", err)
		http.Error(w, "failed to decode request body", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	if _, ok := w.(http.Flusher); !ok {
		r.log.Warnf("Bowser does not support server-sent events")
		return
	}

	outCh := infinity.NewChannel[string]()
	errCh := make(chan string, 1)
	doneCh := make(chan struct{}, 1)

	go func() {
		if err := exec(req.Context(), r, body.Command, outCh, errCh); err != nil {
			r.log.Warnf("Failed to execute command: %v", err)
		}

		doneCh <- struct{}{}
		outCh.Close()
		close(errCh)
	}()

	defer func() {
		select {
		case <-req.Context().Done():
			// pass
		default:
			_, _ = fmt.Fprintf(w, "event: done\n")
			_, _ = fmt.Fprintf(w, "data: done\n\n")
			w.(http.Flusher).Flush()
		}
	}()

	for {
		select {
		case <-doneCh:
			r.log.Warnf("Command execution finished")
			return
		case err := <-errCh:
			_, _ = fmt.Fprintf(w, "event: error\n")
			_, _ = fmt.Fprintf(w, "data: %s\n\n", encodeSSE(err))
			w.(http.Flusher).Flush()
			continue
		case out := <-outCh.Out():
			_, _ = fmt.Fprintf(w, "event: out\n")
			_, _ = fmt.Fprintf(w, "data: %s\n\n", encodeSSE(out))
			w.(http.Flusher).Flush()
			continue
		case <-req.Context().Done():
			r.log.Warnf("Client closed connection")
			return
		case <-time.After(3 * time.Second):
			_, _ = fmt.Fprintf(w, ": ping\n\n")
			w.(http.Flusher).Flush()
			continue
		}
	}
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

func exec(ctx context.Context, r *runPrepare, command string, outCh *infinity.Channel[string], errCh chan string) error {
	arg := []string{"-d", r.opt.DistroName, "sh", "-c", command}
	cmd := util.SilentCmdContext(ctx, wsl.Find(), arg...)

	cmd.Env = []string{"WSL_UTF8=1"}
	out := ch2Writer(outCh)
	cmd.Stdout = out
	stderr := recordWriter(out)
	cmd.Stderr = stderr

	if err := cmd.Run(); err != nil {
		newErr := fmt.Errorf("%s\n%s", stderr.LastRecord(), err)
		errCh <- fmt.Sprintf(newErr.Error())
		return fmt.Errorf("run exec command error: %w", newErr)
	}

	return nil
}

type chWriter struct {
	ch *infinity.Channel[string]
	mu sync.Mutex
}

func (w *chWriter) Write(p []byte) (n int, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.ch.In() <- string(p)
	return len(p), nil
}

func ch2Writer(ch *infinity.Channel[string]) io.Writer {
	return &chWriter{
		ch: ch,
	}
}

type writer struct {
	w    io.Writer
	last []byte
}

func (w *writer) Write(p []byte) (n int, err error) {
	w.last = p
	return w.w.Write(p)
}

func (w *writer) LastRecord() string {
	return string(w.last)
}

func recordWriter(w io.Writer) *writer {
	return &writer{
		w: w,
	}
}

func encodeSSE(str string) string {
	return strings.ReplaceAll(strings.TrimSpace(str), "\n", "\ndata: ")
}
