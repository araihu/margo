# Goal da sessão: Margo v0.0.1

## Objetivo

Terminar a implementação verificável de Margo v0.0.1, preservar os contratos
do design aceito e entregar HTML/PDF humanos determinísticos. Este arquivo é o
estado operacional curto; o handoff completo está em
[`docs/HANDOFF.md`](docs/HANDOFF.md). O histórico expandido permanece no Git
(`git log -- GOAL.md`).

## Estado atual

- Status: `BLOCKED_BY_EXTERNAL_AUTHORITY`.
- Branch de implementação: `impl/v0.0.1-core`.
- Último checkpoint funcional: `b0ad36a7a6c706a419dbd4f9797fd3be3355ecbe`;
  tree `1463298b8ecd9f791b5280dc0723dfcdcaa249d9`.
- Commit de handoff: `7a9eb0dc661ac3c6d7d4ea74a34caa3755a8f12b`;
  tree `c6466f06a59d9937a6ae2c3c6c10eeb43ce0dffb`.
- Remoto: `origin/impl/v0.0.1-core` sincronizado.
- `main` local foi fast-forward para o commit de handoff; `origin/main` ainda
  está no bootstrap porque a proteção exige Pull Request e `Multi-module CI`.
- Worktree: `/private/tmp/margo-v001-implementation`.
- `test/browser/.cache/` é cache M0 intencional e não rastreado; nenhum outro
  arquivo deve ser incluído sem revisão.

## Autoridade aceita

- Design: `/private/tmp/gs-goshtoso-markdown-design/docs/GOSHTOSO_MARKDOWN_DESIGN.md`;
  head `bfcf296db63eb18b5e54d61ceb3156c193b98ecd`;
  SHA-256 `6b41bc995de83d6835a96fd9e73ddb59d642e87bd6ce13aaac3c0c7852499fc8`.
- Plano R17: `/private/tmp/margo-v001-plan-integration-r17`;
  head `b2bfb2bfd2140ae5472847c0c1e47a048cc1c528`;
  tree `1a1509f4f27b384616e005b704dcac9808961292`;
  manifest `5709b0322b7cba7203930f85aebb06d9c06ab6580b67cd60e427869565b7125c`;
  veredito independente: `acceptable`.

## Entregue localmente

- Root C0-C8 implementado: compilação, parsing, política, render plan,
  HTML semântico, tabelas, código, Mermaid, sinks e composição.
- HTML standalone light/dark usa CSS Goshtoso embutido, tema `modern`, TOC,
  marca, watermark e superfícies por modo.
- Benchmark exaustivo:
  `testdata/markdown/margo-full-feature-set.md`.
- HTML light:
  `output/html/margo-v0.0.1-optimistic.html`, 343588 bytes,
  SHA-256 `b830a04167d509368d36762bd9da181ce7213eb623f6f99bb51a2389f4b00a02`.
- HTML dark:
  `output/html/margo-v0.0.1-optimistic-dark.html`, 343591 bytes,
  SHA-256 `6d14c24b8008c720706d35c7bcfcf3ebcac714534a7a216d44ee33f4d2de5c27`.
- PDF light/dark checked: 20 páginas A4; evidência `margo/pdf-print/v2`;
  hashes atuais em `docs/HANDOFF.md`. Tabela Mermaid longa preserva as linhas
  finais em continuação; não há evidência de linha perdida no artefato atual.
- M0 local candidato: runner checked, Node `v26.5.0`, npm `11.17.0`, Chromium
  revision `1169` / `136.0.7103.25`, `49 passed`, `network=0`.
- CI corrigido no commit atual: `GOWORK=off`, `GOFLAGS=-mod=readonly`,
  `go mod verify`, hashes antes/depois de `go.mod`/`go.sum` e exclusão de
  pacotes somente de teste.

## Gates locais recentes

Passaram:

```text
GOWORK=off GOFLAGS=-mod=readonly go test ./... -count=1
GOWORK=off GOFLAGS=-mod=readonly go vet ./...
node --check test/browser/print-pdf.mjs
node --check test/browser/generate-evidence.mjs
node --test test/browser/generate-evidence.test.mjs
test/browser/run-playwright.sh --check --env-file test/browser/.cache/node-env.checked.sh
```

Reprodução equivalente do CI passou para root, `charts`, `pdf` e `cmd/margo`,
sem mutar metadados de módulo. `git diff --check` passa.

## Blockers reais

- I1a/I1b: proxy completo, receipt de fonte/ZIP e handoff externo ausentes.
- I2/I3: identidades verificadas para `charts`, `pdf` e `cmd/margo` ausentes.
- H1-H6, P1-P7, D1-D5 e O5: não iniciar legitimamente enquanto essas
  identidades não existirem; módulos opcionais continuam esqueletos.
- T6: `release/table-handoff.json` foi explicitamente movido pelo usuário para
  o fim do backlog; não criar arquivo, tag, release ou pin fictício.
- Aceitação independente formal de M0 ainda não foi emitida; resultado local é
  candidato, não aceite externo.

## Próximo passo autorizado

1. Integrar este checkpoint em `main` e publicar apenas o commit já revisado.
2. Obter receipts I1a/I1b/I2/I3 e aceitação M0 por autoridade externa.
3. Revalidar sob `GOWORK=off GOFLAGS=-mod=readonly` antes de iniciar módulos
   opcionais.
4. Retomar H/P/D/O5/T6 somente com identidades verificáveis e ownership
   explícito.

Não inventar `replace`, pseudo-versão, `go.sum`, proxy, handoff, tag, release
ou publicação para forçar GREEN.
