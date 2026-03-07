# compose

This directory contains **Docker Compose configurations** used for local development environments.

These files define shared infrastructure services that multiple features may rely on.

Typical services include:

* PostgreSQL
* Redis
* Message queues
* Local object storage
* Observability components

## Purpose

The goal is to provide a **consistent local environment** for development and testing.

Instead of each feature defining its own infrastructure stack, common services
can be defined here and reused across the repository.

## Examples

Typical files might include:

```
common.yml
postgres.yml
redis.yml
observability.yml
```

Each file should define a **focused and composable environment component**.

For example:

* `postgres.yml` → PostgreSQL database
* `redis.yml` → Redis instance
* `common.yml` → commonly required services

These files can be combined when starting environments.

Example:

```
docker compose \
  -f environments/compose/common.yml \
  -f environments/compose/postgres.yml \
  up
```

## Guidelines

* Keep each compose file **small and modular**
* Avoid feature-specific configuration here
* Prefer reusable services shared by multiple features
