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

- Status: `IN_PROGRESS`; T6 foi movido para o fim do backlog. HTML/PDF otimista
  está preservado. Emenda v2 foi aprovada e o replay obrigatório
  M1 -> M4 -> M5 está autorizado.
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
- Último checkpoint funcional: `e9f490795efe3388b4fd1a60b63617f5590baef4`,
  tree `637669bee842968300f6d61b316dc571357ea670`, enviado para
  `origin/impl/v0.0.1-core`. O wrapper de Table do adaptador Margo agora mantém
  16 px de ritmo antes da prosa seguinte, sem alterar o componente Goshtoso;
  os ajustes anteriores de código inline, blocos, Mermaid e watermark permanecem.
  M0-M4 ainda são
  candidatos: `darwin-arm64`, `darwin-x64` e `linux-x64` já passaram; falta o
  runner Windows, além de I1b, validação M5, executor M6,
  readiness M7 e revisão independente; o preview otimista não antecipa essa
  aceitação.

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
- Emenda proposta para desbloquear M5, ainda sem autoridade de implementação:
  `docs/MERMAID_NORMALIZATION_AMENDMENT_V2.md`.
- Linhas exatas propostas para o perfil de redução M5:
  `docs/proposals/MERMAID_NORMALIZATION_REDUCTIONS_V2.proposed.json`.
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
- C8 RED confirmou a ausência dos contratos de identidade de artefato,
  manifesto e mapeadores com o comando focado do plano. GREEN/REFACTOR passaram
  o foco e `GOWORK=off GOFLAGS=-mod=readonly go test -race ./... -count=1`.
  `ArtifactFingerprint` usa o domínio `margo/artifact/v1`, inclui documento,
  instância, tipo, serializer, configuração, engine e projeção terminal
  canônica, mas exclui `ExecutionID`; `ArtifactDigest` cobre exatamente os
  bytes emitidos. `Manifest` clona defensivamente, valida caminhos/duplicatas e
  digere entradas ordenadas; `AdjacentMapper`, `PreserveMapper` e `FlatMapper`
  mapeiam apenas caminhos fornecidos, sem descoberta/globbing, rejeitando escape
  do source root. C8 commit local: `fe7c77956700c2fea48ca4b24a40d176be173de2`,
  tree `af706f547f0edc25ac1ebc79a178dece93f40f66`; checkpoint staged com os seis
  caminhos exatos e filtros proibidos passou.
- T0 foi executado numa worktree externa dedicada
  `/private/tmp/gs-margo-v001-t0`, baseada em `origin/main` após `git fetch`.
  O RED literal do plano não falhou porque o cache local já continha o templ;
  a verificação não usou binário ambient e retornou `v0.3.1020`. O comando de
  download literal do plano para `github.com/a-h/templ/cmd/templ@...` é
  inválido no Go (esse caminho é pacote de ferramenta, não módulo); a execução
  verificável usou o módulo correto `github.com/a-h/templ@v0.3.1020`, mantendo o
  `go get -tool` exato como único escritor. `go mod verify`, versão e geração
  templ passaram sob `GOWORK=off GOFLAGS=-mod=readonly`, sem drift de módulos.
  T0 commit externo: `fe63b98b673e9e17bff026de6b8ef9010b072957`, tree
  `8f79fb236c08a84f4b66c4c5592a2964ecee4e0e`; `go.mod` SHA-256
  `ef79634c6c638af738e2376cf5ad5def138d5ee895d4b21196e014afec05531f`,
  `go.sum` SHA-256
  `144bd7e6c648060d43c19d985516b300f3f7ae199cfc86a00c5378f4849bcf80`, lock
  SHA-256 `b1c9775a3fbdbb6d9704416fae3292650ebec2ab38cd574e8165fea4eb450642`.
  O T0 checkpoint staged somente `go.mod`, `go.sum` e
  `tools/table-toolchain.lock`; a worktree permanece preservada para revisão.
- T1 foi executado numa segunda worktree externa
  `/private/tmp/gs-margo-v001-t1`, derivada do T0 aceito local
  `fe63b98b673e9e17bff026de6b8ef9010b072957`. O RED de teste foi inconclusivo
  porque o checkout ainda não tinha os testes/símbolos novos e o filtro não
  encontrou testes; a implementação então adicionou `SortModeAuto|None|Server|Client`,
  `Cell.SortValue`, resolução matricial, validação fail-closed e wrappers de
  render que validam antes do primeiro byte. O foco T1 e a suíte completa de
  `components/table` passaram sob `GOWORK=off GOFLAGS=-mod=readonly`.
  T1 commit externo: `243212745e09f920762feba6454927d4d3858351`, tree
  `d1e39f28b4fe1159eafdb0ffa8d13a169345ff5a`; os hashes de `go.mod`/`go.sum`
  permaneceram `ef79634c...`/`144bd7e6...`. O checkpoint contém somente os
  quatro caminhos T1 e não toca templates ou JavaScript gerados.
- T2 foi executado na worktree serial `/private/tmp/gs-margo-v001-t2`, derivada
  do T1 `243212745e09f920762feba6454927d4d3858351`. O RED mostrou que `none`
  ainda emitia `hx-get`; a correção limitou o header sortable aos modos `auto`
  e `server`, regenerou `components/table/table_templ.go` com o templ pinado e
  manteve o servidor legado intacto. O foco T2 e a suíte completa de
  `components/table` passaram sob `GOWORK=off GOFLAGS=-mod=readonly`.
  T2 commit externo: `bccff2fdf35ddcac975181ef9c2f083c866c4147`, tree
  `8644cecc9fabd69823db6d0b3c3f55b40ab40d31`; checkpoint contém somente
  `table.templ`, `table_templ.go` e `sort_render_test.go`.
- T3 foi executado em `/private/tmp/gs-margo-v001-t3`, derivada do T2
  `bccff2fdf35ddcac975181ef9c2f083c866c4147`. O RED não encontrava
  `data-table-client-sort`; o GREEN adicionou botão nativo, `aria-sort`,
  `data-table-sort-key`, índices de origem zero-based e `data-sort-value` com
  prioridade para `Cell.SortValue` e fallback textual normalizado, sem HTMX em
  modo client. A geração templ pinada e foco T3/T2 passaram; a suíte completa
  de `components/table` também passou. T3 commit externo:
  `0a4a67aca943854fdfffa2c7cd6a4b3790ee665c`, tree
  `6d3a03c55f02fcefb8b6ec6244957b35e8beb04d`; checkpoint contém somente
  `table.templ`, `table_templ.go` e `client_sort_render_test.go`.
- T4 foi executado em `/private/tmp/gs-margo-v001-t4`, derivada do T3
  `0a4a67aca943854fdfffa2c7cd6a4b3790ee665c`. O RED inicial do teste
  `TestTableClientSort` falhou porque o runtime ainda não inicializava o modo
  client; o GREEN/REFACTOR adicionou comparação natural estável, ciclo
  source/ascending/descending/source, ordenação apenas no `tbody`, preservação
  de grupos de linhas e sentinelas, foco nativo do botão e atualização de
  `aria-sort`, sem requisições HTMX. O fixture DOM executado pelo Node passou
  para ordem natural `item 1`, `item 2`, `item 10`, ordem reversa e retorno à
  ordem de origem. O comando exato do Step 4 passou: teste Go/Node focado,
  `just js`, `just js-check` e suíte `components/table`, todos sob
  `GOWORK=off GOFLAGS=-mod=readonly`.
