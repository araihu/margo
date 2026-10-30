# Publication metadata and final artifacts

Margo renders publication metadata. The host selects article covers, author
identities, public routes, feeds and deployment policy.

## Social images per page

Configured sites with an absolute HTTPS `site.base_url` can override
`site.social_image` in Markdown frontmatter:

```yaml
---
title: How the cluster works
social_image:
  path: ./assets/cover.png
  alt: A small cluster connected to shared storage
structured_data:
  type: BlogPosting
---
# How the cluster works
```

The path is relative to the Markdown file, optionally starting with `./`.
Use a normalized path below that document's directory. URLs, query strings,
fragments and parent-directory traversal are rejected. PNG, JPEG, GIF and WebP
are supported, up to 8 MiB per image and within the site's existing source-asset
byte budget. Dimensions and MIME type come from the image bytes.

Open Graph and Twitter use the same absolute, base-path-aware image URL and
alternative text. The image is published as a separate artifact even with
`assets: inline`, because social crawlers need an HTTP URL. Omit the field to
use the existing site-wide 1280x640 JPEG or PNG fallback. This setting does not
insert an image in the visible article or choose its JSON-LD image. Declare
`structured_data.image` separately when that image represents the article.
Native `Article` and `BlogPosting` declarations emit `og:type: article`, including
pages without author or date fields. Existing publication metadata keeps its
article projection. Head metadata also follows docs navigation and history.

## Visible dates

Set a Go time layout in `site.yaml`:

```yaml
publication:
  date_format: "02/01/2006"
```

`2026-10-28T23:30:00-03:00` then displays as `28/10/2026`. Formatting uses the
declared timezone, without converting to UTC or the build machine's timezone.
The `<time datetime>` attribute, article head metadata, JSON-LD, route reports
and manifest retain the original timestamp. Omitting the setting preserves the
existing visible timestamp. Go's named months and weekdays are English; use
numeric layouts when the publication needs another language. Consumer-supplied
publication headers retain control of their own date text.

## Author and tag links

Use `publication_links` in each page's frontmatter to keep author profiles and
archive destinations in the native header:

```yaml
authors: [Ana Silva, Rui Costa]
tags: [go]
publication_links:
  authors:
    Ana Silva: https://example.com/about/
  tags:
    go: https://example.com/tags/go/
```

Names must match declared authors or tags. Destinations must be absolute
HTTP(S) URLs, including any public base path. Each group accepts up to 64 links.
Margo escapes link attributes and labels. Omitted names remain plain text, so
Rui Costa receives no invented profile. The host owns route selection and must
declare matching JSON-LD author profile URLs separately when applicable. Links
work in configured and directory builds, alongside native date formatting.

## Browser and application identity

Add optional fields beside the existing `site.icon`:

```yaml
site:
  icons:
    - {path: assets/icon.svg, sizes: any}
    - {path: assets/icon-32.png, sizes: 32x32}
    - {path: assets/favicon.ico, rel: shortcut icon}
    - {path: assets/apple-touch-icon.png, rel: apple-touch-icon, sizes: 180x180}
  webmanifest: site.webmanifest
  application_name: My journal
  theme_color: '#007d9c'
```

This snippet extends an otherwise complete site configuration. Paths are relative
to `site.yaml`. Icon `rel` defaults to `icon`; supported values are `icon`,
`shortcut icon` and `apple-touch-icon`. Browser icons accept PNG, safe SVG and
ICO. Apple icons require PNG. Each icon is limited to 1 MiB. PNG sizes must
match decoded dimensions; SVG sizes must be `any` or omitted. At most 32 icons
can be declared. A declared `icon` with the same path as `site.icon` replaces
the default link; other declarations supplement it.

`webmanifest` must contain a JSON object of at most 64 KiB. Margo publishes its
bytes unchanged. The host supplies the manifest's referenced files and keeps
its URLs relative to the manifest's published location or sets an intentional
absolute URL. Margo does not rewrite `start_url`, `scope` or icon references.
The application name supplies both browser and Apple mobile title metadata.
Theme colors accept six-digit hexadecimal values. All declared files enter the
exact artifact manifest, and head URLs respect `base_path`.

## Finalize application-owned artifacts

Applications can add feeds or redirects to a `site.Result` without copying
Margo's digest or manifest JSON implementation:

```go
result.Artifacts = append(result.Artifacts, site.Artifact{
    Path: "index.xml", Content: feed,
})
result, err = site.Finalize(result)
if err != nil {
    return err
}
manifest, err := site.MarshalManifest(result, "")
if err != nil {
    return err
}
// Write result.Artifacts and manifest at site.ManifestPath in host-owned staging.
```

`Finalize` validates output paths, rejects duplicate, case-insensitive and
file/directory collisions, copies and sorts artifact bytes, and recomputes
`result.Manifest` through the root `margo.Manifest` contract. The reserved
`margo-manifest.json` path must not be included in `Artifacts`.

`MarshalManifest` also recomputes digests from current artifacts rather than
trusting a stale manifest. Its second argument is an optional trusted policy
identity. It emits the same versioned document used by `margo site` and
`margo serve`, retaining configuration and route metadata. Neither function
reparses HTML or updates route semantics. If the host changes canonical URLs,
structured data or route records, it must update that metadata too. Filesystem
staging, publication and deployment remain the host's responsibility.

## Page indexing

Site builds accept a boolean `noindex` frontmatter field. Use it for a generated error page or another utility page that must remain available without being advertised to search engines:

```yaml
---
title: Page not found
noindex: true
---
```

Margo emits `<meta name="robots" content="noindex, follow">` in the initial HTML head and omits the page from `sitemap.xml` and `llms.txt`. The artifact, route record, links and internal site navigation remain available. The route manifest records `noindex: true`. Omitting the field or setting it to `false` preserves normal indexing behavior. Non-boolean values fail with `site.indexing_invalid`.

This directive is not access control and does not change the HTTP status returned by the hosting platform. Configure the host to return HTTP 404 for missing URLs. Keep the error document outside the sitemap even when the host serves its explicit file URL with HTTP 200.
