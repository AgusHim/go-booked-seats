# Public Metadata, Prerender, and Sharing

The responsive web app provides:

- Indonesian base HTML metadata;
- per-community and per-event document title and description;
- runtime OpenGraph and Twitter card tags;
- canonical URLs without hash fragments;
- Web Share API with clipboard fallback.

The production build can additionally produce crawler-readable static pages:

```sh
SEO_API_BASE_URL=https://api.usloop.id \
PUBLIC_SITE_URL=https://usloop.id \
npm run build:seo
```

The command builds the Vite application, reads every active community from
`GET /api/v1/communities` and every published event from
`GET /api/v1/events`, then writes:

- `dist/communities/{slug}/index.html`;
- `dist/events/{slug}/index.html`;
- `dist/sitemap.xml`;
- `dist/robots.txt`;
- `dist/prerender-manifest.json`.

Each static detail page contains canonical, OpenGraph, Twitter, semantic HTML,
and JSON-LD (`Organization` or `Event`) before JavaScript executes. React then
replaces the small prerendered body with the interactive application.

The static host must resolve the generated `index.html` before applying its SPA
fallback. Run `build:seo` against the production-read replica/API after
publishing or unpublishing content so removed routes do not remain in a previous
artifact.

Local tests verify escaping, crawler-visible content, structured data, sitemap,
and robots policies. Production release verification must still inspect the
deployed URL with search and social-card debugging tools because those tools
cannot validate an unpublished local artifact.
