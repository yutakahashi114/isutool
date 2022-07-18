package trace

import (
	"bytes"
	"context"
	"encoding/csv"
	"log"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/dustin/go-humanize"
	"golang.org/x/exp/slices"
)

type Tracer struct {
	rmsMutex sync.Mutex
	rms      requestMetrics

	smsMutex sync.Mutex
	sms      sqlMetrics

	hmsMutex sync.Mutex
	hms      httpMetrics

	filePrefix string

	*SQLLogger
}

func New(filePrefix string) *Tracer {
	t := &Tracer{
		rms:        make(requestMetrics, 0, 100000),
		sms:        make(sqlMetrics, 0, 2500000),
		hms:        make(httpMetrics, 0, 100000),
		filePrefix: filePrefix,
	}
	t.SQLLogger = newSQLLogger(t)
	return t
}

func (t *Tracer) Start(duration time.Duration, db queryDB) {
	t.StartTrace(duration, db)
	t.StartProfileCPU(duration)
	t.StartProfileMem(duration)
}

func (t *Tracer) StartTrace(duration time.Duration, db queryDB) {
	t.rms = t.rms[:0]
	t.sms = t.sms[:0]
	t.hms = t.hms[:0]
	t.bindedSQLLogger.queries = t.bindedSQLLogger.queries[:0]
	t.bindedSQLLogger.sampled = make(map[string]struct{}, 30)
	go func() {
		time.Sleep(duration)
		t.writeStats()
		t.bindedSQLLogger.writeExplain(db, t.filePrefix)
		log.Println("trace finish")
	}()
}

func (t *Tracer) SubStart(ctx context.Context, query string) (stop func()) {
	start := time.Now()
	return func() {
		duration := time.Since(start)
		path, _ := ctx.Value(pathKey).(string)
		sm := &baseMetric{
			path:     path,
			query:    query,
			duration: float64(duration.Microseconds() / 1000),
		}
		go func() {
			t.smsMutex.Lock()
			t.sms = append(t.sms, sm)
			t.smsMutex.Unlock()
		}()
	}
}

type ctxKey struct{ string }

var pathKey = &ctxKey{"path"}

// func (t *Tracer) setPath(path string) {
// 	goroutineID := getGoroutineID()

// 	t.goroutineIDMutex.RLock()
// 	p, ok := t.goroutineIDMap[goroutineID]
// 	t.goroutineIDMutex.RUnlock()
// 	if !ok {
// 		t.goroutineIDMutex.Lock()
// 		p, ok = t.goroutineIDMap[goroutineID]
// 		if !ok {
// 			s := ""
// 			p = &s
// 			t.goroutineIDMap[goroutineID] = p
// 		}
// 		t.goroutineIDMutex.Unlock()
// 	}
// 	*p = path
// }

// func (t *Tracer) getPath() string {
// 	goroutineID := getGoroutineID()

// 	t.goroutineIDMutex.RLock()
// 	p, ok := t.goroutineIDMap[goroutineID]
// 	t.goroutineIDMutex.RUnlock()
// 	if !ok {
// 		return ""
// 	}
// 	return *p
// }

var goroutineBytes = []byte("goroutine ")

func getGoroutineID() string {
	bs := make([]byte, 64)

	bs = bs[:runtime.Stack(bs, false)]
	bs = bytes.TrimPrefix(bs, goroutineBytes)
	bs = bs[:bytes.IndexByte(bs, ' ')]

	return string(bs)
}

type requestMetric struct {
	path     string
	duration float64
}

type requestMetrics []*requestMetric

type baseMetric struct {
	path     string
	query    string
	duration float64
}

type baseMetrics []*baseMetric

type sqlMetrics baseMetrics

type httpMetrics baseMetrics

type requestStat struct {
	path      string
	count     int
	min       float64
	max       float64
	mean      float64
	total     float64
	sqlStats  []baseStat
	httpStats []baseStat
}

type baseStat struct {
	path  string
	query string
	count int
	min   float64
	max   float64
	mean  float64
	total float64
	cpr   float64 // count per request
}

const queryLenLimit = 1000

func (t *Tracer) writeStats() error {
	rss := t.rms.makeRequestStats(t.sms, t.hms)
	f, err := os.Create(t.filePrefix + "_trace.tsv")
	if err != nil {
		return err
	}
	defer f.Close()
	w := csv.NewWriter(f)
	w.Comma = '\t'
	w.Write([]string{"path", "count", "total(ms)", "mean(ms)", "min(ms)", "max(ms)", "query_count/request_count", "query"})
	for _, rs := range rss {
		w.Write([]string{
			rs.path,
			strconv.Itoa(rs.count),
			formatFloat(rs.total),
			formatFloat(rs.mean),
			formatFloat(rs.min),
			formatFloat(rs.max),
			"-",
			"-",
		})
		if len(rs.sqlStats) > 0 {
			var sqlTotal float64
			for _, ss := range rs.sqlStats {
				sqlTotal += ss.total
			}
			w.Write([]string{
				"- sql total time",
				"-",
				formatFloat(sqlTotal),
				"-",
				"-",
				"-",
				"-",
				"-",
			})
		}
		for _, ss := range rs.sqlStats {
			q := ss.query
			if len(q) > queryLenLimit {
				q = q[:queryLenLimit]
			}
			w.Write([]string{
				"-",
				strconv.Itoa(ss.count),
				formatFloat(ss.total),
				formatFloat(ss.mean),
				formatFloat(ss.min),
				formatFloat(ss.max),
				formatFloat(ss.cpr),
				q,
			})
		}

		if len(rs.httpStats) > 0 {
			var httpTotal float64
			for _, ss := range rs.httpStats {
				httpTotal += ss.total
			}
			w.Write([]string{
				"- http total time",
				"-",
				formatFloat(httpTotal),
				"-",
				"-",
				"-",
				"-",
				"-",
			})
		}
		for _, ss := range rs.httpStats {
			q := ss.query
			if len(q) > queryLenLimit {
				q = q[:queryLenLimit]
			}
			w.Write([]string{
				"-",
				strconv.Itoa(ss.count),
				formatFloat(ss.total),
				formatFloat(ss.mean),
				formatFloat(ss.min),
				formatFloat(ss.max),
				formatFloat(ss.cpr),
				q,
			})
		}
	}

	w.Write([]string{"request", "", "", "", "", "", "", ""})
	for _, rs := range rss {
		w.Write([]string{
			rs.path,
			strconv.Itoa(rs.count),
			formatFloat(rs.total),
			formatFloat(rs.mean),
			formatFloat(rs.min),
			formatFloat(rs.max),
			"-",
			"-",
		})
	}

	w.Write([]string{"sql", "", "", "", "", "", "", ""})
	for _, rs := range baseMetrics(t.sms).makeStats() {
		w.Write([]string{
			"-",
			strconv.Itoa(rs.count),
			formatFloat(rs.total),
			formatFloat(rs.mean),
			formatFloat(rs.min),
			formatFloat(rs.max),
			"-",
			rs.query,
		})
	}
	w.Flush()

	return nil
}

