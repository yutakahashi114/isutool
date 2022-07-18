package trace

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"goji.io"
	"goji.io/pat"
)

func (t *Tracer) Goji(g *goji.Mux) *GojiWrapper {
	return &GojiWrapper{t, g}
}

type GojiWrapper struct {
	t *Tracer
	*goji.Mux
}

func (gw *GojiWrapper) trace(method string, pattern string, h http.HandlerFunc) http.HandlerFunc {
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
			gw.t.rmsMutex.Lock()
			gw.t.rms = append(gw.t.rms, rm)
			gw.t.rmsMutex.Unlock()
		}()
	}
}

func (gw *GojiWrapper) HandleFunc(p goji.Pattern, h func(http.ResponseWriter, *http.Request)) {
	pattern, ok := p.(*pat.Pattern)
	if !ok {
		gw.Mux.HandleFunc(p, h)
		return
	}
	var method string
	for m := range pattern.HTTPMethods() {
		method += m
	}
	gw.Mux.HandleFunc(p, gw.trace(method, pattern.String(), h))
}

func (gw *GojiWrapper) Handle(p goji.Pattern, h http.Handler) {
	pattern, ok := p.(*pat.Pattern)
	if !ok {
		gw.Mux.Handle(p, h)
		return
	}
	var method string
	for m := range pattern.HTTPMethods() {
		method += m
	}
	gw.Mux.Handle(p, gw.trace(method, pattern.String(), h.ServeHTTP))
}
