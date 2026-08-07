# Goal da sessão: Margo v0.0.1

## Objetivo final

Terminar a execução do plano de implementação do Margo v0.0.1 e seguir com a
implementação verificável, preservando os contratos do design aceito. Esta
sessão deve manter um registro vivo para sobreviver a compactação de contexto:
cada marco importante, decisão, falha, identidade Git, teste e bloqueio deve
ser registrado aqui.

O objetivo só pode ser marcado como concluído quando os gates do plano forem
executados e houver evidência independente de implementação, testes, escopo,
identidades e estado limpo. Até lá, este arquivo permanece `IN_PROGRESS`.

## Estado atual

- Status: `IN_PROGRESS`.
- Plano aceito: revisão R17, veredito `acceptable`.
- Design aceito: commit `bfcf296db63eb18b5e54d61ceb3156c193b98ecd`, SHA-256
  `6b41bc995de83d6835a96fd9e73ddb59d642e87bd6ce13aaac3c0c7852499fc8`.
- Plano aceito: manifest SHA-256
  `5709b0322b7cba7203930f85aebb06d9c06ab6580b67cd60e427869565b7125c`.
- Snapshot aceito preservado: `/private/tmp/margo-v001-plan-integration-r17`,
  branch `docs/v0.0.1-implementation-plan`, HEAD
  `b2bfb2bfd2140ae5472847c0c1e47a048cc1c528`, tree
  `1a1509f4f27b384616e005b704dcac9808961292`.
- Worktree de implementação desta sessão:
  `/private/tmp/margo-v001-implementation`, branch `impl/v0.0.1-core`.
- Base desta implementação: o HEAD R17 aceito acima; o snapshot aceito não é
  editado.
- Repositório: `https://github.com/araihu/margo`.
- HEAD atual da implementação: `11d3d193b40d6810a77edf88f89d50a5eeffc43f`,
  tree `015d2dd2da617e4b1c344c914498a2fe316a06e5`.

## Ordem de execução vinculante

1. C0: provisionar dependências, ferramentas e o contrato
   `rootModuleToC5.v1`; C0 é o único escritor de `go.mod`, `go.sum` e
   `tools/toolchain.lock`.
2. T0: preparar a ferramenta do Table no worktree próprio do Goshtoso; T1-T6
   seguem serialmente a aceitação de cada predecessor.
3. C1-C4: congelar API, parsing, política e plano de renderização do root.
4. C5-C7: finalizar a fonte root e a decisão de head owner antes de I1a.
5. I1a/I1b: criar e verificar o proxy completo, a normalização e as provas de
   origem; consumidores nunca alteram o handoff.
6. M0-M7, H1-H6, P1-P7, D1-D5 e O1-O7: executar somente após os predecessores
   aceitos, com os gates readonly, browser, acessibilidade, PDF, CLI e social.
7. I2-I4: integrar, produzir evidência de release candidate e parar antes de
   tag/publicação/deploy sem autoridade explícita.

## Referências de contexto e especificação

- Design aceito: `/private/tmp/gs-goshtoso-markdown-design/docs/GOSHTOSO_MARKDOWN_DESIGN.md`.
- Planos aceitos no worktree: `/private/tmp/margo-v001-implementation/docs/superpowers/plans/`.
- Planos fonte: `/Users/guilhermecastro/Documents/Codex/2026-08-06/margo-v001-implementation-plan/docs/superpowers/plans/`.
- Roadmap: `docs/superpowers/plans/2026-08-06-margo-v0.0.1-roadmap.md`.
- Core/API: `docs/superpowers/plans/2026-08-06-margo-v0.0.1-repository-core.md`.
- Table: `docs/superpowers/plans/2026-08-06-margo-v0.0.1-goshtoso-table.md`.
- Runtime/browser: `docs/superpowers/plans/2026-08-06-margo-v0.0.1-mermaid-runtime.md`.
- Charts: `docs/superpowers/plans/2026-08-06-margo-v0.0.1-charts.md`.
- PDF: `docs/superpowers/plans/2026-08-06-margo-v0.0.1-pdf-platform.md`.
- Decks: `docs/superpowers/plans/2026-08-06-margo-v0.0.1-decks-marpit.md`.
- CLI: `docs/superpowers/plans/2026-08-06-margo-v0.0.1-cli-output.md`.
- Traceability: `docs/superpowers/plans/2026-08-06-margo-v0.0.1-traceability.md`.
- Goshtoso predecessor repository/worktrees: `/Users/guilhermecastro/repos/araihu/goshtoso` and `/private/tmp/gs-*`.
- Accepted/rejected review snapshots: `/private/tmp/margo-v001-plan-integration-r17` and the preserved `r4`-`r16` worktrees/refs in the Margo repository.
- Control-plane ledgers: `/Users/guilhermecastro/.codex/state/orchestrating-control-planes/019fd537-7d93-7982-bbb4-467aa50e3a9b.yaml`, its `.branches.yaml`, `.worktrees.yaml`, and `registry.lock`.

