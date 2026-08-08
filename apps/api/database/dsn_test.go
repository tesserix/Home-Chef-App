package database

// Prod routes every API pod through the CNPG PgBouncer pooler in transaction
// mode, where pgx's default named prepared statements collide across pooled
// server connections ("prepared statement already exists"). The DSN has to
// force an exec mode that never names a statement.

import "testing"

func TestPoolerSafeDSN(t *testing.T) {
	cases := []struct {
		name string
		dsn  string
		want string
	}{
		{
			"keyword dsn gets the exec mode appended",
			"host=db port=5432 user=homechef dbname=homechef_db sslmode=require",
			"host=db port=5432 user=homechef dbname=homechef_db sslmode=require default_query_exec_mode=exec",
		},
		{
			"url dsn gets it as a query parameter",
			"postgres://homechef@db:5432/homechef_db",
			"postgres://homechef@db:5432/homechef_db?default_query_exec_mode=exec",
		},
		{
			"url dsn with existing parameters keeps them",
			"postgres://homechef@db:5432/homechef_db?sslmode=require",
			"postgres://homechef@db:5432/homechef_db?sslmode=require&default_query_exec_mode=exec",
		},
		{
			"an explicit mode is left alone",
			"host=db default_query_exec_mode=cache_statement",
			"host=db default_query_exec_mode=cache_statement",
		},
		{
			"empty dsn is left alone",
			"",
			"",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := poolerSafeDSN(tc.dsn); got != tc.want {
				t.Errorf("poolerSafeDSN(%q) = %q, want %q", tc.dsn, got, tc.want)
			}
		})
	}
}
