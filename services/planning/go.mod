module github.com/davejduke/obvious/services/planning

go 1.22

require (
	github.com/davejduke/obvious/shared/logging v0.0.0
	github.com/davejduke/obvious/shared/metrics v0.0.0
	github.com/go-chi/chi/v5 v5.1.0
	github.com/google/uuid v1.6.0
)

require (
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/prometheus/client_golang v1.19.1 // indirect
	github.com/prometheus/client_model v0.6.1 // indirect
	github.com/prometheus/common v0.53.0 // indirect
	github.com/prometheus/procfs v0.14.0 // indirect
	golang.org/x/sys v0.20.0 // indirect
	google.golang.org/protobuf v1.34.1 // indirect
)

replace (
	github.com/davejduke/obvious/shared/logging => ../../shared/logging/go
	github.com/davejduke/obvious/shared/metrics => ../../shared/metrics/go
	github.com/davejduke/obvious/shared/types => ../../shared/types/go
)
