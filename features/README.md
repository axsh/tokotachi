# features

This directory contains **all feature implementations** in the project.

Each feature represents a **self-contained functional component** that may include
its own source code, dependencies, tests, and development environment.

Features are typically created using **feature templates** defined in the project
template catalog.

## Feature Structure

Each feature resides in its own directory:

```
features/
  feature-a/
  feature-b/
  feature-c/
```

A typical feature directory may include:

```
feature-name/
  README.md
  feature.yaml
  .devcontainer/
  src/
  tests/
```

Depending on the template used, a feature may contain:

* Go services
* Python workers
* Hybrid Go + Python implementations
* Supporting contracts or schemas

## Key Principles

Features should be:

* **Isolated** – minimal coupling with other features
* **Self-contained** – dependencies and configuration local to the feature
* **Template-based** – created from approved templates
* **Agent-friendly** – structured for automated development tools

## Creating a New Feature (Future Work)

New features should be generated using the project scaffolding tool.

Example:

```
featurectl new my-feature --template go-service
```

This will expand a verified template and create the feature inside this directory.

## Collaboration Model

Multiple developers or agents may work on different features simultaneously,
often using:

* Git worktrees
* Dev Containers
* Independent development environments

## Current Features

| Feature | Description | Language |
|---|---|---|
| `tt` | Development environment orchestrator CLI | Go |
| `integration-test` | Integration test suite for tt | Python |
