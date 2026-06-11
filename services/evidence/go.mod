module github.com/davejduke/obvious/services/evidence

go 1.22

require (
	github.com/davejduke/obvious/shared/types v0.0.0
	github.com/gin-gonic/gin v1.10.0
	github.com/google/uuid v1.6.0
	github.com/jackc/pgx/v5 v5.6.0
	github.com/redis/go-redis/v9 v9.5.1
	go.uber.org/zap v1.27.0
)

replace github.com/davejduke/obvious/shared/types => ../../shared/types/go
