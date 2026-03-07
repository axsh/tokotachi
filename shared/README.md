# shared

This directory contains **shared resources used across multiple features**.

These resources are intended for reuse and should represent stable,
well-defined components that are beneficial to multiple parts of the project.

Examples of shared resources include:

* API schemas
* Data contracts
* Test data
* Shared utilities
* Common libraries

## Structure

```
shared/
  libs/
  schemas/
  contracts/
  testdata/
```

### Important Guidelines

Only place resources here if they meet at least one of the following criteria:

* Used by **multiple features**
* Represents a **stable interface or contract**
* Provides **general utilities** applicable across the project

Avoid placing feature-specific code here.

If a component is only used by one feature, it should remain within that feature's directory.

## Stability

Items placed in this directory should be treated as **shared dependencies**.
Changes may affect multiple features and should be reviewed carefully.