- T4 commit externo: `3e12fc6326b03c507cd4d96c506df68d9728b0a6`, tree
  `9079b75bbc15215785063d0c6fff9375ddd3ffd0`; checkpoint staged somente os
  três caminhos T4 e passou os manifestos name-status/summary/raw,
  `git diff --cached --check` e o filtro de paths proibidos. Hashes finais:
  `assets/js/src/components/table.js`
  `3a74ecf87998ca72f7ecadede7d92726447791b374e0775dc4bb449634713e1a`,
  `assets/js/src/components/table_test.go`
  `c0f58dda43471246d3eabb50b6a9ef267acef072bee1d12e48ed782fa489a427` e
  `assets/js/goshtoso.min.js`
  `d1fc46d93b7100a3a0772cc671874c6bb7567eacd4057931fe6caac41d4c9e39`.
- T5 foi executado em `/private/tmp/gs-margo-v001-t5`, derivada do T4
  `3e12fc6326b03c507cd4d96c506df68d9728b0a6`. O RED de impressão foi
  reproduzido com o fixture DOM antes da restauração; o GREEN/REFACTOR passou
  `beforeprint` para a ordem de origem e `afterprint` para a ordem ativa,
  preservando o botão focado e `aria-sort`. A cobertura E2E injeta uma tabela
  client-only na página real, prova ordenação natural, ciclo reverso,
  impressão e ausência de requests de sorting. `templ generate`, `just js`,
  `just js-check`, `skillgen`, root tests, integração current-source,
  deployability pinned e o E2E focado passaram com o workspace temporário
  explícito `go.work` quando necessário.
- T5 commit externo: `dafb16aefdffd424d99e46d40fdec40de66509c7`, tree
  `9b6849fec23041aa602f811aaa90ab4b5910a3eb`; checkpoint staged somente os
  seis caminhos T5 e passou manifestos name-status/summary/raw, modo e
  `git diff --cached --check`. Hashes finais:
  `assets/js/src/components/table.js`
  `6b1532f48fb55e26aec87091af88124f3d93c15fe51ec94b1bf97a20c4f31461`,
  `assets/js/src/components/table_test.go`
  `bc8fde103200ce3d5e0ddf5fae5e502393275801c83857648e43dd3bdbb42847`,
  `assets/js/goshtoso.min.js`
  `ef42642a39b9d843761610231d9b1a77acf505155e811611c8d683a32b63736b`,
  `site/tests/e2e/table_client_sort_test.go`
  `ba6a37874086be664cff2bfeafde0ae40301da4e90b916bcb0b2d48865595eab`, e as
  duas referências geradas têm SHA-256
  `2e94489a5fbbbf6e1c30a44c7e029baa548ca6c5979c82ebaf06327aee544d4b`.
- T6 RED foi executado em `/private/tmp/gs-margo-v001-t6`, derivada do T5
  `dafb16aefdffd424d99e46d40fdec40de66509c7`: `GOWORK=off GOFLAGS=-mod=readonly
  go test ./internal/releasehandoff -run TestTableHandoff -count=1` falhou com
  `directory not found`, como esperado porque o registro/verificador ainda não
  existe. A implementação do T6 está parada nesta fronteira até que o dono
  autorizado do Goshtoso publique a release/tag imutável e entregue o recibo
  concreto (moduleVersion/tag, releaseCommit/tree, artefatos, CI, owner e
  transporte). Nenhuma identidade, tag ou release será inventada pelo Margo.

### 2026-08-07

- Auditoria de continuidade: o worktree Margo `impl/v0.0.1-core` permanece
  limpo. O worktree T6 `codex/margo-v001-t6`, derivado do T5 aceito
  `dafb16aefdffd424d99e46d40fdec40de66509c7`, agora contém somente os caminhos
  de implementação do verificador T6 como alterações não commitadas; o
  `release/table-handoff.json` concreto ainda não existe. Nenhuma tag, release,
  push ou mutação externa foi feita.
- Artefatos HTML verificados e apresentados ao usuário: o preview semântico
  fragmentário em `/tmp/margo-semantic-preview/margo-semantic-preview.html`
  permanece com 3.225 bytes e SHA-256
  `3ae8855b93dbae1d98c21b1ca5cb11b1d965316237fddf025fb528fe8772d155`.
  O preview atual é um documento standalone completo em
  `/tmp/margo-contract-preview/margo-contract-preview.html`, gerado pelo
  `RenderStandalone` do Margo com HTML semântico, tabela, código, CSS embutido,
  fingerprint e sem rede; tem 5.685 bytes e SHA-256
  `9da32cd65c1cdd2838e5c859d220903f362a08dbcd2682dc0be69634219fa1b6`.
  O helper foi executado com `GOWORK=off GOFLAGS=-mod=mod go run .` em um
  diretório temporário que referencia somente o worktree local do Margo; a
  saída foi validada com exatamente um `doctype`, um `<html>` e um
  fechamento `</html>`. A correção de offsets do frontmatter preserva a
  linguagem `go` no bloco de código e o CSS de impressão remove o cabeçalho
  interativo de copiar do PDF.
- PDF local foi produzido a partir do mesmo HTML standalone pelo Chromium
  instalado, sem download ou fallback:
  `/tmp/margo-contract-preview/margo-contract-preview.pdf`, 113.762 bytes,
  SHA-256 `36e89cf0f456fd9079969c51f748afe11345d19127be5ac3b2e0fc4cb9b41597`.
  `file` identifica PDF 1.4 de uma página; `pdfinfo` confirma Letter 612x792,
  tagged, sem JavaScript e sem criptografia; `pdftotext` recupera o título,
  conteúdo, tabela e código. Esta é prova de saída local, não substitui os
  gates P1-P7 nem cria autoridade T6.
- Corpus humano de produto adicionado em `testdata/markdown/`: o documento
  otimista `margo-full-feature-set.md` cobre shell, Markdown rico, tabelas,
  código, charts, Mermaid, paginação, HTML/PDF e a fronteira de release. As
  fatias executáveis em `testdata/markdown/slices/` isolam shell, tabela,
  código e composição pequena. O loop obrigatório passa primeiro por uma fatia
  e só depois pelo documento grande, para não usar uma regressão de composição
  como diagnóstico de uma feature isolada.
- HTML/PDF do corpus foram gerados para revisão humana em
  `/tmp/margo-human-review-latest/`. Manifesto atual:
  `manifest.tsv`; o documento grande gera HTML de 21.353 bytes, PDF de
  222.948 bytes e 4 páginas; as quatro fatias geram PDFs de uma página. O
  corpus também cobre decks estáticos, CLI/mapeamento, acessibilidade e
  metadata social como contratos otimistas para as próximas tarefas. Esses
  arquivos são artefatos de revisão local, não entram no handoff T6 nem são
  tratados como release.
- A suíte root foi reexecutada no HEAD `6e0fc7e` antes deste registro com
  `GOWORK=off GOFLAGS=-mod=readonly go test ./... -count=1`: os pacotes root,
  `internal/authority`, `internal/canonicaljson`, `internal/htmlpolicy` e
  `internal/socialcheck` passaram; não houve alteração de módulos ou arquivos
  de implementação.
- Os gates adicionais de qualidade passaram no mesmo checkout: `GOWORK=off
  GOFLAGS=-mod=readonly go vet ./...` e `GOWORK=off
  GOFLAGS=-mod=readonly go test -race ./... -count=1`. A prova de corrida passou
  para os mesmos pacotes; o resultado não altera o bloqueio de integração T6.
