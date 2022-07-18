package trace

import (
	"net/http"
	"time"
)

func (t *Tracer) RoundTripper(rt http.RoundTripper) http.RoundTripper {
	if rt == nil {
		rt = http.DefaultTransport
	}
	return &traceRoundTripper{t: t, rt: rt}
}

type traceRoundTripper struct {
	t  *Tracer
	rt http.RoundTripper
}

func (trt *traceRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	start := time.Now()

	resp, err := trt.rt.RoundTrip(req)

	duration := time.Since(start)
	ctx := req.Context()
	path, _ := ctx.Value(pathKey).(string)
	hm := &baseMetric{
		path:     path,
		query:    req.Method + " " + req.URL.String(),
		duration: float64(duration.Microseconds()) / 1000,
	}
	go func() {
		trt.t.hmsMutex.Lock()
		trt.t.hms = append(trt.t.hms, hm)
		trt.t.hmsMutex.Unlock()
	}()
	return resp, err
}
