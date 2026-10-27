# Structured data in sites

A site can declare JSON-LD in the consumer-owned `structured_data` frontmatter
field. Margo serializes it into one `application/ld+json` script in the page head.
This works in directory builds and configured layouts, without unsafe HTML.
The generic document renderer does not interpret this field.

```yaml
---
title: About the author
structured_data:
  '@context': https://schema.org
  '@type': AboutPage
  '@id': https://example.com/about/#webpage
  url: https://example.com/about/
  name: About the author
  mainEntity:
    '@type': Person
    name: Ana Silva
    url: https://example.com/about/
---
# About the author

Ana Silva writes about software.
```

## Articles from page metadata

Configured sites can opt into the native article projection instead of supplying
JSON-LD fields twice:

```yaml
---
title: How the cluster works
description: A tour of the cluster.
authors: [Ana Silva, Rui Costa]
publishedAt: "2026-10-27T10:00:00-03:00"
structured_data:
  type: BlogPosting
  image: https://example.com/posts/cluster/assets/cover.jpg
  author_urls:
    Ana Silva: https://example.com/about/
---
# How the cluster works
```

`type` accepts `Article` or `BlogPosting`. Only `image` and `author_urls` are
additional options. Images and profile URLs must be absolute HTTP(S) URLs;
profiles must name an author declared on the page. Unknown options fail.
The canonical route supplies `url`, `mainEntityOfPage` and `@id`. The ID suffix
is `#article` or `#blogposting`. The normalized title, description, language,
authors, publication date and modification date supply the corresponding
Schema.org properties. Dates retain their timezone; modification cannot precede
publication. Missing optional values are omitted, including images and author
profiles. The site's social-image fallback is not used as an article image.

This projection requires a configured canonical URL. Directory-only builds can
use the explicit JSON-LD form with their host's public identity instead.
Nothing opts a page into article markup implicitly: author metadata alone does
not turn a home page or archive into an article.

## Explicit JSON-LD

Use `@graph` for multiple entities in one declaration, such as `WebSite` and
`WebPage` on the home page. The root must be an object with an absolute HTTP(S)
`@context` string and either a non-empty `@type` or a non-empty `@graph` of typed
objects. A type can be a string or a non-empty array of strings. Serialized data
is limited to 64 KiB. Inline context definitions, root arrays and untyped graph
nodes are not supported by this contract.

For explicit JSON-LD, Margo does not fetch contexts, infer types from routes,
invent author identities, or validate a vocabulary's properties. The host owns those semantics, including
canonical URLs, locale and base-path handling. Keep public URLs absolute and
match names, dates, images and list entries to visible content. A blog can use
`BlogPosting` for articles, `CollectionPage` for archives, `AboutPage` for its
about page and `ContactPage` for contact details. Exclude unpublished content.

Omitting the field preserves existing output. Invalid declarations fail with
`site.structured_data_invalid` during preparation. A declaration combined with
an existing JSON-LD script fails with `site.structured_data_duplicate`; consolidate
entities into `@graph` instead of emitting another script. JSON strings escape
HTML-sensitive characters, so a title containing `</script>` remains data.
Route reports and the site manifest retain the serialized declaration as
`structuredData`. This metadata grants no scripting or network capability.

This is a serialization contract, not a promise of Google rich results. Check
[Article guidance](https://developers.google.com/search/docs/appearance/structured-data/article),
[site names](https://developers.google.com/search/docs/appearance/site-names)
and the relevant [Schema.org types](https://schema.org/docs/schemas.html), then
validate the final HTML for the host's content model.