- Consulta remota somente-leitura com `git ls-remote --tags` confirmou tags
  Goshtoso `v0.1.2` (`29838e67aa4b28aaa43fd5b6e15b0116ca597347`) e posteriores,
  mas não forneceu `release/table-handoff.json` nem a prova de que uma dessas
  tags é o release autorizado derivado do T5 desta sessão. Uma tag existente,
  sem recibo canônico, owner, CI e digests do T5, não satisfaz o T6.
- A API pública de releases foi consultada somente-leitura para `v0.1.2` até
  `v0.1.7`: os assets publicados são apenas `styles.css` e
  `goshtoso-theme.css`; não há asset `table-handoff.json` ou equivalente. O
  mesmo bloqueio T6 persistiu em várias continuações, e não resta uma ação
  segura dentro deste checkout que possa produzir a autoridade externa.
- T6 local avançou sem inventar autoridade: `internal/releasehandoff` e
  `tools/verify-table-handoff.go` implementam o schema fechado, digest
  canônico sem `recordDigest` no preimage, validação do owner distinto do
  reviewer, identidade da árvore revisada, artefatos gerados, recibo do owner
  e digest dos bytes da fonte de `head`.
- Gates locais T6 passaram no worktree `/private/tmp/gs-margo-v001-t6`:
  `GOWORK=off GOFLAGS=-mod=readonly go test ./internal/releasehandoff
  -count=1`, `GOWORK=off GOFLAGS=-mod=readonly go test ./components/table
  ./internal/releasehandoff -count=1`, `go vet ./internal/releasehandoff
  ./tools` e `git diff --check`. O CLI também falha fechado com
  `table.handoff_arguments_required` sem argumentos.
- Decisão de prioridade: o handoff externo T6 não bloqueia mais a produção de
  HTML/PDF. Ele fica deferred até o fim do backlog; nenhuma release/tag será
  inventada, e o recibo continuará sendo exigido antes de I1a/publicação final.
- Integração visual standalone concluída em `5d17d4d`: o render chama
  `github.com/araihu/goshtoso/assets.StylesCSS()`, embute esses bytes antes de
  `assets/document.css` e emite `data-theme="modern"` por padrão. Testes
  comprovam igualdade byte a byte do CSS Goshtoso, ordem dos estilos, override
  `minimal`, ausência de dependência de rede e terminação correta dos tokens.
- `assets/document.css` permanece escopado a `.goshtoso-document` e agora cobre
  somente ritmo de headings/prosa/listas, largura de leitura, links, inline
  code, blockquote, dark mode, overflow e print. O contrato visual do produto
  foi registrado em `PRODUCT.md` para impedir a criação de um segundo design
  system dentro do Margo.
- Gates finais desta mudança passaram: `templ generate` sem atualizações,
  `GOWORK=off GOFLAGS=-mod=readonly go test ./... -count=1`, `go vet ./...`,
  `go test -race ./... -count=1`, `git diff --check` e detector Impeccable sem
  findings. `go.mod` e `go.sum` permaneceram byte-idênticos.
- Artefatos humanos atuais: HTML
  `/tmp/margo-human-review-latest/html/margo-full-feature-set.html`, SHA-256
  `1767ff89c0ba9853ec809f2609447c4256b0d42439ad1cc90f26b1924b7b5e4b`;
  PDF `/tmp/margo-human-review-latest/pdf/margo-full-feature-set.pdf`, SHA-256
  `1ce0e006a52ab7aa2824d5a9a145a12d64f9b7a438b61bf6c57addae404d5b95`,
  155.463 bytes, cinco páginas, tagged, sem JavaScript ou criptografia. A
  matriz visual passou em 390/1440 px, temas modern/goshtoso/minimal e modos
  claro/escuro, sem overflow de página, erros de console ou requests remotos.
  O tamanho ainda é Letter; A4 pertence ao pipeline PDF/frontmatter posterior.
- Lane de suporte Goshtoso reconhecida na task
  `019fda89-cab7-7ea2-871e-4c5a1673bbb1`. Ela pode ser acionada para pesquisa
  de comportamento do repositório, dúvidas de integração/API ou mudanças
  Goshtoso estritamente escopadas. Nenhum trabalho foi delegado neste
  checkpoint. Qualquer mudança de código nessa lane deve usar worktree novo de
  `origin/main`, executar geração/testes relevantes e devolver branch, arquivos
  e gates exatos. O recebimento foi confirmado diretamente para a task.
- Lane de suporte Manja/control-plane reconhecida na task
  `019fda8e-05e5-7aa0-a9c9-0cf42c470dd9`. Ela permanece sob demanda e sem
  mutações: pode pesquisar comportamento, arquitetura, integração ou API e só
  faz mudança escopada com isolamento e gates quando solicitada. Nenhum merge,
  push, tag, release, publicação, promoção ou cleanup foi autorizado.
- Auditoria de próxima tarefa detectou uma lacuna no plano D1: a API pública
  atual do `Compiler` não oferece a `deck.Compile` uma forma de provar que
  `deck.Extension()` está registrado, enquanto D1 não possui os caminhos do
  compiler root. Nenhuma API pública foi inventada; D1 permanece aguardando
  reconciliação desse seam.
- O M0 foi iniciado como trabalho de infraestrutura reversível para destravar
  runtime/PDF, apesar de I1b permanecer deferred junto com a cerimônia T6. O
  RED exato partiu de `test/browser/.cache` ausente e falhou em
  `cd test/browser`, comprovando que não havia harness ou fallback ambient.
- Os locks M0 fixam Node `v26.5.0`, npm `11.17.0`, Playwright `1.52.0`,
  css-tree `3.1.0` e Chromium revision `1169`/version `136.0.7103.25` para
  `darwin-arm64`, `darwin-x64`, `linux-x64` e `windows-x64`. Os quatro
  arquivos Node e Chromium foram baixados e tiveram os SHA-256 do plano
  reproduzidos; os executáveis Node registrados têm SHA-256
  `cbee2298...`, `272dc328...`, `3de740a9...` e `119d6fa7...`. O npm CLI
  POSIX tem `8e5f6f34...`; o Windows tem `3ce7cba6...`.
- No runner local, a assinatura Node passou com `GOODSIG` e `VALIDSIG
  C82FA3AE1CBEDC6BE46B9360C43CEC45C17AB93C`; fixtures de keyring errado e
  assinatura adulterada falham. O Chromium `darwin-arm64` instalado tem
  executável SHA-256
  `76ed7250f9edf622ce49b35b3d9b999d0200fad34654c5603fc6bb3313814b68`.
- O cache npm agora usa schema canônico `margo/npm-cache/v1`, digest próprio e
  nove linhas (root omitido mais oito tarballs). Os bytes verificados são
  inseridos diretamente no cacache do npm pinado; check mode compara receipt,
  lock, root, versões, conjunto exato de cache keys e bytes SHA-256. Provas
  negativas cobrem root errado, lock stale, pacote ausente, bytes adulterados
  e entrada extra, sempre antes de `npm ci`.
- A primeira execução Playwright travou antes de criar Chromium. A reprodução
  mínima isolou incompatibilidade entre o loader ESM de Playwright 1.52
  (`module.register`) e Node 26.5.0. `PW_DISABLE_TS_ESM=1` faz o Playwright usar
  ESM nativo e preserva os `.spec.mjs`. Depois, um segundo RED mostrou
  `executablePath` no nível incorreto; movê-lo para `use.launchOptions` fez o
  runner usar somente o Chromium verificado, sem download silencioso.
- Gate M0 local final: cache imutável durante bootstrap `--check`, `npm ci
  --offline`, cinco testes Playwright em um worker e zero requests de página.
  Passaram locks, cache, DOMParser/XMLSerializer, CSSOM/computed style,
  css-tree, estrutura/CSS desconhecido e signer/signature negativos. Saída:
  `margo.harness_ok node=v26.5.0 npm=11.17.0 chromium_revision=1169
  chromium_version=136.0.7103.25 network=0`.
