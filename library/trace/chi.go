package trace

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
)

type ChiWrapper struct {
	t *Tracer
	chi.Router
}

func (cw *ChiWrapper) trace(method string, pattern string, h http.HandlerFunc) http.HandlerFunc {
	path := fmt.Sprintf("%s %s", method, pattern)
	return func(w http.ResponseWriter, r *http.Request) {
		r = r.WithContext(
			context.WithValue(r.Context(), pathKey, path),
		)

		start := time.Now()
		h(w, r)
		duration := time.Since(start)

		rm := &requestMetric{
			path:     path,
			duration: float64(duration.Microseconds()) / 1000,
		}
		go func() {
			cw.t.rmsMutex.Lock()
			cw.t.rms = append(cw.t.rms, rm)
			cw.t.rmsMutex.Unlock()
		}()
	}
}

func (cw *ChiWrapper) Connect(pattern string, h http.HandlerFunc) {
	cw.Router.Connect(pattern, cw.trace(http.MethodConnect, pattern, h))
}

func (cw *ChiWrapper) Delete(pattern string, h http.HandlerFunc) {
	cw.Router.Delete(pattern, cw.trace(http.MethodDelete, pattern, h))
}

func (cw *ChiWrapper) Get(pattern string, h http.HandlerFunc) {
	cw.Router.Get(pattern, cw.trace(http.MethodGet, pattern, h))
}

func (cw *ChiWrapper) Head(pattern string, h http.HandlerFunc) {
	cw.Router.Head(pattern, cw.trace(http.MethodHead, pattern, h))
}

func (cw *ChiWrapper) Options(pattern string, h http.HandlerFunc) {
	cw.Router.Options(pattern, cw.trace(http.MethodOptions, pattern, h))
}

func (cw *ChiWrapper) Patch(pattern string, h http.HandlerFunc) {
	cw.Router.Patch(pattern, cw.trace(http.MethodPatch, pattern, h))
}

func (cw *ChiWrapper) Post(pattern string, h http.HandlerFunc) {
	cw.Router.Post(pattern, cw.trace(http.MethodPost, pattern, h))
}

func (cw *ChiWrapper) Put(pattern string, h http.HandlerFunc) {
	cw.Router.Put(pattern, cw.trace(http.MethodPut, pattern, h))
}

func (cw *ChiWrapper) Trace(pattern string, h http.HandlerFunc) {
	cw.Router.Trace(pattern, cw.trace(http.MethodTrace, pattern, h))
}
