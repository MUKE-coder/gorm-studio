# Changelog

All notable changes to GORM Studio are documented here. The format is based on
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and the project aims
to follow [Semantic Versioning](https://semver.org/spec/v2.0.0.html) (see
[STABILITY.md](STABILITY.md)).

## [Unreleased]

This release is a security-hardening pass. All changes are backward compatible
at the `studio.Mount` / `studio.Config` API level; the behavioral changes below
close security gaps and may reject inputs that previously (incorrectly)
succeeded.

### Security

- **SQL editor read-only enforcement.** Read-only mode is now enforced on the
  SQL editor's read path as well. A write disguised as a read — a stacked
  `SELECT 1; DELETE ...` or `PRAGMA x = y` — is rejected instead of executed.
- **DDL blocklist hardening.** The blocklist now strips comments and rejects
  multi-statement input before matching, so `/* */ DROP`, leading whitespace,
  case changes, and stacked statements no longer evade it. `VACUUM` and
  `REINDEX` are blocked (`VACUUM INTO` could write an arbitrary file).
- **SQL data import is fail-closed.** Statement splitting is quote-aware, every
  statement must be an `INSERT`, and the batch runs in a transaction — a
  non-INSERT statement aborts the import with nothing applied.
- **Composite primary keys.** Row get/update/delete require all key columns; a
  partial id no longer matches (and deletes/updates) many rows. Bulk delete
  refuses composite-key tables.
- **Column-type validation.** Imported column types are validated before being
  written into `CREATE TABLE`, closing a DDL-injection vector.
- **Credentials.** The login token is kept in memory only (never in browser
  storage), so a same-origin XSS cannot read it from storage.
- **Security headers.** `Content-Security-Policy` (HTML page),
  `X-Content-Type-Options`, `X-Frame-Options`, and `Referrer-Policy` are set.

### Added

- **`Config.Scope`** — a per-query hook to enforce row-level constraints
  (e.g. multi-tenancy) that Studio would otherwise bypass.
- **`Config.TablePolicy`** — `Hidden` (omitted everywhere) and `ReadOnly`
  (browsable, not mutable) tables.
- **`Config.AuditLogger` / `AuditEvent`** — records every mutation with the
  acting user; `DefaultAuditLogger` logs to stdout.
- **`Config.MaxImportBytes` / `Config.MaxImportRows`** — import size and row
  limits (defaults 32 MiB / 100k). Excel import streams rows to bound memory.
- **`Config.RateLimit`** — per-client-IP rate limiting on the SQL and import
  endpoints.
- **Import dry run** — `?dry_run=true` previews a schema/data/models import
  without applying it.
- **Streaming exports** — full-database exports page through rows instead of
  loading whole tables into memory.
- **Fuzz tests** for the SQL/DBML/Go-struct/statement parsers, plus a CI
  fuzz-smoke job.

## [1.0.0]

- Initial public release: schema introspection, data grid with CRUD/filtering,
  relationship navigation, raw SQL editor, bulk operations, export/import in
  multiple formats, ERD rendering, and Go struct code generation behind a
  single `studio.Mount()` call.
