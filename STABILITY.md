# Stability & Versioning

GORM Studio follows [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## What "stable" covers

The following are the public API and are covered by the compatibility promise —
breaking changes to them require a major version bump:

- The `studio.Mount(router, db, models, ...Config)` function signature.
- The exported fields of `studio.Config`, `studio.TablePolicy`, and
  `studio.RateLimitConfig`. New fields may be **added** in minor releases;
  existing fields will not change type or meaning without a major bump.
- The `studio.AuditEvent` struct and `studio.DefaultAuditLogger`.
- The HTTP API surface under the configured prefix (routes, request shapes, and
  success response shapes documented in `docs/api-reference.md`).

## What is not covered

- The embedded frontend HTML/JS (implementation detail; may change any time).
- Exact wording of error messages and log lines.
- Internal helpers and any identifier not exported from the `studio` package.
- The precise formatting of exported SQL/CSV/JSON beyond being valid and
  round-trippable.

## Security fixes

Security fixes may tighten input validation in a minor or patch release even if
that rejects inputs which previously (incorrectly) succeeded. Such changes are
called out under **Security** in [CHANGELOG.md](CHANGELOG.md).

## Supported Go and databases

- Go: the versions exercised in CI (see `.github/workflows/ci.yml`).
- Databases: SQLite, PostgreSQL, and MySQL. Cross-database behavior is exercised
  by the integration test suite (`-tags integration`) in CI.

## Reporting issues

Please report security issues privately to the maintainer rather than opening a
public issue.
