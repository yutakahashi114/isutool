package trace

import (
	"bytes"
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/olekukonko/tablewriter"
	sqldblogger "github.com/simukti/sqldb-logger"
)

func (t *Tracer) DB(dsn string, driver driver.Driver) *sql.DB {
	return sqldblogger.OpenDriver(
		dsn,
		driver,
		t.SQLLogger,
	)
}

type SQLLogger struct {
	t               *Tracer
	bindedSQLLogger *bindedSQLLogger
}

func newSQLLogger(t *Tracer) *SQLLogger {
	return &SQLLogger{
		t: t,
		bindedSQLLogger: &bindedSQLLogger{
			m:             sync.RWMutex{},
			sampled:       make(map[string]struct{}, 30),
			queries:       make([]*queryArgs, 0, 30),
			strReplacer:   strings.NewReplacer("'", "''"),
			queryReplacer: strings.NewReplacer("?", "%v"),
		},
	}
}

var queryAliasKey = &ctxKey{"queryAliasKey"}

func (t *Tracer) QueryAlias(ctx context.Context, a string) context.Context {
	return context.WithValue(ctx, queryAliasKey, a)
}

func (l *SQLLogger) Log(
	ctx context.Context,
	level sqldblogger.Level,
	msg string,
	d map[string]interface{},
) {
	var query string
	var arg interface{}
	var duration float64
	for k, v := range d {
		switch k {
		case "query":
			query, _ = v.(string)
		case "args":
			arg = v
		case "duration":
			duration, _ = v.(float64)
		}
	}
	var alias string
	if alias, _ = ctx.Value(queryAliasKey).(string); alias == "" {
		alias = query
	}

	path, _ := ctx.Value(pathKey).(string)
	sm := &baseMetric{
		path:     path,
		query:    msg + " " + alias,
		duration: duration,
	}
	go func() {
		l.t.smsMutex.Lock()
		l.t.sms = append(l.t.sms, sm)
		l.t.smsMutex.Unlock()
	}()
	if msg == "PrepareContext" || msg == "StmtClose" {
		return
	}
	go l.bindedSQLLogger.bindedQuery(query, arg)
}

type bindedSQLLogger struct {
	m             sync.RWMutex
	sampled       map[string]struct{}
	queries       []*queryArgs
	strReplacer   *strings.Replacer
	queryReplacer *strings.Replacer
	db            queryDB
}
type queryArgs struct {
	query string
	args  []any
}

func (l *bindedSQLLogger) bindedQuery(
	query string,
	arg interface{},
) {
	l.m.RLock()
	_, ok := l.sampled[query]
	l.m.RUnlock()
	if ok {
		return
	}
	l.m.Lock()
	if _, ok := l.sampled[query]; ok {
		l.m.Unlock()
		return
	}
	l.sampled[query] = struct{}{}
	l.m.Unlock()
	if query == "" {
		return
	}
	if args, ok := arg.([]any); ok {
		l.queries = append(l.queries, &queryArgs{query: query, args: args})
	} else {
		l.queries = append(l.queries, &queryArgs{query: query})
	}
}

func (l *bindedSQLLogger) writeExplain(db queryDB, filePrefix string) error {
	all, err := os.Create(filePrefix + "_explain.sql")
	if err != nil {
		return err
	}
	defer all.Close()
	buf := &bytes.Buffer{}
	buf.Grow(1000 * len(l.queries))
	for _, qs := range l.queries {
		if err := l.explain(db, qs, buf); err != nil {
			log.Println(err)
		}
	}
	_, err = buf.WriteTo(all)
	if err != nil {
		return err
	}
	return err
}

type explainResult struct {
	id           int
	selectType   sql.NullString
	table        sql.NullString
	partitions   sql.NullString
	joinType     sql.NullString
	possibleKeys sql.NullString
	key          sql.NullString
	keyLen       sql.NullInt32
	ref          sql.NullString
	rows         sql.NullInt64
	filtered     sql.NullFloat64
	extra        sql.NullString
}

type queryDB interface {
	Query(query string, args ...any) (*sql.Rows, error)
}