func formatFloat(f float64) string {
	str := humanize.Commaf(f) + "000"
	i := strings.Index(str, ".")
	if i == -1 {
		return humanize.Commaf(f) + ".000"
	}
	if i+4 >= len(str) {
		return humanize.Commaf(f)
	}
	return str[:i+4]
	// return strconv.FormatFloat(f, 'f', 4, 64)
}

func (rms requestMetrics) makeRequestStats(sms sqlMetrics, hms httpMetrics) []requestStat {
	sqlStatsMap := baseMetrics(sms).makeStatsMap()
	httpStatsMap := baseMetrics(hms).makeStatsMap()
	statMap := make(map[string]requestStat, 30)
	for _, rm := range rms {
		s, _ := statMap[rm.path]

		s.count++
		if rm.duration > s.max {
			s.max = rm.duration
		}
		if rm.duration < s.min || s.min == 0 {
			s.min = rm.duration
		}
		s.total += rm.duration

		statMap[rm.path] = s
	}

	ss := make([]requestStat, 0, len(statMap))
	for path, s := range statMap {
		s.path = path
		s.mean = s.total / float64(s.count)
		{
			sqlStats := sqlStatsMap[path]
			sc := float64(s.count)
			for i := range sqlStats {
				sqlStats[i].cpr = float64(sqlStats[i].count) / sc
			}
			s.sqlStats = sqlStats
			delete(sqlStatsMap, path)
		}
		{
			httpStats := httpStatsMap[path]
			sc := float64(s.count)
			for i := range httpStats {
				httpStats[i].cpr = float64(httpStats[i].count) / sc
			}
			s.httpStats = httpStats
			delete(httpStatsMap, path)
		}

		ss = append(ss, s)
	}
	slices.SortFunc(ss, func(a, b requestStat) bool {
		return a.total > b.total
	})
	for _, sqlStats := range sqlStatsMap {
		ss = append(ss, requestStat{
			path:     "unknown",
			sqlStats: sqlStats,
		})
	}
	for _, httpStats := range httpStatsMap {
		ss = append(ss, requestStat{
			path:      "unknown",
			httpStats: httpStats,
		})
	}
	return ss
}

func (bms baseMetrics) makeStatsMap() map[string][]baseStat {
	statMapPerPathMap := make(map[string]map[string]baseStat, 10)
	for _, bm := range bms {
		statMapPerPath, ok := statMapPerPathMap[bm.path]
		if !ok {
			statMapPerPath = make(map[string]baseStat, 10)
		}
		s, _ := statMapPerPath[bm.query]

		s.count++
		if bm.duration > s.max {
			s.max = bm.duration
		}
		if bm.duration < s.min || s.min == 0 {
			s.min = bm.duration
		}
		s.total += bm.duration

		statMapPerPath[bm.query] = s
		statMapPerPathMap[bm.path] = statMapPerPath
	}

	statsMap := make(map[string][]baseStat, len(statMapPerPathMap))
	for path, statMapPerPath := range statMapPerPathMap {
		stats := make([]baseStat, 0, len(statMapPerPath))
		for query, s := range statMapPerPath {
			s.query = query
			s.mean = s.total / float64(s.count)
			stats = append(stats, s)
		}
		slices.SortFunc(stats, func(a, b baseStat) bool {
			return a.total > b.total
		})
		statsMap[path] = stats
	}
	return statsMap
}

func (bms baseMetrics) makeStats() []baseStat {
	statMap := make(map[string]baseStat, 30)
	for _, rm := range bms {
		s, _ := statMap[rm.query]

		s.count++
		if rm.duration > s.max {
			s.max = rm.duration
		}
		if rm.duration < s.min || s.min == 0 {
			s.min = rm.duration
		}
		s.total += rm.duration

		statMap[rm.query] = s
	}

	ss := make([]baseStat, 0, len(statMap))
	for query, s := range statMap {
		s.query = query
		s.mean = s.total / float64(s.count)
		ss = append(ss, s)
	}
	slices.SortFunc(ss, func(a, b baseStat) bool {
		return a.total > b.total
	})
	return ss
}
