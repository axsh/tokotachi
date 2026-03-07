# environments

This directory contains **environment configuration files** used during development and testing.

These configurations define supporting services and infrastructure components that
may be shared across multiple features in the repository.

Typical examples include:

* Databases
* Message brokers
* Caches
* Observability tools
* Local service orchestration

The goal of this directory is to provide **reusable development infrastructure**
without tying it to a specific feature.

## Structure

```
environments/
  compose/
  ...
```

### compose/

Contains Docker Compose configurations used to launch shared development services.

## Design Principles

Environment definitions here should be:

* **Reusable across features**
* **Independent of specific feature implementations**
* **Suitable for local development**

Feature-specific environment setup should remain inside the corresponding
feature directory if it cannot be shared.

## Usage

Developers or automation tools can start shared services using Docker Compose
definitions located in this directory.

Example:

```
docker compose -f environments/compose/common.yml up
```
