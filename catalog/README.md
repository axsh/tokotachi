# catalog

This directory contains the **template catalog configuration** used by this repository.

The catalog defines which feature templates are available and how they are resolved.
Templates themselves are typically stored in external repositories and are referenced
here by logical name.

This approach ensures that:

* Only **verified and approved templates** are used in this project
* Template versions are **pinned for reproducibility**
* The project can update template sources in a **controlled manner**

## Typical Contents

Examples of files stored here:

* `template-catalog.lock.yaml`
  Defines which external template catalog repository and version are used.

* `allowed-templates.yaml`
  Lists the logical template names that are allowed to be used in this project.

* `project-defaults.yaml`
  Default parameters applied when generating new features from templates.

## How It Is Used

When a new feature is created using the project scaffolding tool (for example `featurectl`):

1. The catalog is read.
2. The logical template name is resolved.
3. The template source repository and version are determined.
4. The template is downloaded and expanded into the `features/` directory.

## Important Notes

* Do **not modify template versions casually**, as doing so may affect reproducibility.
* Template updates should be reviewed and coordinated with the team.
* Templates themselves are maintained in a **separate template repository**.