- M0 commit local: `ea5960fe714f9d3824b878dbfaf1982e148c4685`, tree
  `d81be549d0f80327dff1ac2322f29af1ed4f1e4b`, exatamente 25 caminhos sob
  `test/browser` pertencentes a M0. O checkpoint passou manifestos staged
  name-status/summary/raw, `git diff --cached --check`, filtros proibidos e
  hashes C0 intactos. Cache, `node_modules` e resultados foram movidos para
  fora da árvore; são regeneráveis e o checkout voltou a conter só fonte.
- `2026-08-07T04:54:00-03:00`: o segundo runner M0, `darwin-x64`, foi
  executado integralmente neste host via Rosetta, não apenas inspecionado. A
  primeira tentativa falhou fechada antes de `npm ci` porque `/tmp` não é o
  realpath canônico no macOS; a repetição usou
  `/private/tmp/margo-m0-darwin-x64-r17`. Node `v26.5.0` reproduziu SHA-256
  `272dc3281fd8aec27d7306dac185c34bfea8c02563b73a223716ca11240913a1`,
  npm `11.17.0` reproduziu
  `8e5f6f3429f8cdbe693cdc29904e9d5a7b127a494bd15c804bd54c7403bfcbe7`
  e Chromium revision `1169`/version `136.0.7103.25` reproduziu executável
  SHA-256
  `09fe8230222c6045bca032a8a1d29cced4b871b4b1ce20b8e44960bbf5acdf86`.
  Os cinco testes `@margo-harness` passaram duas vezes, a segunda em 11,9 s,
  com `network=0`. O manifesto do cache tem 17 arquivos, SHA-256
  `947b0e1d9542cf89462054ef27d0f0c25bec514883624288c927f3857811c274`
  e ficou byte-idêntico antes/depois (`cmp` verde). Identidades auxiliares:
  browser receipt
  `66a46413425eb1ffa555386ee214247a70370fad6fa04ee85e46527c73726e95`,
  npm-cache receipt
  `50294c99b9718c3410ab68e4e15b999586cb4d695cc599e871daa6e7c58f0792`
  e checked env
  `65043a57b49f04f5fed8163911706f38e0697db31cfdda2c48d4bc27a78dae9e`.
  Material regenerável foi preservado fora da árvore em
  `/private/tmp/margo-m0-darwin-x64-r17/worktree-generated`; o worktree
  rastreado voltou a ficar limpo antes deste registro.
- `2026-08-07T05:02:50-03:00`: o terceiro runner M0, `linux-x64`, foi
  executado em container `linux/amd64` sob Colima arm64/QEMU. Um bind inicial
  por VirtioFS foi rejeitado como prova porque não preservou os symlinks
  `bin/npm`/`bin/npx`; essa tentativa permanece em
  `/Users/guilhermecastro/Documents/Codex/2026-08-07/margo-m0-linux-x64-evidence/failed-bind-generated`.
  A prova aceita usou o container preservado
  `margo-m0-linux-x64-r17-v2`, filesystem Linux nativo e somente
  `git archive` do commit `c7ce6494b5b80935b788da7ad6ac41c597244729` como
  fonte. Node `v26.5.0` reproduziu SHA-256
  `3de740a941a630439b22b79aafb383f4912b76354a2694765e8d59a033746538`,
  npm `11.17.0` reproduziu
  `8e5f6f3429f8cdbe693cdc29904e9d5a7b127a494bd15c804bd54c7403bfcbe7`
  e Chromium revision `1169`/version `136.0.7103.25` reproduziu executável
  SHA-256
  `88aa5dcd6b16a1692c77445b64343e192fef3051ad6537bb2819ff97392b2ee2`.
  Os cinco testes `@margo-harness` passaram três vezes (`13,1 s`, `11,6 s` e
  `11,8 s`) com `network=0`. A terceira passagem comparou 17 arquivos do cache
  byte a byte; manifesto SHA-256
  `fa14db6c0077865b9b7519625312d566dbf11e6a7ac44ceac82ed0395685374d`.
  Recibos copiados para
  `/Users/guilhermecastro/Documents/Codex/2026-08-07/margo-m0-linux-x64-evidence-v2`:
  browser
  `759576d47162ca39323324d10c11b84e2b3610c5fa4bc7c3222c8b1231f774af`,
  npm cache
  `d02174cea1cfda2e9b4448359382761c177f6be003b287211683c6255569d3c2`
  e checked env
  `2a7c8db9fe5e22b9a0b42d3a8a97133f5cfddb00242c02334599f838ff41147c`.
  O worktree detached Linux e o worktree principal ficaram limpos; container e
  caches foram preservados, sem merge, publicação ou cleanup.
- M1 RED falhou primeiro porque `github.com/araihu/margo/assets` e o perfil
  ainda não existiam. O comando literal também referencia
  `./internal/svgprofile`, embora esse pacote e seus arquivos pertençam
  exclusivamente a M5; M1 não antecipou esse ownership.
- A inspeção do byte pinado descobriu que `mermaid.esm.min.mjs` não é
  standalone: ele importa chunks relativos. O manifest Muamba foi ampliado
  dentro do ownership `assets/mermaid/*` para a closure ESM estritamente
  necessária às famílias fechadas `flowchart` e `sequence`: 35 downloads no
  total, incluindo runtime, licença e 33 módulos. `TestMermaidSupportedFamilyESMClosureIsEmbedded`
  percorre imports estáticos e os imports dinâmicos admitidos e rejeita arquivo
  ausente ou extra.
- Identidades M1: runtime SHA-256
  `028ee006287b85ea6ee0a670f5d9f10e22e8e46e33c7e345fbf1f60d443a5c21`,
  hash Muamba
  `sha384:4ebed2d056672dc504310c8a5be4d28abe2b2a08c0c11487650801f9528cb8cb2ad6faf66bbb1ae9db2aeff023fd414f`,
  asset-set digest
  `sha256:cacf0c0b392817ddbf4c9cdab673c511a77eb13789178965820d7a41365dd390`
  e `ValidatorProfileFingerprint`
  `e77eb18195f8509b1c52ea4f32c1bcb5ba948122a3f196438e3646e09c2dc5cf`.
- O corpus M1 contém oito fontes: `basic`, `conditional`,
  `id-reference-heavy` e `style-heavy` para flowchart e sequence. Chromium
  verificado renderizou todas usando somente os arquivos locais, com zero
  requests não-`file:`. O audit temporário está em
  `/tmp/margo-mermaid-audit/render-audit.json`; não é artefato de release.
- Gates M1 passaram: `muamba verify --strict`, `generate-go --check` com
  `--package assets`, testes focados de identidade/preimage/corpus, suíte root,
  `go vet` e race dos pacotes M1. `go.mod`/`go.sum` continuaram nos hashes C0.
  O checkpoint contém exatamente 48 paths M1. Commit local:
  `c27b06d6b5405ccab830ad25832eea54d925ed2b`, tree
  `56b0324a58dfc46f51ffd973f8cf8a3102675be7`.
- M2 RED confirmou a ausência do compilador/preflight Mermaid. GREEN/REFACTOR
  congelaram `TaskDescriptor` com ordinal, SHA-256 da fonte, runtime digest,
  profile fingerprint, ID determinístico e hash da configuração fixa. O
  preflight rejeita frontmatter dentro do fence e diretivas `init` ou
  `initialize` mesmo com variação de case/whitespace; a política aceita somente
  o literal `strict`. O hook privado roda durante `Compile`; M2 não antecipa a
  execução browser ou SVG pertencentes a M3-M6.
