# API v1 Conventions

## Base Path

Endpoint baru menggunakan `/api/v1`. Endpoint legacy `/api` dan `/admin_api`
tetap tersedia selama migration window.

## Request ID

- Client boleh mengirim UUID melalui header `X-Request-ID`.
- Server mempertahankan UUID yang valid atau membuat UUID baru.
- Response selalu mengembalikan `X-Request-ID`.
- Envelope v1 menyertakan `meta.request_id`.
- Log backend harus memakai request ID yang sama.

## Success

```json
{
  "data": {},
  "meta": {
    "request_id": "00000000-0000-4000-8000-000000000000"
  }
}
```

Gunakan status HTTP yang sesuai: `200`, `201`, atau `204`.

## Error

```json
{
  "error": {
    "code": "COMMUNITY_ACCESS_DENIED",
    "message": "Community access denied",
    "fields": {
      "field_name": "reason"
    }
  },
  "meta": {
    "request_id": "00000000-0000-4000-8000-000000000000"
  }
}
```

- `code` stabil dan dipakai frontend/analytics.
- `message` aman ditampilkan atau diterjemahkan.
- `fields` opsional untuk validation error.
- Internal error, SQL, credential, stack trace, dan provider payload tidak
  dikembalikan ke client.

## Authentication dan Tenant

- Bearer token user berbeda dari ticket-limited token.
- Portal community mengambil `community_id` dari path.
- Server selalu memverifikasi membership dan permission.
- `community_id` dari body tidak dapat mengganti tenant path.
- Platform admin bypass tetap masuk audit log pada implementasi production.

## Pagination dan Time

- Daftar besar akan memakai cursor pagination.
- Timestamp API memakai RFC 3339 UTC.
- Event menyimpan timezone IANA secara terpisah untuk rendering.

## Idempotency

Operasi commerce dan booking kritis menggunakan header `Idempotency-Key`.
Create community/invitation belum memakai idempotency key karena belum termasuk
commerce path, tetapi unique constraint tetap wajib.

Kontrak awal tersedia di [`../api/openapi.yaml`](../api/openapi.yaml).
