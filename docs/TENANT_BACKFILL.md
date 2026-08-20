# Legacy Event Tenant Backfill

Legacy events are linked to communities through
`event_community_assignments`. Existing assignments are never overwritten by
the backfill command.

## Preconditions

1. Back up the target database and record its checksum.
2. Apply and verify versioned migrations.
3. Choose an existing user who is authorized to own the legacy tenant.
4. Review the event and assignment counts from the read-only dry run.

## Dry run

```bash
go run ./cmd/backfill-event-tenants \
  -owner-email owner@example.com \
  -community-name "Legacy Usloop" \
  -community-slug legacy-usloop \
  -community-type dakwah
```

The default mode performs reads only. It reports total, assigned, and
unassigned events and whether a community would be created.

## Apply

Run the identical command with `-apply` only after the dry-run counts have been
approved:

```bash
go run ./cmd/backfill-event-tenants \
  -owner-email owner@example.com \
  -community-name "Legacy Usloop" \
  -community-slug legacy-usloop \
  -community-type dakwah \
  -apply
```

The apply operation is transactional and idempotent. It creates the community
and owner membership when absent, then assigns only events which do not already
have a tenant. A second apply must report zero new assignments.

Afterward, manually review the generated `legacy_backfill` assignments before
tenant-scoped event APIs treat them as authoritative.
