package api

import (
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"
)

var errNotImplemented = errors.New("not implemented")

// HTTPError carries an HTTP status code and message through the handler chain.
type HTTPError struct {
	Code int
	Msg  string
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("%d: %s", e.Code, e.Msg)
}

// isUniqueViolation reports whether err is a PostgreSQL unique constraint violation (23505).
func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
