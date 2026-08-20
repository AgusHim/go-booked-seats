# Public Event Discovery

Public discovery uses `/api/v1/events`. The legacy `/api/events` contract
remains temporarily for existing booking/admin screens and must not be used for
new public surfaces.

An event is discoverable only when:

1. `events.status = published`;
2. it has exactly one reviewed or backfilled `event_community_assignments`
   record;
3. the assigned community is active.

Draft, unassigned, and inactive-tenant events are excluded at repository level.
The public repository never creates or updates an event during a read.

Supported list filters:

- `q`: case-insensitive name, description, or location search;
- `community`: exact community slug;
- `location`: partial location match;
- `date_from` and `date_to`: `YYYY-MM-DD`;
- `page` and `limit`, with a maximum limit of 50.

`GET /api/v1/me/events` returns published events from communities followed by
the authenticated user. Following affects personalization only and never grants
organizer permissions.
