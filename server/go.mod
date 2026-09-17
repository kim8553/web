module github.com/local/9yin-go-server

go 1.23.0

toolchain go1.23.2

require (
	github.com/DATA-DOG/go-sqlmock v1.5.2
	github.com/go-sql-driver/mysql v1.9.3
	golang.org/x/text v0.28.0
)

require filippo.io/edwards25519 v1.1.0 // indirect

require github.com/Hiroko103/go-quicklz v0.0.0-20190115215310-59904abc50d0

replace github.com/DATA-DOG/go-sqlmock => ./_builddeps/sqlmock
