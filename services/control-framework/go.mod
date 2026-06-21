module github.com/davejduke/obvious/services/control-framework

go 1.22

require (
	github.com/davejduke/obvious/shared/logging v0.0.0
	github.com/davejduke/obvious/shared/metrics v0.0.0
	github.com/davejduke/obvious/shared/types v0.0.0
	github.com/go-chi/chi/v5 v5.1.0
	github.com/google/uuid v1.6.0
	github.com/jackc/pgx/v5 v5.6.0
)

require (
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20221227161230-091c0ba34f0a // indirect
	github.com/jackc/puddle/v2 v2.2.1 // indirect
	golang.org/x/crypto v0.17.0 // indirect
	golang.org/x/sync v0.6.0 // indirect
	golang.org/x/text v0.14.0 // indirect
)

replace (
	github.com/davejduke/obvious/shared/logging => ../../shared/logging/go
	github.com/davejduke/obvious/shared/metrics => ../../shared/metrics/go
	github.com/davejduke/obvious/shared/types => ../../shared/types/go
)