## Progresso

### 2026-08-06

- R17 foi aceito por revisão independente; o plano contém 49 tarefas, 34/34
  seções normativas, 22/22 critérios de aceitação e SG01/AUTH01/HEAD01.
- O branch `main` do Margo está limpo no bootstrap `608c0f4`.
- Worktree novo criado em `/private/tmp/margo-v001-implementation` a partir do
  R17 aceito; snapshots R17 e anteriores permanecem preservados.
- `2026-08-06T22:15:05-03:00`: C0 RED executado no worktree novo com os quatro
  comandos do plano. `go tool templ version` e `go tool muamba --version`
  falharam com `no such tool`; as consultas `go list -m -json` para Goshtoso
  v0.1.2 e x/mod v0.30.1-0.20251115032019-269c237cf350 retornaram zero, sem
  escrever módulo. O RED confirma que não há fallback para executável ambient.
- `2026-08-06T22:25`: C0 GREEN/REFACTOR passaram: `go mod verify`, `go tool
  templ version`, `go tool muamba --version`, `go list -deps ./...` e a prova
  independente do x/mod (sums, `.info`/`.mod`/`.zip` e source manifest).
- C0 commit local: `19df7d9dcb17eadea9bd01b144fbb6c70252b312`, tree
  `ffad0a7760a0827694adf926c1e3093a2b63f46f`; `go.mod` SHA-256
  `0eb36e99f0c59989a8c8772899acafa7b30dd205c241801b2d1c52ad775617fe`,
  `go.sum` SHA-256
  `1c7ae9b89ad246a943998c8e7a4a4f19bd59a53f84409e0d930ebe9b1670ddbb`.
- C1 RED confirmou `undefined: New`/`undefined: Marshal`; C1 GREEN/REFACTOR
  passaram testes unitários, canonical JSON, snapshot defensivo e race.
- C1 commit local: `00bc4be4fac5c624cadeed37333f722443478020`, tree
  `564e407e87cc5aeaa2e43f6f92f35073e2710c9a`; `go test ./... -count=1`
  também passou.
- Commit de integração do hook de normalização C1 -> C2:
  `9bdcb81` (somente `compiler.go`/`document.go`), mantendo os módulos
  congelados.
- C2 RED confirmou que parser, diagnósticos e canonicalização ainda estavam
  ausentes. C2 GREEN/REFACTOR passaram frontmatter estrito (inclusive
  `goshtoso.unknown_field` com line/column/pointer), YAML limits, perfil
  Goldmark GFM/footnotes/linkify/task list e IDs determinísticos de headings.
- C2 commit local: `b6a4b22ad82a4e5da756de5b9cda22731bcf0299`, tree
  `3b54c9baa174431f958af8f0b0d7b753340c971f`; `go test ./... -count=1`
  passou. `go.mod`, `go.sum` e `tools/toolchain.lock` não foram alterados.