- Gates M2 passaram: suíte focada, `GOWORK=off GOFLAGS=-mod=readonly go test
  ./... -count=1`, race focado repetido dez vezes e `git diff --check`. Commit
  funcional `41bc8a94a4785b2a69e0c774fa2fc94ba8505edd`, tree
  `28b5ec4d7789486a1f4d5811d27ad11fe2dab7ea`, enviado para
  `origin/impl/v0.0.1-core`.
- Preview humano otimista gerado a partir de
  `testdata/markdown/margo-full-feature-set.md`: HTML standalone com o CSS
  Goshtoso embutido diretamente e `data-theme="modern"`, seguido de impressão
  pelo Chromium local pinado 136.0.7103.25. Saídas locais:
  `output/html/margo-v0.0.1-optimistic.html` (290994 bytes, SHA-256
  `3915d8f8cf78a4f9e348d4000dfa9b9c25bcc4b8bcdac63e8843e1b55f9631f7`) e
  `output/pdf/margo-v0.0.1-optimistic.pdf` (155609 bytes, PDF 1.4, Letter,
  cinco páginas, SHA-256
  `5a7c7e787b5963508c66d8a3878b8e553cc4d622a6402cd149667b3b2c9cfbba`).
  Todas as páginas foram renderizadas para PNG e inspecionadas: sem cortes,
  sobreposições ou tabela quebrada. É um preview otimista do HTML atual,
  impresso externamente; não é evidência de que o backend PDF nativo P1-P7 ou
  a execução Mermaid M3-M6 estejam concluídos.
- O RED browser M3 revelou que `playwright.config.mjs` descobria somente
  `harness/*.spec.mjs`; por isso os specs downstream reservados a M3-M7 nunca
  executariam. O seam M0 foi corrigido para `**/*.spec.mjs`, mantendo os cinco
  testes `@margo-harness` verdes. Commit isolado
  `b8dbbf24e3a9acd7e30b14b87d9bc68af51eebf6`, tree
  `0a95686587f7c392c6a35f71477cc0ba0e4ecba0`, enviado ao origin.
- M3 RED confirmou `RuntimeDescriptor`, `RuntimeReport`, `ExecutionID` e o
  allocator ausentes. A segunda rodada RED demonstrou que Go aceitava
  `tasks:null` e chaves JSON duplicadas e que o JavaScript projetava report
  não terminal. GREEN/REFACTOR alinharam Go/Chromium em protocolo, IDs,
  grafo de dependências, duplicatas, forged/malformed reports, transições,
  isolamento, allocator base36 e projeção canônica sem routing state.
- Gates M3 passaram: suíte Go focada, `go test ./...`, `go vet ./...`, race
  focado dez vezes, quatro testes Playwright `@runtime-schema` em Chromium
  136.0.7103.25 com network 0, `git diff --check`, ownership exato de oito
  arquivos e hashes C0 intactos. Commit funcional
  `e9a6abd3f37167272d0f3b499ba7d873c96a9bb6`, tree
  `394b10c1138f4088d16ffe8ad5e6986b2f6b64aa`, enviado ao origin.
- Novo requisito humano: `testdata/markdown/margo-full-feature-set.md` deve se
  tornar o benchmark de integração exaustivo da biblioteca, não um exemplo
  curto. Ele precisa exercitar imagens/figuras, famílias de listas,
  CommonMark/GFM/footnotes, tabelas, links, code, HTML sob policy, Mermaid real,
  composição longa/paginação, TOC, cabeçalho, rodapé, logo/ícone, watermark,
  stamps e backdrops. O próximo checkpoint deve testar a presença e a saída de
  cada família antes de regenerar HTML/PDF.
- Benchmark exaustivo implementado e enviado em três checkpoints:
  `1a0e16f` adicionou o corpus de 434 linhas/14.710 bytes e o teste de matriz
  semântica; `08c2d7c` compôs TOC, logo, backdrop, stamps, watermark e shell
  standalone; `cec4a49` corrigiu a família Mermaid incompatível, glifo sem
  cobertura e paginação do TOC. O renderer agora prova `del`, task checkboxes,
  footnotes/backlinks, título de imagem e três placements Mermaid, preservando
  o CSS Goshtoso como base e `document.css`/`standalone.css` apenas para
  composição Margo.
- Preview final regenerado com Chromium pinado `136.0.7103.25`: HTML
  `output/html/margo-v0.0.1-optimistic.html`, 388.969 bytes, SHA-256
  `2ac835345ad60748ddab0f416cb6db9e8d0cc07797286f49d205ff69dca193d7`;
  PDF A4 `output/pdf/margo-v0.0.1-optimistic.pdf`, PDF 1.4, 14 páginas,
  416.908 bytes, SHA-256
  `81b320222362f1167f559f5387f8ff78ceb24be9056b0f9e4d395bc13faabaa7`.
  Todas as 14 páginas foram renderizadas e inspecionadas; header/logo,
  footer, `Page N / 14`, TOC, imagens vetorial/raster, listas, tabelas, code,
  footnotes, watermark/backdrop e três SVGs Mermaid estão legíveis e sem
  clipping. O report browser registra três diagramas, 32 requests locais,
  zero request bloqueado e zero console error; SHA-256 do report
  `a1bdf027f0ffb971249b87ad9e71ee9ac09e91514c87ed1dab5e6a78653930da`.
- A tentativa de usar `stateDiagram-v2` falhou fechada porque a closure ESM M1
  deliberadamente suporta apenas `flowchart` e `sequence`; o benchmark foi
  corrigido para um segundo flowchart de readiness, sem download, fallback ou
  falsa alegação de suporte. A transformação browser do preview insere os SVGs
  reais, mas não substitui a futura aceitação M4-M7.
- Gates finais do checkpoint passaram: foco benchmark/standalone, `go test
  ./...`, `go vet ./...`, `go test -race ./...`, browser pinado com network
  denial, `pdfinfo`, extração de furniture em páginas alternadas e `git diff
  --check`.
- M4 RED chegou ao Playwright verificado e falhou nos três testes exatamente
  por `assets/runtime/svg-normalize.js` ausente. O primeiro ensaio foi
  corretamente descartado porque o recibo de cache npm estava vinculado a
  outro caminho absoluto; o cache foi materializado no path registrado antes
  de repetir o RED, sem rede ou fallback ambient.
- M4 GREEN/REFACTOR implementou os cinco estágios aceitos: parse SVG/XML
  destacado, mapas disjuntos root/descendentes, IDs por ordem documental,
  reescrita de `href`/`xlink:href`, ARIA IDREF, presentation/marker URLs,
  inline CSS e stylesheets via css-tree/CSSOM, ancoragem única no root,
  remoção de branches sem match, serialização/reparse e varredura final de
  resolução. O corpus cobre dez IDs descendentes e rejeita root/descendant
  duplicado, referência não resolvida/externa, site desconhecido e root
  divergente antes de inserção.
- Gates M4 passaram: três testes Playwright `@svg-normalize` em Chromium
  `136.0.7103.25` com network 0, `TestNormalizationVectors`, `go test ./...`,
  race focado vinte vezes, `git diff --check` e auditoria staged literal dos
  onze paths M4. Cache, `node_modules` e resultados regeneráveis criados pelo
  gate foram removidos depois da prova. Commit funcional
  `0853bf932e264f33f06f373870d376edd0f96cfc`, tree
  `e39bfb79f6c95d96f1d4203d0039ad94abfeb053`, enviado ao origin.