func (l *bindedSQLLogger) bind(qa *queryArgs) string {
	args := make([]any, 0, len(qa.args))
	for _, arg := range qa.args {
		var a interface{}
		switch arg := arg.(type) {
		case string:
			str := l.strReplacer.Replace(arg)
			if len(str) > 100 {
				str = str[:100]
			}
			a = "'" + str + "'"
		case time.Time:
			a = "'" + arg.Format("2006-01-02 15:04:05") + "'"
		default:
			a = arg
		}
		args = append(args, a)
	}
	return fmt.Sprintf(l.queryReplacer.Replace(qa.query), args...)
}

func (l *bindedSQLLogger) explain(db queryDB, qa *queryArgs, buf *bytes.Buffer) error {
	if len(qa.query) > queryLenLimit {
		buf.WriteString(qa.query[:queryLenLimit])
		buf.WriteString("\ntoo long query\n\n")
		return nil
	}
	buf.WriteString(l.bind(qa))
	buf.WriteString(" -- ")
	buf.WriteString(qa.query)
	buf.WriteString("\n")

	q := qa.query
	// cf. https://techblog.zozo.com/entry/mysql-explain-plan-examination
	// 検査対象をSELECT、UPDATE、DELETEに限定
	pre := strings.ToUpper(q[:6])
	if !strings.HasPrefix(pre, "SELECT") && !strings.HasPrefix(pre, "UPDATE") && !strings.HasPrefix(pre, "DELETE") {
		buf.WriteString("\n")
		return nil
	}

	// EXPLAINの実行
	rows, err := db.Query("EXPLAIN "+q, qa.args...)
	if err != nil {
		return err
	}

	// EXPLAINの出力結果を元に警告を作成
	warnings := []string{}
	results := []explainResult{}
	for rows.Next() {
		var r explainResult
		err := rows.Scan(&r.id, &r.selectType, &r.table, &r.partitions, &r.joinType, &r.possibleKeys, &r.key, &r.keyLen, &r.ref, &r.rows, &r.filtered, &r.extra)
		if err != nil {
			return err
		}
		if r.joinType.Valid && (r.joinType.String == "ALL" || r.joinType.String == "index") && r.extra.Valid && strings.Contains(r.extra.String, "Using where") {
			warnings = append(warnings, "絞り込みに必要なインデックスが不足している可能性があります")
		}
		if strings.Contains(r.extra.String, "Using filesort") {
			warnings = append(warnings, "インデックスが用いられていないソート処理が行われています")
		}
		if strings.Contains(r.extra.String, "Using temporary") {
			warnings = append(warnings, "クエリの実行に一時テーブルを必要としています")
		}
		results = append(results, r)
	}
	if len(warnings) == 0 {
		buf.WriteString("no warning\n")
		return nil
	}

	for _, warning := range warnings {
		buf.WriteString(warning)
		buf.WriteString("\n")
	}

	table := tablewriter.NewWriter(buf)
	table.SetHeader([]string{"id", "select_type", "table", "partitions", "type", "possible_keys", "key", "key_len", "ref", "rows", "filtered", "extra"})
	for _, r := range results {
		table.Append([]string{
			strconv.Itoa(r.id),
			mapFromNullString(r.selectType),
			mapFromNullString(r.table),
			mapFromNullString(r.partitions),
			mapFromNullString(r.joinType),
			mapFromNullString(r.possibleKeys),
			mapFromNullString(r.key),
			mapFromNullInt32(r.keyLen),
			mapFromNullString(r.ref),
			mapFromNullInt64(r.rows),
			mapFromNullFloat64(r.filtered),
			mapFromNullString(r.extra),
		})
	}
	table.Render()
	buf.WriteString("\n")

	return nil
}

func mapFromNullString(s sql.NullString) string {
	if !s.Valid {
		return "NULL"
	}
	return s.String
}

func mapFromNullInt32(s sql.NullInt32) string {
	if !s.Valid {
		return "NULL"
	}
	return strconv.Itoa(int(s.Int32))
}

func mapFromNullInt64(s sql.NullInt64) string {
	if !s.Valid {
		return "NULL"
	}
	return strconv.Itoa(int(s.Int64))
}

func mapFromNullFloat64(s sql.NullFloat64) string {
	if !s.Valid {
		return "NULL"
	}
	return fmt.Sprintf("%f", s.Float64)
}