- C3 RED confirmou os símbolos de política, allowlist HTML, limites de recurso
  e validação de tokens ausentes. GREEN/REFACTOR passaram o comando focado do
  plano e o corpus de bypass 20 vezes:
  `GOWORK=off GOFLAGS=-mod=readonly go test . ./internal/htmlpolicy -run
  'Test(Policy|HTML|URL|Token|Resource|YAML)' -count=1` e
  `GOWORK=off GOFLAGS=-mod=readonly go test ./internal/htmlpolicy -run
  TestSanitizerBypassCorpus -count=20`.
- C3 commit local: `3c819da` (`feat: enforce fail-closed document policy`),
  seguido do commit de integração `0ff4d94` (`refactor: freeze effective
  policy during compile`). A política usa `RawHTMLDeny`/`RawHTMLSanitized`,
  `OutputBytes` positivo limitado a `64<<20`, rejeita raw HTML não declarado,
  mismatch de host e HTML/URL/CSS inseguros; `Compile` armazena o valor
  efetivo no `Document`. `GOWORK=off GOFLAGS=-mod=readonly go test ./...
  -count=1` passou. HEAD atual é `0ff4d943ef5d55e60e8fe23d0cd532738d6c9a07`,
  tree `44899657863b92d4856d41740c7c91f7b13198ce`.
- O teste C2 de preservação de raw HTML foi atualizado para declarar
  explicitamente `rawHTML: sanitized` e executar sob um host que permite essa
  capacidade; isso mantém a prova de AST sem violar o gate fail-closed C3.
- C4 RED confirmou que a API de extensões, o `RenderContext`, o registro
  imutável e o plano de renderização ainda não existiam.
- C4 GREEN/REFACTOR passaram o foco do plano com
  `GOWORK=off GOFLAGS=-mod=readonly go test . -run
  'Test(Extension|Registry|DivergentCompiler|CompilerPolicyMutation|EquivalentCompiler|MissingChart)' -count=1`
  e o gate de concorrência com
  `GOWORK=off GOFLAGS=-mod=readonly go test -race . -run
  'TestConcurrent.*Extension|TestConcurrentRenderPlan' -count=20`.
- C4 criou `ExtensionRegistration`, `ExtensionFactory`, `RenderContext`,
  sessões por render, fences reconhecidos, falha fechada para
  `goshtosochart` sem integração e execução em buffer privado antes da saída.
  Durante a prova foi corrigido o clone do plano para preservar também os nós
  (antes eles eram zerados antes da cópia).
- C4 commits locais: `30a168a` (`feat: freeze extension render plans`) e
  `e227eb4` (`refactor: connect compiler to frozen render plans`), HEAD/tree
  registrados acima. O artefato de prova HTML foi gerado em
  `/tmp/margo-contract-preview/margo-contract-preview.html`; ele é um preview
  do contrato C4 de extensão, não a saída semântica final do C5.
- Próximo marco: C5 (renderer semântico, adapters de tabela/código e HTML),
  mantendo o handoff C0 read-only.
- C5 RED confirmou a ausência de HTML semântico, adapters Goshtoso e golden
  output. GREEN/REFACTOR passaram a geração templ determinística, o foco
  semântico/tabela/código, a suíte completa do root e o race focado. O renderer
  agora produz headings, listas, quotes, links, imagens, tabelas client-only e
  CodeBlock/Chroma; rejeita sorting server-side e mantém raw HTML sob a política
  sanitizada.
- C5 commit local: `a6b09ac7617c649e5f03389752a1bafb5722fc6`, tree
  `b2457e8da0f95d6d86d8da720cbd223602b8a50d`. Golden HTML:
  `testdata/render/semantic.html`. Amostra gerada do renderer:
  `/tmp/margo-semantic-preview/margo-semantic-preview.html`.
- C6 RED confirmou a ausência de assets, tokens, marca e shell standalone com
  `GOWORK=off GOFLAGS=-mod=readonly go test . -run
  'Test(Offline|LibraryCSS|HeadOwnerSelection|Theme|Brand|AssetOverride|Standalone)' -count=1`.
