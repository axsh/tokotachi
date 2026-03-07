# shared libraries

This directory contains **shared libraries** that may be used by multiple features.

These libraries provide reusable functionality that is not specific to any single feature.

Typical examples include:

* Utility libraries
* Shared domain logic
* Common helper functions
* Cross-feature tooling support

## Language Organization

Libraries may be organized by programming language:

```
shared/libs/
  go/
  python/
```

Each language directory may contain one or more reusable libraries.

Example:

```
shared/libs/go/logging
shared/libs/go/config
shared/libs/python/utils
```

## Design Guidelines

Shared libraries should:

* Be **stable and well-tested**
* Avoid feature-specific logic
* Provide clear interfaces
* Minimize dependencies on individual features

If a piece of code is only used by a single feature, it should remain within
that feature rather than being placed here.

## Versioning Considerations

Changes to shared libraries may affect multiple features.
Care should be taken when modifying existing APIs.