- Feedback visual do PDF otimista na página 3 identificou o grande `M` como o
  `Brand.Backdrop`: em impressão ele ainda usava `inset-block-start: 35%` e o
  posicionamento inline direito herdado, deslocando o centro para a direita.
  `TestStandalonePrintBackdropUsesPageCenter` preserva o watermark textual no
  canto e exige backdrop fixo em 50%/50% com tradução de -50% nos dois eixos.
  Gates passaram: `go test ./...`, `go vet ./...`, race focado no benchmark e
  standalone, `git diff --check`. Commit funcional
  `0eee6ed0d49361fbc1b70572866d53089e5430e3`, tree
  `7578d57d8fdab46a44dbeeef9b36315e6edd673b`, enviado ao origin.
- Artefatos humanos regenerados após a correção: HTML
  `output/html/margo-v0.0.1-optimistic.html`, 389.094 bytes, SHA-256
  `56605f6fef46dcb6faf78945125c369646fa8ea45190464564c6eb4b14bd0e86`;
  PDF A4 de 14 páginas `output/pdf/margo-v0.0.1-optimistic.pdf`, 427.346
  bytes, SHA-256
  `4a2a42a877ded8aa51597a138789a16755087168965f49f7c096655429ac9046`.
  Chromium pinado `136.0.7103.25` mediu backdrop 384x384 no viewport A4,
  centro `(397, 561.5)` exatamente igual ao centro do viewport, sem request
  remoto ou erro de console. Todas as 14 páginas foram rasterizadas em contact
  sheet; página 3 foi inspecionada em 1190x1684 e o `M` ficou centralizado, sem
  clipping. Texto extraído tem SHA-256
  `b735c507c50206b6f932dd9099357319b866a0d219415cb9388a8ca1e64e24d6`.
- Feedback visual da página 8 mostrou componentes CodeBlock consecutivos com
  bordas encostadas. O Goshtoso atual já oferece `data-code-block`, mas o Margo
  fixa Goshtoso v0.1.2; `document.css` agora usa o atributo público e um fallback
  fechado `div:has(> .codeblock)`. A margem pertence à composição documental,
  não ao componente base, evitando interferência em cards e layouts compactos.
  `TestDocumentCSSSpacesConsecutiveGoshtosoCodeBlocks` passou por RED/GREEN e o
  Chromium pinado mediu `margin-top: 16px` e gaps `[16,16,16,16,16]` nos seis
  blocos consecutivos da seção 7, com zero request remoto e zero erro de console.
  `break-inside: avoid` preserva cada painel em PDF.
- Artefatos regenerados e revisados nas 14 páginas: HTML 389.232 bytes,
  SHA-256 `32de0d8734c9c4034918da8c94d33ee3a0308312103fa6f690d034135c264b18`;
  PDF A4/PDF 1.4 de 14 páginas, 418.028 bytes, SHA-256
  `6586e739d03b38be1be5b3325253fabe574454e25d6aa3a17c821895e150a4e3`;
  texto extraído SHA-256
  `37176e3f8bd292a77e913424c9e447ff10acbb7c62f9368e48f56d5f4b08cd7c`;
  evidência browser SHA-256
  `e5ef0d7c7e1a13eb76337722199ab30f5d22aa5e4567fdc8ca3211fd25425acc`.
  Gates completos passaram: `go test ./...`, `go vet ./...`, race focado,
  `pdfinfo`, `pdftotext`, inspeção visual de todas as páginas e
  `git diff --check`. Commit funcional `dfa9a3b49aeab94ebc5a8be127234bc9534ecc44`,
  tree `55ff7052e1cd6e4351559240cb7df220465ec2ad`, enviado ao origin.
- Feedback visual adicional da página 9 mostrou que a prosa imediatamente após
  o último bloco HTML ainda encostava na borda. O primeiro teste novo encontrou
  um falso positivo global em `document.css`; ele foi reforçado para extrair a
  regra exata dos CodeBlocks e então reproduziu o RED pela ausência de
  `margin-block-end`. A regra agora possui 16 px lógicos antes e depois. O
  Chromium pinado mediu `marginTop: 16px`, `marginBottom: 16px`, gaps
  consecutivos `[16,16,16,16,16]` e exatamente 16 px entre o bloco HTML e o
  parágrafo "The long unbroken literal...". Margens colapsadas preservam um
  único intervalo de 16 px entre blocos, sem duplicá-lo.
- Artefatos humanos regenerados após a correção bidirecional: HTML 389.115
  bytes, SHA-256
  `7cfd39d791a89604309f889237ff7176f2e8bf3ea9c64ecc8aa2827d3703f7ee`;
  PDF A4/PDF 1.4 de 14 páginas, 418.018 bytes, SHA-256
  `fbb223e1854230495bb15d00bc9640e77a71274e34c8c254c5adc48b4f0f61a3`;
  texto extraído SHA-256
  `56b5159e9bdc969cfef9ff37c1e132def8a2104e6f2e3237a567347b2fefccec`;
  evidência browser SHA-256
  `37422044c29359b96231534caf9d49368779597b51a28cbc07f331a362dcbec6`.
  Zero request remoto e zero erro de console; backdrop permaneceu centrado.
  Todas as 14 páginas foram rasterizadas e inspecionadas, com revisão original
  da página 9. `go test ./...`, `go vet ./...`, race focado, `pdfinfo`,
  `pdftotext` e `git diff --check` passaram. Commit funcional
  `9c0f1541bc23d6a8472ae686053f47a3e2251f7e`, tree
  `d06f48ebeb5783787e004fb8162a2ee2090b5edd`, enviado ao origin.
- Feedback visual da página 10 mostrou `Mermaid source` fechado no PDF. Uma
  tentativa somente por CSS foi rejeitada porque o elemento nativo `details`
  fechado continuou ocultando o conteúdo no Chromium. O renderer agora emite
  `<details open>`: o HTML começa expandido, mas o controle continua podendo
  recolher e reabrir a fonte. Em impressão, `document.css` reduz a fonte,
  permite quebra de linha, impede overflow horizontal e mantém resumo e código
  no mesmo bloco. O browser pinado confirmou três disclosures abertos e
  visíveis, alternância fechado/aberto no HTML, `white-space: pre-wrap`,
  `break-inside: avoid` e `scrollWidth == clientWidth` nos três blocos de
  impressão. A inspeção também eliminou o resumo órfão que antes aparecia no
  fim da página 12.
- `2026-08-07T04:38:56-03:00`: benchmark regenerado com três SVGs Mermaid
  (20.919, 26.022 e 20.875 bytes), 32 requests locais na etapa de runtime, zero
  request bloqueado/remoto e zero erro de console. HTML final 389.546 bytes,
  SHA-256 `abf7aa1c6db07f6bbf55fdf8e4e36ba6049a129f12ed02ca7d7a2e620ee53bee`;
  PDF A4/PDF 1.4 de 16 páginas, 437.012 bytes, SHA-256
  `6a4a33b4f15bf57c81fb3034b19b873bea1f08f2c0268116b7f854cd384a5a1e`;
  evidência browser 3.526 bytes, SHA-256
  `4374211a0eedd522480029d1043c6fdece37e709d41d5bc064048222245d1f59`;
  texto extraído 22.847 bytes, SHA-256
  `11916b19bf4ab4ef14e6a394100b9004759b410927dc66259e032a23ecff1cc1`.
  Todas as 16 páginas foram rasterizadas e inspecionadas. `go test ./...`,
  `go vet ./...`, race focado, hashes imutáveis de `go.mod`/`go.sum` e
  `git diff --check` passaram. Commit funcional
  `ea73b12e9d16e09dd9a78ecb5d248986eb219935`, tree
  `1166139ab2667b520a0e90cf1810f3ca70a33da2`, enviado ao origin.
