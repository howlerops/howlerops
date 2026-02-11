package database

import (
	"context"
	"database/sql"
	"strings"

	"github.com/sirupsen/logrus"
)

// DBQueryer is the interface shared by *sql.DB and *sql.Conn, allowing
// retry logic to transparently switch from a pooled connection to a
// dedicated (pinned) connection when a prepared statement error is detected.
type DBQueryer interface {
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
}

// isPreparedStmtError checks if an error is caused by a connection pooler
// (pgcat, PgBouncer in transaction mode) reassigning the backend connection
// between Parse and Bind/Execute in the PostgreSQL extended query protocol.
func isPreparedStmtError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "prepared statement") && strings.Contains(msg, "does not exist")
}

// retryExec retries an ExecContext call on a pinned *sql.Conn when
// the initial attempt fails with a prepared statement error. This handles
// connection poolers in transaction mode that reassign backends between
// protocol messages.
func retryExec(ctx context.Context, db *sql.DB, logger *logrus.Logger, query string, args ...interface{}) (sql.Result, error) {
	result, err := db.ExecContext(ctx, query, args...)
	if err == nil || !isPreparedStmtError(err) {
		return result, err
	}

	if logger != nil {
		logger.Warn("Prepared statement error on ExecContext, retrying with pinned connection (pooler compatibility)")
	}

	conn, connErr := db.Conn(ctx)
	if connErr != nil {
		return nil, err // return original error
	}
	defer conn.Close()

	return conn.ExecContext(ctx, query, args...)
}

// retryQueryRow retries a QueryRowContext+Scan sequence on a pinned *sql.Conn
// when Scan returns a prepared statement error. The scanFn receives a *sql.Row
// and should call Scan on it.
func retryQueryRow(ctx context.Context, db *sql.DB, logger *logrus.Logger, query string, args []interface{}, scanFn func(*sql.Row) error) error {
	row := db.QueryRowContext(ctx, query, args...)
	err := scanFn(row)
	if err == nil || !isPreparedStmtError(err) {
		return err
	}

	if logger != nil {
		logger.Warn("Prepared statement error on QueryRowContext, retrying with pinned connection (pooler compatibility)")
	}

	conn, connErr := db.Conn(ctx)
	if connErr != nil {
		return err // return original error
	}
	defer conn.Close()

	row = conn.QueryRowContext(ctx, query, args...)
	return scanFn(row)
}

// retryQuery retries a QueryContext call on a pinned *sql.Conn when
// the initial attempt fails with a prepared statement error. The caller
// is responsible for closing the returned *sql.Rows AND the cleanup function.
// The cleanup function closes the pinned connection (if one was used).
func retryQuery(ctx context.Context, db *sql.DB, logger *logrus.Logger, query string, args ...interface{}) (*sql.Rows, func(), error) {
	rows, err := db.QueryContext(ctx, query, args...)
	if err == nil || !isPreparedStmtError(err) {
		return rows, func() {}, err
	}

	if logger != nil {
		logger.Warn("Prepared statement error on QueryContext, retrying with pinned connection (pooler compatibility)")
	}

	conn, connErr := db.Conn(ctx)
	if connErr != nil {
		return nil, func() {}, err // return original error
	}

	rows, err = conn.QueryContext(ctx, query, args...)
	if err != nil {
		conn.Close()
		return nil, func() {}, err
	}

	// Return a cleanup that closes the pinned connection AFTER rows.Close()
	return rows, func() { conn.Close() }, nil
}