- C6 GREEN/REFACTOR passaram geração templ determinística, foco C6, suíte
  completa do root e race focado. O shell agora embute `assets/document.css`,
  expõe tokens validados, rejeita override inválido sem fallback, aceita
  componentes de marca confiáveis e produz HTML offline determinístico com
  `ri-00000000`, fingerprint de conteúdo e escopo `.goshtoso-document`.
  `HeadOwnerSelection` foi congelado como `margo/socialMetadataTags` porque o
  Goshtoso `v0.1.2` inspecionado não expõe `head.Metadata`; o caminho e digest
  da fonte de API são registrados em `standalone.go` para verificação pelo
  handoff T6/I1a.
- C6 executou a prova de hashes C0 antes/depois da geração e testes: `go.mod`
  permaneceu `0eb36e99f0c59989a8c8772899acafa7b30dd205c241801b2d1c52ad775617fe`
  e `go.sum` permaneceu
  `1c7ae9b89ad246a943998c8e7a4a4f19bd59a53f84409e0d930ebe9b1670ddbb`.
- C6 commit local: `d41fe882a0e944efd4a8cd3ab66e3c8e59b0c222`, tree
  `48aa14bea58be57be213adde8f9baf0daa758677`. O checkpoint staged apenas os
  12 caminhos C6 efetivamente existentes e passou `git diff --cached --check`,
  manifestos name-status/summary/raw e filtro de paths proibidos.
- C7 RED confirmou a ausência de autoridade, metadata social e verificador de
  preview. GREEN/REFACTOR passaram geração templ, suíte completa, race focado e
  a prova de PNG `1280x640`, 28.740 bytes, SHA-256
  `9d570d7851a54e2024da10b3e48cbdd19f544c12f06ca4e372b426f0609b2974`.
  O modo público emite exatamente um conjunto inicial de tags; o modo privado
  rejeita/omite URLs sociais. `AuthorityRecord` usa JSON fechado, digest
  canônico sem campo `recordDigest`, origem HTTPS e transporte sem redirects.
- C7 preserva a seleção C6 `margo/socialMetadataTags` sem editar `standalone.go`
  ou qualquer módulo root. Hashes C0 continuaram estáveis antes/depois de
  templ/testes. C7 commit local: `11d3d193b40d6810a77edf88f89d50a5eeffc43f`,
  tree `015d2dd2da617e4b1c344c914498a2fe316a06e5`; checkpoint com 14 paths
  exatos passou os manifestos staged e `git diff --cached --check`.

## Decisões e limites

- Não editar o worktree R17 aceito nem qualquer snapshot rejeitado.
- Não fazer push, PR, merge, tag, release, publicação, deploy ou limpeza de
  worktrees sem autorização explícita.
- Manter um escritor por caminho; C0 é o único escritor dos módulos root.
- Toda tarefa usa o RED/GREEN/REFACTOR do plano, o gate de paths e o checkpoint
  de commit correspondente.
- Se uma dependência, API ou gate estiver ausente, registrar o bloqueio e parar
  nessa fronteira; não inventar uma substituição silenciosa.

## Como atualizar este arquivo

Após cada evento importante, atualizar `Estado atual`, `Progresso`,
`Decisões e limites` ou `Bloqueios`, incluindo data/hora, comando ou teste,
resultado, paths afetados e HEAD/tree quando houver commit. Em dúvida sobre o
contexto, consultar primeiro este arquivo e depois os planos referenciados.

## Bloqueios e dúvidas

- Nenhum bloqueio confirmado neste momento.
- Atenção de reprodutibilidade: o transfer record C0 guarda SHA-256 bruto dos
  arquivos `go.mod`/`go.sum`, mas o comando literal do plano C5 usa
  `git hash-object`, que neste repositório produz SHA-1 de blob e nunca pode
  igualar aqueles campos. O checkpoint C5 usou `shasum -a 256` sobre os bytes
  (os valores continuam `0eb36e99...` e `1c7ae9b8...`); esta divergência do
  texto do plano precisa ser reconciliada antes da aceitação independente do
  gate, sem alterar os módulos ou inventar uma identidade.
- Registrar aqui qualquer bloqueio reproduzível antes de alterar a ordem ou o
  contrato do plano.