- Feedback visual da página 4 mostrou que o fundo `surface-alt` isolado era
  quase indistinguível do papel no tema `modern`. A primeira variante com
  `outline-strong` resolveu contraste, mas foi rejeitada na inspeção por fazer
  os literais parecerem campos de formulário. O override final permanece
  escopado a `.goshtoso-document :not(pre) > code`: mistura 50/50 os tokens
  `surface-alt` e `outline` em OKLCH, usa borda `outline`, foreground forte e
  2 px de padding vertical; dark mode usa os pares `*-dark`. Não há cor fixa ou
  substituição do CSS base Goshtoso.
- `2026-08-07T04:47:55-03:00`: o Chromium pinado mediu os dez literais inline
  no HTML e no perfil de impressão. No `modern`, texto/fundo ficou 14,47:1,
  fundo/papel 1,24:1 e borda/papel 1,48:1, com borda sólida de 1 px e padding
  vertical de 2 px. As amostras larga e estreita foram inspecionadas, inclusive
  um literal quebrado entre linhas; as 16 páginas do PDF também foram
  rasterizadas e revisadas. Zero request remoto/bloqueado e zero erro de
  console.
- Artefatos regenerados: HTML 389.787 bytes, SHA-256
  `2d851630ff13b305bcc8283f546a93029e59579a253c4db4bf2a4274e68f3054`;
  PDF A4/PDF 1.4 de 16 páginas, 440.985 bytes, SHA-256
  `df382db2e3e4ffd129c285054d001279de8bb0da0fda6b3d90073ff2afc57d89`;
  evidência browser 14.055 bytes, SHA-256
  `baca422a8ff9b22fc479215538215cf68269c67129b16d7f153f768fdefaceff`;
  texto extraído 22.847 bytes, SHA-256
  `11916b19bf4ab4ef14e6a394100b9004759b410927dc66259e032a23ecff1cc1`.
  `go test ./...`, `go vet ./...`, race focado, hashes imutáveis de
  `go.mod`/`go.sum` e `git diff --check` passaram. Commit funcional
  `c38b3f77e05d0e177475764deb3baebb8608316e`, tree
  `fe41b94dc7b1bdb2a30f1277bebfa76a1dc0e4a1`, enviado ao origin.
- `2026-08-07T05:11:20-03:00`: feedback visual da página 8 reproduziu ausência
  de ritmo depois da tabela densa. O DOM usa o wrapper
  `[data-table-client-sort="true"]`, portanto a regra semântica aplicada ao
  elemento `table` não alcançava a prosa irmã. O teste
  `TestDocumentCSSSpacesGoshtosoTableFromFollowingProse` falhou antes da regra
  e passou após adicionar `margin-block-end: calc(var(--spacing) * 4)` somente
  ao wrapper do adaptador Margo.
- O Chromium pinado mediu 16 px entre tabela e parágrafo no perfil de tela e
  16 px no perfil de impressão, com `margin-bottom: 16px`, zero request
  remoto/bloqueado e zero erro de console. As 15 páginas foram rasterizadas e
  revisadas; a página 8 mostra separação clara sem regressão no heading ou nos
  blocos seguintes. O detector visual retornou zero achados.
- Artefatos regenerados: HTML SHA-256
  `62bc549f4afe541f75761cc2cfbd24c24cfc43149009568f477c6316a2cd78d2`;
  PDF A4 de 15 páginas, 437.197 bytes, SHA-256
  `3a5e591e3df50aac88bef9029802121cecf893a88653bb283df45856d9543ccf`;
  evidência browser SHA-256
  `8c33b59f23296a6c51af283c0ae54aba0751c28ddd2b6787dafdcdfd3098ee8c`.
  `go test ./...`, `go vet ./...`, race focado, hashes imutáveis de
  `go.mod`/`go.sum` e `git diff --check` passaram. Commit funcional
  `e9f490795efe3388b4fd1a60b63617f5590baef4`, tree
  `637669bee842968300f6d61b316dc571357ea670`, enviado ao origin.
- `2026-08-07T05:16:23-03:00`: a trilha PDF P1-P7 foi reavaliada contra o
  roadmap. P1 exige I2; I2 exige M7; M7 permanece atrás de M5. Portanto iniciar
  `pdf/` agora violaria predecessores e ownership, mesmo sendo o próximo foco
  de produto.
- A contradição M5 foi convertida em uma emenda proposta verificável em
  `docs/MERMAID_NORMALIZATION_AMENDMENT_V2.md`. Ela preserva o validador
  fail-closed e cria uma redução M4 fechada, perfilada e fingerprinted: remove
  somente branches mortos listados, expande somente os três seletores sequence
  provados, descarta somente os dois keyframes não referenciados e remove
  somente `filter:none` comprovadamente no-op. Nenhum seletor de atributo,
  at-rule ou `filter` sobrevive para M5.
- Os três artefatos de auditoria foram re-hashados e continuam exatos. O corpus
  soma 444 regras, 118 vivas, 326 mortas e 16 keyframes; as quatro provas
  sequence somam 12 expansões, todas com um carrier e um target, zero nome de
  animação vivo e zero request não local.
- Checkpoint documental da emenda: commit
  `3a53838f2bb87da4c5b7d9afc44c3a80b071416a`, tree
  `01ccb259f9c4661b3019da326a105402221cf518`, enviado para
  `origin/impl/v0.0.1-core`; arquivo com 221 linhas e SHA-256
  `59e203a19e196e297b93c4b8af95a77949a5834b064ca1ca94eb09e8e2b209b1`.
- `2026-08-07T05:23:08-03:00`: a identidade de cada linha proposta foi corrigida
  para excluir `OriginalRootID` e `NormalizedRootID`. O AST canônico substitui
  o root por `#margo-reduction-root` e IDs descendentes por ordinais estáveis.
  Uma segunda geração com todos os roots substituídos produziu bytes idênticos.
- O inventário exato foi congelado em
  `docs/proposals/MERMAID_NORMALIZATION_REDUCTIONS_V2.proposed.json`: 120 linhas
  únicas cobrem 427 branches mortos em 326 regras, três rewrites cobrem as 12
  expansões sequence, quatro linhas família/keyframe cobrem 16 ocorrências e
  uma linha `filter:none` cobre dois rules/três elementos. Manifest SHA-256
  `cd703d58c45b3e7f0ae5ab23f4d4d7ee023c419420925674855dcd8785790826`.
- Checkpoint do inventário: commit
  `61c6ce09b2f6b8ccdc4e046c5fd70b6e473e4a84`, tree
  `8337d531e1b3e2b68be6d304683e88ca3d9ebb39`, enviado ao origin. A emenda
  revisada tem SHA-256
  `4bcb0bd9a7cb282fe1abf379c0fb426ea32ca1a97ae54e35cfe7ac38d3218b9a`.
- `2026-08-07T05:23:54-03:00`: terceiro ciclo consecutivo confirmou o mesmo
  bloqueio. Branch local e remoto estão limpos e iguais em
  `9a4d80f95c832319abe9789f392429a6356b8c28`; não há aprovação posterior aos
  bytes `PROPOSED`. Iniciar M1/M4/M5 mudaria o design sem autoridade; iniciar
  P1 violaria I2/M7. A sessão deve permanecer preservada até a resposta humana
  exata registrada no gate da emenda.
