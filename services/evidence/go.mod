module github.com/davejduke/obvious/services/evidence

go 1.22

require (
	github.com/davejduke/obvious/shared/logging v0.0.0
	github.com/davejduke/obvious/shared/metrics v0.0.0
	github.com/go-chi/chi/v5 v5.1.0
	github.com/google/uuid v1.6.0
)

replace (
	github.com/davejduke/obvious/shared/logging => ../../shared/logging/go
	github.com/davejduke/obvious/shared/metrics => ../../shared/metrics/go
	github.com/davejduke/obvious/shared/types => ../../shared/types/go
)
