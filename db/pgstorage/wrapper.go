package pgstorage

import (
	"context"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/0xPolygonHermez/zkevm-node/log"
	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v4"
)

const (
	defaultDBTimeout = 5 * time.Second
)

var (
	queryCnt atomic.Int64
)

// execQuerierWrapper automatically adds a ctx timeout for the querier, also add before and after logs
type execQuerierWrapper struct {
	execQuerier
}

func (w *execQuerierWrapper) Exec(ctx context.Context, sql string, arguments ...interface{}) (commandTag pgconn.CommandTag, err error) {
	dbCtx, cancel := getCtxWithTimeout(ctx, defaultDBTimeout)
	defer func() {
		if cancel != nil {
			cancel()
		}
	}()

	i := queryCnt.Add(1)
	log.Debug(i, queryCnt.Load())
	logger := log.WithFields("logid", fmt.Sprintf("db_query_%v", i))
	startTime := time.Now()
	logger.Debugf("Exec begin, sql[%v], arguments[%v]", removeNewLine(sql), arguments)

	tag, err := w.execQuerier.Exec(dbCtx, sql, arguments...)

	logger.Debugf("Exec end, err[%v] processTime[%v]", err, time.Since(startTime).String())
	return tag, err
}

func (w *execQuerierWrapper) Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
	dbCtx, cancel := getCtxWithTimeout(ctx, defaultDBTimeout)
	defer func() {
		if cancel != nil {
			cancel()
		}
	}()

	i := queryCnt.Add(1)
	logger := log.WithFields("logid", fmt.Sprintf("db_query_%v", i))
	startTime := time.Now()
	logger.Debugf("Query begin, sql[%v], arguments[%v]", removeNewLine(sql), args)

	rows, err := w.execQuerier.Query(dbCtx, sql, args...)

	logger.Debugf("Query end, err[%v] processTime[%v]", err, time.Since(startTime).String())
	return rows, err
}

func (w *execQuerierWrapper) QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row {
	dbCtx, cancel := getCtxWithTimeout(ctx, defaultDBTimeout)
	defer func() {
		if cancel != nil {
			cancel()
		}
	}()

	i := queryCnt.Add(1)
	logger := log.WithFields("logid", fmt.Sprintf("db_query_%v", i))
	startTime := time.Now()
	logger.Debugf("QueryRow begin, sql[%v], arguments[%v]", removeNewLine(sql), args)

	row := w.execQuerier.QueryRow(dbCtx, sql, args...)

	logger.Debugf("QueryRow end, processTime[%v]", time.Since(startTime).String())
	return row
}

func (w *execQuerierWrapper) CopyFrom(ctx context.Context, tableName pgx.Identifier, columnNames []string, rowSrc pgx.CopyFromSource) (int64, error) {
	dbCtx, cancel := getCtxWithTimeout(ctx, defaultDBTimeout)
	defer func() {
		if cancel != nil {
			cancel()
		}
	}()

	i := queryCnt.Add(1)
	logger := log.WithFields("logid", fmt.Sprintf("db_query_%v", i))
	startTime := time.Now()
	logger.Debugf("CopyFrom begin, tableName[%v]", tableName)

	res, err := w.execQuerier.CopyFrom(dbCtx, tableName, columnNames, rowSrc)

	logger.Debugf("CopyFrom end, res[%v] err[%v] processTime[%v]", res, err, time.Since(startTime).String())
	return res, err
}

func getCtxWithTimeout(ctx context.Context, timeout time.Duration) (context.Context, func()) {
	if _, hasDeadline := ctx.Deadline(); hasDeadline {
		return ctx, nil
	}
	return context.WithTimeout(ctx, timeout)
}

func removeNewLine(s string) string {
	return strings.Replace(s, "\n", " ", -1)
}