- `2026-08-07T10:22:54-03:00`: product owner respondeu `aprovado` ao gate
  exato de `docs/MERMAID_NORMALIZATION_AMENDMENT_V2.md`. Aprovação vincula o
  manifest proposto
  `cd703d58c45b3e7f0ae5ab23f4d4d7ee023c419420925674855dcd8785790826` e o
  commit-base `8b680725bcde20c9a31efcd52956226db045b952`. M1/M4/M5 devem ser
  reexecutados em TDD, nesta ordem. Nenhuma ampliação de gramática, release,
  publicação, deploy, merge ou tag foi autorizada.

## Decisões e limites

- Não editar o worktree R17 aceito nem qualquer snapshot rejeitado.
- Push frequente do branch `impl/v0.0.1-core` foi explicitamente autorizado e
  deve acompanhar checkpoints verdes. PR, merge, tag, release, publicação,
  deploy e limpeza de worktrees continuam proibidos sem autorização explícita.
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

- Bloqueio M5 encerrado em `2026-08-07T10:22:54-03:00`: product owner aprovou
  explicitamente `margo-mermaid-svg-normalization/v2` e o replay
  M1 -> M4 -> M5. A implementação continua limitada às linhas congeladas no
  manifest `cd703d58...`.
- T6 deferred: ainda não há `release/table-handoff.json` concreto. O revisor
  não cria tag/publicação; o gate volta ao caminho crítico somente antes de
  I1a e do release final. Os arquivos locais do verificador permanecem
  preservados no worktree T6 até que o record concreto seja recebido.
- Atenção de reprodutibilidade: o transfer record C0 guarda SHA-256 bruto dos
  arquivos `go.mod`/`go.sum`, mas o comando literal do plano C5 usa
  `git hash-object`, que neste repositório produz SHA-1 de blob e nunca pode
  igualar aqueles campos. O checkpoint C5 usou `shasum -a 256` sobre os bytes
  (os valores continuam `0eb36e99...` e `1c7ae9b8...`); esta divergência do
  texto do plano precisa ser reconciliada antes da aceitação independente do
  gate, sem alterar os módulos ou inventar uma identidade.
- M0 candidato, não aceito: `darwin-arm64`, `darwin-x64` e `linux-x64`
  executaram o provisionamento e o browser gate. Os locks cobrem quatro
  runners, mas `windows-x64` ainda precisa executar o fluxo limpo. Este host
  não possui `pwsh`, portanto os scripts PowerShell foram inspecionados e
  mantidos simétricos, mas não receberam prova runtime local. I1b e revisão
  independente também continuam predecessores formais da aceitação M0.
- M1 candidato, não aceito: além dos predecessores formais M0/I1b, o pacote
  `internal/svgprofile` é ownership de M5. A prova M1 usa `./profiles` e
  `./internal/mermaid`; o comando literal M1 que inclui `./internal/svgprofile`
  não é executável sem violar o plano.
- M4 candidato, não aceito: o Step 4 literal manda executar
  `go test ./internal/svgprofile -run TestNormalizationVectors`, mas esse
  pacote não existe e pertence exclusivamente a M5. A prova M4 executável usa
  o path owned `internal/mermaid/normalization_test.go`; M4 não criou nem
  alterou `internal/svgprofile`. A revisão M4 precisa aceitar essa correção de
  invocação ou corrigir o plano antes da promoção.
- M5 possui contradição reproduzida no corpus pinado. Auditoria dos oito
  fixtures Mermaid 11.16.1, Chromium `136.0.7103.25`, zero request não local:
  444 style rules, 118 vivas, 326 mortas e 16 `@keyframes` (todos sem animação
  viva). Flowchart não tem seletor/propriedade proibida viva. Cada um dos quatro
  sequence outputs tem três seletores vivos `[id$="-arrowhead"]`,
  `[id$="-crosshead"]` e `[id$="-sequencenumber"]`; conditional/style-heavy
  também têm `filter: none` vivo em `.labelBox`. O normalizador M4 deve rejeitar
  at-rules e seletores de atributo antes de M5, enquanto o design exige que o
  mesmo corpus positivo normalize/valide. Evidências locais:
  `/tmp/margo-mermaid-audit/render-audit.json`, SHA-256
  `d051d53d2dd7a47e48ff2956d43dd915f933b94b08605e3f219f515d0bd227c1`, e
  `live-css-audit.json`, SHA-256
  `55c4e5a85b528a159da8b68a2265e31f54a2828564b058571d29b68428859294`.
  Correção recomendada requer decisão de contrato: pré-normalização fechada
  que remove somente regras sem match/at-rules não referenciadas, expande apenas
  os três padrões upstream sequence para IDs normalizados exatos e remove o
  no-op `filter: none`; qualquer outra at-rule, atributo selector ou `filter`
  continua falhando. Isso altera explicitamente o design/perfil M1/M4 e não
  pode ser aplicado silenciosamente por M5.
- Prova adicional limitada da proposta M5 foi executada sem editar o contrato:
  `/tmp/margo-mermaid-audit/m5-correction-proof.json`, SHA-256
  `adc80c85cc487328f70d8193fbcabd8ce73234439d1bf63b3cff3f9b5552d686`.
  Em cada um dos quatro fixtures sequence, cada seletor upstream
  `[id$="-arrowhead"] path`, `[id$="-crosshead"] path` e
  `[id$="-sequencenumber"]` atinge exatamente um carrier e um target, podendo
  ser expandido para um seletor de ID normalizado exato. Os únicos keyframes
  são `dash` e `edge-animation-frame`, com zero `animation-name` computado. Nos
  fixtures conditional/style-heavy, remover `filter:none` de `.labelBox`
  preservou `getComputedStyle(...).filter == "none"` para todos os três
  elementos atingidos. O browser registrou zero request. Esta é a evidência
  aceita para a redução fechada v2; implementação deve reproduzi-la sem ampliar
  o perfil.
- A correção proposta está congelada em
  `docs/MERMAID_NORMALIZATION_AMENDMENT_V2.md`. O gate de autorização exato é:
  `Approve margo-mermaid-svg-normalization/v2 as specified in
  docs/MERMAID_NORMALIZATION_AMENDMENT_V2.md, including the human-reviewed
  reduction profile and M1 -> M4 -> M5 replay.` A resposta `aprovado` foi
  registrada em `2026-08-07T10:22:54-03:00`; M1/M4/M5 estão autorizados nesta
  ordem. M6, M7, I2 e P1 continuam sucessores e não podem antecipar o replay.
- Bytes upstream exatos de alguns chunks Mermaid contêm whitespace terminal,
  inclusive dentro de template literals de shader. `muamba verify --strict`
  prova esses bytes; `git diff --check` foi aplicado a todos os arquivos M1
  autorais excluindo `assets/mermaid/**`. Alterar vendor para satisfazer o
  whitespace checker quebraria os hashes pinados.
- `muamba generate-go` não infere package em `assets/` quando o único arquivo Go
  é o próprio output gerado; os gates M1 usam `--package assets`. A forma literal
  do plano sem esse argumento falha com `package name required`.
- Atenção de invocação: o comando E2E literal do plano, executado da raiz com
  `GOWORK=off`, não alcança o módulo separado `site/`; a prova T5 usou o
  `go.work` temporário do checkout e `-tags='e2e table'`, mantendo
  `GOWORK=off GOFLAGS=-mod=readonly` nos gates root/site que não dependem do
  workspace. O erro da forma literal é um limite de invocação do plano, não
  uma falha do runtime T5.
- Registrar aqui qualquer bloqueio reproduzível antes de alterar a ordem ou o
  contrato do plano.
