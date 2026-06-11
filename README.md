# AIAUDITOR

Autonomous AI-powered cybersecurity audit reasoning engine applying IIA standards to automate NIS 2 compliance auditing.

## Architecture

- **API Layer:** Go 1.26 microservices
- **Reasoning Engine:** Python 3.14 (deterministic algorithms — no LLM for audit conclusions)
- **Frontend:** React 19 + Next.js 15
- **Database:** PostgreSQL 18 (OLTP)
- **Messaging:** Kafka (Redpanda)
- **Cache:** Redis 7.2

## Project Structure

```
services/          # Go microservices
engine/            # Python reasoning engine
frontend/          # Next.js frontend
shared/            # Shared types and contracts
infra/             # Docker Compose and infrastructure
scripts/           # Build and utility scripts
```

## Quick Start

```bash
make up        # Start all containers
make migrate   # Run database migrations
make seed      # Seed NIS 2 control data
make test      # Run all tests
```
