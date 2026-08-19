# Deck font bundle

These files are vendored for the Margo deck profile and exposed under the
profile aliases `Margo Sans`, `Margo Serif` and `Margo Mono`.

| Alias | Upstream family | Upstream release | License | File | SHA-256 |
| --- | --- | --- | --- | --- | --- |
| Margo Sans | Source Sans 3 | v19 | SIL OFL-1.1 | `margo-sans.woff2` | `7a19a7027e125257d310c6dbd78ae3a30b5ea1e3794d60b12bb28227a003bfda` |
| Margo Serif | Source Serif 4 | v14 | SIL OFL-1.1 | `margo-serif.woff2` | `c1df4596be5029233ed2afbb8b2f6ea20784b3fb1aa5d6b5c6519ccd85eb3dfb` |
| Margo Mono | Source Code Pro | v31 | SIL OFL-1.1 | `margo-mono.woff2` | `8b774aaa5137a38ef40f4ac9d36db9a5eee152b2f66589dfdc82ff007fc87135` |

Upstream CSS manifests:

- <https://fonts.googleapis.com/css2?family=Source+Sans+3:wght@400;600;700;800&display=swap>
- <https://fonts.googleapis.com/css2?family=Source+Serif+4:wght@400;600;700&display=swap>
- <https://fonts.googleapis.com/css2?family=Source+Code+Pro:wght@400;600&display=swap>

The aliases preserve the frozen Margo theme contract while keeping the binary
bundle self-contained. Each variable WOFF2 is registered for the exact theme
weights required by the profile. Full license text: <https://scripts.sil.org/OFL>.
