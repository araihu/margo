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

- Status: `IN_PROGRESS`; T6 foi movido para o fim do backlog. Emenda v2 e o
  replay M1 -> M4 -> M5 estão verdes. O standalone agora projeta o mesmo
  documento em modo claro ou escuro e os HTML/PDF otimistas dos dois modos
  estão preservados. O M0 local foi reprovisionado com receipt de Chromium,
  Node/npm, cache npm e checked environment; a aceitação formal independente
  ainda não foi inferida. M6 e M7 estão implementados e publicados; I1b, os
  gates independentes e os sucessores formais continuam pendentes.
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
- Último checkpoint funcional de implementação: `d9d2ede`, tree
  `2ce4f48295ad9a23fa68f84bb9b1a0d1ff2224a4`, enviado para
  `origin/impl/v0.0.1-core`. Este checkpoint inclui o HTML otimista versionado,
  modo claro/escuro, contraste Mermaid, TOC adaptativa, fila Mermaid
  process-global, configuração congelada por tarefa, `SourceRootID`
  determinístico, normalização/validação antes da inserção, hashes/tamanhos de
  saída, readiness/composição, quebra protegida de blocos e margens de PDF
  com fundo por modo. Também corrige o instalador M0 para extrair o Chromium
  antes de criar um receipt aninhado na raiz de extração, adiciona o contrato
  local sem rede desse caso e aplica `break-before: page` inline com restauração
  do estilo original. Stamps dark preservam `--color-surface-dark-alt` em vez
  de ficarem transparentes no print. O registro anti-compaction da auditoria visual está no
  commit documental `551a165`; os checkpoints documentais mais recentes são
  `ba8bc33` (margem PDF regenerada) e `c794a0c` (auditoria readonly), tree
  funcional anterior `2eb3f7d8d226a83b921fa5023bbe9190f4820f07`, enviado para
  `origin/impl/v0.0.1-core` após o registro documental `4debc9b`. O último
  checkpoint documental antes desta nota é `5975f91`, tree
  `e7c632389e9da3e5a824c9a49008c49d14b4d233`, enviado para
  `origin/impl/v0.0.1-core`; este apontador registra o snapshot anterior para
  impedir referência stale após compactação. M0-M7 continuam candidatos até revisão
  independente; o receipt M0 está presente apenas como artefato efêmero em
  `test/browser/.cache`, e I1b/T6 e os sucessores formais seguem pendentes.

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
- Emenda aprovada para desbloquear M5 e vinculada ao replay M1 -> M4 -> M5:
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
- `2026-08-07T10:26:45-03:00`: replay M1 verde. Teste RED exigiu algoritmo v2
  e o objeto `normalizationReductions` byte-equivalente ao manifest aprovado.
  Perfil agora usa `margo-mermaid-svg-normalization/v2`; fingerprint RFC 8785
  é `cd9edc30096cae2622b8e3489361465b6bcba66ad891934353bfdfb0035fff24`.
  `muamba verify --strict`, geração `--check`, `go test ./profiles
  ./internal/mermaid` e `git diff --check` passaram. Commits `26042c6` e
  `54e0888cf0337e60529fc7ef36fefc508dc14a77`, tree
  `f4b3658d4efcc470f60f75c2ccf7a841c9702959`, enviados ao origin. Próximo
  checkpoint obrigatório: M4.
- `2026-08-07T10:50:33-03:00`: replay M4 verde. O normalizador agora aplica
  somente as reduções aprovadas pelo perfil v2: 427 branches mortos, 12
  expansões dos três seletores sequence, 16 keyframes não referenciados e duas
  declarações `filter:none` comprovadamente no-op. IDs e todas as referências
  SVG/CSS são reescritos e reparsados canonicamente; remoção de qualquer linha
  aprovada e reduções não perfiladas falham fechadas. O runner M0 oficial
  reprovisionou Node `v26.5.0`, npm `11.17.0` e Chromium
  `136.0.7103.25`/revision `1169`; os oito testes `@svg-normalize` passaram com
  `network=0`. `GOWORK=off GOFLAGS=-mod=readonly go test ./internal/mermaid
  -count=1` e `git diff --check` passaram. Commit
  `5baa7d6757363abc8020926b73286f52906c0f45`, tree
  `5580d37a405a18f06f52b29abda32d0b1b07fc43`, enviado ao origin. Cache,
  `node_modules` e resultados do harness foram preservados fora do worktree em
  `/tmp/margo-m4-harness.HhM9Px`. Próximo checkpoint obrigatório: M5.
- `2026-08-07T11:01:46-03:00`: M5 RED e implementação fail-closed inicial
  reproduziram uma lacuna do perfil, sem alterar `profiles/`. O corpus negativo
  fechado rejeita script, `foreignObject`, evento, namespace/elemento/atributo
  desconhecido, referência externa, ID não normalizado, seletores host/
  atributo/universal/sibling/pseudo proibido, custom property, at-rule,
  propriedade/função/valor CSS não listado. Os 19 vetores e os testes Go do
  schema/corpus passam. Porém os oito SVGs positivos normalizados são rejeitados
  por `mermaid.svg_css_forbidden`: todos carregam no root exatamente uma
  declaração `max-width:<número finito>px`, com valores auditados entre
  `394.9765625px` e `769.453125px`; `max-width` não está em
  `profile.cssProperties`. A correção mínima proposta é adicionar somente
  `"max-width":"length"`, mudando o fingerprint de
  `cd9edc30096cae2622b8e3489361465b6bcba66ad891934353bfdfb0035fff24`
  para `fdcd7a02605775b63074d40b4786e3f8e29fa6f1e6ec2b060ae6ba44f365fe16`.
  Isso é uma correção humana M1 exigida pelo próprio ownership de M5 e não foi
  aplicada silenciosamente. Evidência Playwright preservada em
  `/tmp/margo-m5-blocker.M4seUU/test-results`, incluindo error-context SHA-256
  `b2fffe6438dd9d0ef87aa3c280c1053423a50cc45a48d0ea216f1cdb0fdcda5f`.
- `2026-08-07T11:16:35-03:00`: product owner aprovou a correção mínima M1
  `cssProperties["max-width"] = "length"`. Nenhuma outra linha semântica do
  perfil mudou. O fingerprint novo é
  `fdcd7a02605775b63074d40b4786e3f8e29fa6f1e6ec2b060ae6ba44f365fe16`.
  `muamba verify --strict`, `generate-go --strict --check --package assets`,
  `go test ./profiles ./internal/mermaid ./internal/svgprofile -count=1`, os
  checks de sintaxe JavaScript e os oito testes oficiais `@svg-normalize`
  passaram com `network=0`. Commit
  `6560f0ca68667df2b232c1a40ddca27f3103b796`, tree
  `1d5fa8217c71ff3436f1192531757f6980ff2e32`, enviado ao origin. M5 pode
  retomar o corpus positivo contra esses bytes exatos.
- `2026-08-07T11:20:00-03:00`: o replay positivo M5 passou pelos quatro
  flowcharts depois que o validador tipou `data-points` como base64 canônico de
  JSON ASCII `[{"x":<finito>,"y":<finito>},...]`, em vez de tratá-lo
  incorretamente como path SVG. Os quatro fixtures sequence ainda falham
  fechados: cada output possui exatamente um atributo
  `stroke-width="1pt"`; `stroke-width` está listado como `length`, mas
  `valueGrammar.length` admite somente unidade opcional `px`, `%`, `em` ou
  `rem`. A emenda mínima proposta muda somente a descrição para
  `finite-number-with-optional-px-percent-em-rem-pt-unit` e o parser M5 para
  admitir `pt`; o fingerprint resultante seria
  `ff794dccb3bcb8261ab08dd5568f4eca7d49086d5f9e3dd2a88bfdae813da15b`.
  Nenhuma dessas mudanças de perfil foi aplicada sem aprovação. Error-context
  Playwright SHA-256
  `01cc2363ff2c6b92572fb1019f9daeb9a4576e0045b6934d41c9495100ee05d4`.
- `2026-08-07T11:31:33-03:00`: product owner autorizou a solução durável para
  unidades CSS/SVG. O perfil agora é a única autoridade executável:
  `valueGrammarParameters.lengthUnits` contém o conjunto fechado, único e
  byte-ordenado `["", "%", "em", "pt", "px", "rem"]`; o validador
  interpreta esse conjunto diretamente e não duplica `pt` em regex. O
  fingerprint resultante é
  `6e4899904bf55acdd2b5c39a290dbac378a7f6fdf8e904b41c38c4d9c3fdda75`.
  A regressão M4 passou nos oito testes com `network=0`; M5 passou nos oito
  fixtures positivos, 21 vetores negativos, mismatches e cinco limites reais
  de recurso. `pt` é positivo pelo corpus sequence, `cm` falha fechado e
  `data-points` aceita somente base64 canônico do JSON finito esperado.
  Checkpoints enviados: perfil M1/M4 em `4458f6e`; validador M5 em
  `2310d94ebf6ca42125703c26bd414372d898dbc2`, tree
  `d0ea7ced9f662a5da9b78aaa5ff61464c9f13935`.
- `2026-08-07T12:27:26-03:00`: corpus humano passou a seguir edge-case-first.
  O RED `TestOptimisticBenchmarkPresentsMermaidEdgeCasesBeforeHappyPath`
  falhou pela slice ausente; o GREEN adicionou
  `testdata/markdown/slices/05-mermaid-profile.md` e espelhou no benchmark os
  21 vetores negativos M5, mismatch de perfil/família, cinco limites reais,
  `pt` positivo, `cm` negativo e `data-points` canônico antes do primeiro fence
  Mermaid. Commit `1043804`, enviado ao origin.
- A primeira inspeção PDF expôs uma nova regressão: células Goshtoso Table
  recebiam `` `script`` em vez de `script`. O RED
  `TestMarkdownTableFlattensInlineCodeWithoutMarkdownDelimiters` reproduziu a
  perda; `plainInlineText` agora preserva filhos de `CodeSpan` sem delimitador.
  A regra geral slice/falha/full-artifact foi registrada no README. `go test
  ./...`, race focado, `go vet ./...`, hashes imutáveis de `go.mod`/`go.sum` e
  `git diff --check` passaram. Commit `1593ee5`, tree
  `e5f026d7b9158ebe17473a6253c17a11bd2ca015`, enviado ao origin.
- Artefato completo regenerado: fonte 17.191 bytes, SHA-256
  `8555682d1ef72b08c634d26096f1f30846ffeccddfd3e9adbf5258c4a81273fc`;
  HTML 396.762 bytes, SHA-256
  `30d1c09c4e9583faceea980d85d27c895eee692baabc42b1b89f873ddd53c2fc`;
  PDF A4/PDF 1.4 de 18 páginas, 408.088 bytes, SHA-256
  `ed99e402762cfce39c40d4efd44a117625533875417d227fcb62563df74021a9`.
  Slice Mermaid: fonte 3.060 bytes, SHA-256
  `ff15fb1272b9afdaf180c09bd4270f3da8154fbaae5091b459dfadbb07ff67ae`;
  HTML 334.812 bytes, SHA-256
  `3c0bc104353edd613933bf356d6af45d3f031f0c9a34a0efdc9daec3cf14a73a`;
  PDF A4/PDF 1.4 de cinco páginas, 170.603 bytes, SHA-256
  `4baadb62fdf27231a0521c7964b646e1e668f02b3c4f96b4cda13c14e256edca`.
  Chromium pinado 136.0.7103.25 confirmou 21 linhas, três/dois SVGs, fontes
  abertas, zero request remoto/bloqueado e zero erro de console. Páginas do
  bloco negativo e transição ao happy path foram rasterizadas e inspecionadas.
- `2026-08-07T12:33:00-03:00`: modo escuro standalone implementado por TDD.
  O RED `TestStandaloneDarkColorModeIsExplicitAndPrintSafe` falhou pelos
  símbolos ausentes; o GREEN adicionou o conjunto fechado
  `ColorModeLight`/`ColorModeDark`, `WithStandaloneColorMode`,
  `data-color-mode`, classe `dark`, validação fail-closed e impressão com
  `color-scheme: dark` mais `print-color-adjust: exact`. O segundo RED
  `TestOptimisticBenchmarkPresentsColorModeProjectionBeforeFeatureTour`
  obrigou o corpus a declarar, antes do passeio de funcionalidades, projeções
  light/dark do mesmo documento e rejeição de modo desconhecido antes da saída.
  `templ generate` não deixou update pendente; `go test ./...`, race focado,
  `go vet ./...`, hashes imutáveis de `go.mod`/`go.sum` e `git diff --check`
  passaram. Commit funcional `d424f02`, tree `fa50df4`, enviado ao origin.
- Artefatos completos claro/escuro regenerados do mesmo Markdown de 17.686
  bytes, SHA-256
  `364a4a82a2dc868a9f7002386fe652df33000f6d6a6f15188e56c629747322f3`.
  Light: HTML 398.401 bytes
  `8f42f27d9fa80d145427f173493006118fd1b05b0511c0759edcc86b09017a06`;
  PDF 408.515 bytes
  `d574dc1383f351146d9c5f5b90c5af7d2b1dced09033d3c3d12a0ac9b6cd6749`.
  Dark: HTML 398.404 bytes
  `77c69c2fda36d5bd2b1374150b8af1e7d80f23123d067bbb0f8314bb87bc5d09`;
  PDF A4/PDF 1.4 de 18 páginas, 417.694 bytes,
  `14e7c119d2c90449596ad2d575e4dd592acc2b65d636d783d0789244579c4829`.
  Chromium 136.0.7103.25 confirmou `darkClass=true`, superfícies screen/print
  escuras, três SVGs, 21 vetores, zero request bloqueado e zero erro de
  console/página. TOC, tabela negativa, transição Mermaid e página final foram
  rasterizados e inspecionados. Os SVGs Mermaid aceitos foram reutilizados
  porque a fonte dos fences não mudou e M6 ainda não está implementado; nenhum
  runtime futuro foi alegado.
- `2026-08-07T12:45:00-03:00`: revisão humana encontrou contraste insuficiente
  no `<details>Mermaid source` do PDF escuro. O RED ampliado
  `TestStandaloneDarkColorModeIsExplicitAndPrintSafe` falhou pela ausência do
  seletor escuro; o GREEN adicionou ao `document.css` o fundo, texto, borda e
  título escuros para `.margo-mermaid__source`, mantendo o SVG auto-contido
  claro. A evidência computada confirmou fundo `oklch(0.205 0 0)` e texto
  `oklch(0.87 0 0)` no modo escuro; a página 13 foi rasterizada e reinspecionada.
  `templ generate`, `go test ./...`, race focado, `go vet ./...`, hashes
  imutáveis de `go.mod`/`go.sum` e `git diff --check` passaram. Commit
  `ceea76b`, tree `b6b0a8b`, enviado ao origin. Artefatos finais agora são:
  light HTML 398.714 bytes `089a3256477d95a0595cdfe51a8ca128614b1332a0ca597c6ed7d07145967d84`,
  light PDF 408.515 bytes `f1b6e927b9ea70e7f9aca1baf7c56b7c28df985cd5979f27fed26e41907d0a30`,
  dark HTML 398.717 bytes `f26bf26c23523a761ee90946574b12a3bcce6f752fe893d44652b55d99807b70`,
  dark PDF A4/PDF 1.4 de 18 páginas, 418.878 bytes
  `316e208e8b3ca37870d6e37d7c808f95922bb8d5950b8e26a8dc009e15d5493e`.
- `2026-08-07`: o auditor de contraste virou preflight determinístico embutido
  no harness M0. `test/browser/lint-contrast.mjs` recebe HTML completo, fixa
  mídia `print`, projeta `light`/`dark`, aplica WCAG AA (4.5:1 texto normal,
  3:1 texto grande), rejeita recursos de rede e emite
  `margo/contrast-lint/v1` em JSON ou texto. `run-playwright.sh` e
  `run-playwright.ps1` expõem `--contrast-html`, `--contrast-mode`,
  `--contrast-format`, `--contrast-output` e `--contrast-only`, sempre usando
  os binários/Chromium do receipt M0. O teste unitário do CLI passou (3/3), o
  HTML fixture aprovado passou em ambos os modos, uma mutação de baixo
  contraste retornou exit 1 com três falhas, e uma folha CSS HTTPS externa
  retornou exit 1 com recurso bloqueado. `go test ./... -count=1`, `go vet
  ./...`, `node --check` e `git diff --check` passaram. Commit funcional
  `0a91c81a2be409f071743930a0c53b9c42857e03`, tree
  `7fd44f6b5a4345158d1a4c7f02e0612d14143e9d`, enviado para
  `origin/impl/v0.0.1-core`. A correção `6e15c68` mantém metadados de
  navegação como `canonical` fora do bloqueio e continua rejeitando apenas
  recursos realmente carregáveis; commit enviado ao mesmo branch.
- `2026-08-07T13:55:00-03:00`: revisão humana da TOC do PDF escuro exigiu
  superfície transparente, duas colunas adaptativas e continuação paginada.
  O RED `TestStandaloneTOCPrintLayoutIsAdaptiveAndFragmentable` foi criado;
  o GREEN/REFACTOR adicionou `columns: auto 12rem` no print, `break-inside:
  auto`, `overflow: visible`, `overflow-wrap: anywhere` e `column-fill:
  balance`. A inspeção Chromium/Poppler mostra 36 entradas em duas colunas
  na página 1, página 2 iniciando o artigo, sem texto cortado, em light e dark.
  A regra de tela usa `auto 16rem`; uma verificação em 360px produz uma coluna
  e em 794px produz duas.
- No mesmo checkpoint, o preflight completo do HTML otimista passou em ambos
  os modos: `checked=495`, `failures=0`, `network.blocked=0`, Chromium
  `136.0.7103.25`/revision `1169`. `GOWORK=off GOFLAGS=-mod=readonly go test
  ./... -count=1`, `go vet ./...`, `node --test lint-contrast.test.mjs` (4/4),
  a especificação `@contrast` e `git diff --check` passaram. Artefatos atuais:
  light HTML 331.268 bytes SHA-256
  `16e4c48e3f666025f522140af3ad0557b44ed32fef67c7da49bce36930caee74`, dark
  HTML 331.271 bytes SHA-256
  `a786d58eb8680a10305ebae75e075d43cc0e75a023e14460110fd011cfdc4826`, light
  PDF A4/PDF 1.4 de 16 páginas, 422.704 bytes SHA-256
  `351767e14440c29e98c334480ed2ca7c3488b294c8a29191697fe747f35222f9`, dark
  PDF A4/PDF 1.4 de 16 páginas, 434.622 bytes SHA-256
  `3708427a5df06701e3266ee8d7868a18c4bddf2abc21a00706f8abedcc5dda48`.
- `2026-08-07T14:00:00-03:00`: revisão humana encontrou lista começando no
  fim de página. O RED `TestStandalonePrintBlocksAvoidInternalFragmentation`
  falhou antes do contrato; o GREEN adicionou no `document.css` uma política
  print explícita: headings usam `break-after: avoid-page`; listas, citações,
  definições, disclosure, figuras, tabelas, imagens, pre/code e wrappers
  Goshtoso de tabela/código/Mermaid usam `break-inside: avoid-page`. O escopo é
  somente `article.margo-document`; TOC permanece `break-inside: auto` e
  multicoluna.
- O harness ganhou `@pagination`, que verifica os estilos computados de todos
  esses blocos e confirma TOC fragmentável. O gate direcionado passou 2/2
  (contraste light/dark e paginação), Chromium `136.0.7103.25`, revision
  `1169`, rede bloqueada. `go test ./... -count=1`, `go vet ./...` e
  `git diff --check` passaram. A suíte completa passou 21/22; o único erro é
  o fixture independente de assinatura GPG (`signature-contract`, exit 2).
  Commits enviados: `b61dc9b` (CSS/test unitário) e `9ac3e18` (teste
  Playwright), branch `origin/impl/v0.0.1-core`.
- Artefatos regenerados e inspecionados em light/dark: HTML light 331.884
  bytes SHA-256 `135d1c341dc83fa2c2ffdc4e328bc1b42a5a78439403018d3e20f2f68d824715`,
  HTML dark 331.887 bytes SHA-256
  `9b688a3c20ececfad5f3a411af8f4274f36d686258ad74cc25733f91a49a2603`, PDF
  light A4/PDF 1.4 de 17 páginas, 425.934 bytes, SHA-256
  `f48cd62d3e254700db3178203229a56a666849d55c7ae84fe37ac89591d3a8d3`, PDF
  dark A4/PDF 1.4 de 17 páginas, 437.874 bytes, SHA-256
  `2bf473b4a3141a4df8b575a9a87f186b460629d622e6711993ac99699b2ea613`.
- O mesmo contrato recebeu fallback `page-break-after: avoid`/
  `page-break-inside: avoid` para engines legados de impressão; a projeção
  Chromium permanece byte/layout-equivalente: 17 páginas A4 em cada modo.
- `2026-08-07T14:25:00-03:00`: revisão humana corrigiu a regra da TOC: ela
  reserva a página inteira, usa uma coluna por padrão e só ativa duas colunas
  quando a medição no `beforeprint` mostra que a lista não cabe na altura útil.
  Itens mantêm `page-break-inside: avoid`; `@pagination` cobre TOC curto e
  TOC alto. Commit `00739fd` enviado para `origin/impl/v0.0.1-core`.
  Artefatos atuais: HTML light 333.433 bytes SHA-256
  `e9cd17e62d5c188d468bcbd6a3f12ad5a339a07396f977b1d758ef1bf8a663a8`, HTML
  dark 333.436 bytes SHA-256
  `43ab7c61016fa4c1a40a9ed994687e4317b2bdfbbc73ca12841ecb9e30ae635a`, PDF
  light A4/PDF 1.4 de 17 páginas, 425.077 bytes SHA-256
  `7874e7bc7953e096291f6aa05cb299be8e3568f18506a03d09f69d3d41cb0959`, PDF
  dark A4/PDF 1.4 de 17 páginas, 437.020 bytes SHA-256
  `72c1425f83919b88532099a3f64295d3f0c74ae5064e2bf8b10c9d16ad737051`.

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

- Bloqueio M5 encerrado em `2026-08-07T11:16:35-03:00`: o perfil fechado não
  listava `max-width`, embora todos os oito outputs positivos pinados emitam
  essa declaração root com um valor finito em `px`. A autoridade mínima
  solicitada aprovou `cssProperties["max-width"] = "length"` e o novo
  fingerprint
  `fdcd7a02605775b63074d40b4786e3f8e29fa6f1e6ec2b060ae6ba44f365fe16`,
  mantendo toda outra linha do perfil byte-semanticamente igual. Replay M1 e
  regressão M4 passaram; M5 retomado.
- Bloqueio M5 encerrado em `2026-08-07T11:31:33-03:00`: em vez de codificar
  `pt` novamente no JavaScript, a lista fechada de unidades passou ao perfil
  tipado e o validador consome essa autoridade. `cm` e qualquer unidade não
  listada continuam falhando. Alterações futuras exigem diff do perfil, novo
  fingerprint, corpus positivo/negativo e revisão humana; não há ampliação por
  observação em runtime.
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
- `2026-08-07T14:44:53-03:00`: a revisão humana encontrou que a área de
  margem do PDF escuro, o documento e o chrome de impressão usavam superfícies
  diferentes. O shell agora publica `margo-print-page-background` e a paleta
  de chrome a partir dos tokens Goshtoso; a aplicação de tokens também cobre
  o stylesheet shell. O gerador de PDF de revisão deriva background, texto e
  borda dos valores computados, sem cores literais por modo. O Playwright
  `@pagination` passou 4/4 em light/dark, com html/body/document na mesma
  superfície; o lint WCAG passou 495 nós nos dois modos, zero falhas e zero
  requests bloqueados. Commit `1ebe09b` enviado ao origin.
  Artefatos A4/PDF 1.4 de 17 páginas: HTML light 334.524 bytes,
  SHA-256 `aa7520741fd2a23b3efc004ad3a930d0116b06ccecd3b0dfdcb866387b16581a`;
  PDF light 424.724 bytes,
  SHA-256 `6da4d2db99df4d45bb6df3203c3b287b22ecdcaa866f44f986b52df7ba0e78e2`;
  HTML dark 334.527 bytes,
  SHA-256 `782e7b64e9a212e65171fe314a137eefffc374111848ae2e95236d5539b11b16`;
  PDF dark 436.758 bytes,
  SHA-256 `c6526f9f0ca392d396d97c2b83853a35cfbc5b601646c59d2391b8147e0ff931`.
- `2026-08-07T15:39:46-03:00`: o TOC do standalone deixou de parecer um painel:
  não tem borda nem raio, usa exatamente `--margo-page-background` do documento
  em light e dark, e mantém a regra de paginação/colunas intacta. O teste
  Playwright verificou background e bordas em ambos os modos; a geração offline
  de HTML/PDF foi refeita sem requests externos. HTML light 334.553 bytes,
  SHA-256 `f8007a2314a7a0c52c96802569293467bf7af5fb4c6969ba0a762e70a63e8c7f`;
  HTML dark 334.556 bytes,
  SHA-256 `1a02a23a5954e0486786aa90dd3b9272e17e493550bf5eeabed3f6196ef0c6c9`;
  PDF light 424.326 bytes,
  SHA-256 `f24b64a0f9cfc3f15c1d39244105557de2fde44bb825ab926abd32e98fe2ebb6`;
  PDF dark 436.358 bytes,
  SHA-256 `ef69d3e78da15f1af9caaa0aeb910792b759a90a8b7487f2c2a59e5c876e3cf8`.

### 2026-08-08

- `2026-08-08T02:52:17-03:00`: M6 RED reproduzido: o spec
  `test/browser/specs/mermaid-queue.spec.mjs` falhou antes da implementação
  porque `assets/runtime/mermaid.js` não existia.
- M6 GREEN/REFACTOR implementaram `assets/runtime/mermaid.js` com fila
  process-global compartilhada entre instâncias, `initialize`/`render`
  serializados, configuração base/per-task profundamente congelada,
  `SourceRootID = msrc-<sha256("margo/mermaid-source-root/v1\\n" + instanceID +
  "\\n" + decimal8(blockOrdinal))>`, IDs de tarefa compatíveis com
  `margo-runtime/v1`, execução detached sem `bindFunctions`, normalização e
  validação antes de `XMLSerializer`/inserção, isolamento de falhas e relatório
  estrito com `outputSHA256`/`outputBytes`.
- Gate browser oficial M6 + M4 + M5 passou 15/15 com
  `./run-playwright.sh --check --env-file /private/tmp/margo-m6-offline-env.sh
  --grep '@mermaid-queue|@svg-normalize|@svg-validate'`; receipt/cache M0 foi
  reprovisionado em caminho absoluto temporário, Node `v26.5.0`, Chromium rev
  `1169`, `network=0`. Go `vet`, `node --check` e `git diff --check` passaram.
- `go test ./... -count=1` executou packages internos e parou apenas no link do
  pacote root com `no space left on device`; não houve limpeza de caches ou
  worktrees para mascarar o bloqueio ambiental. Commit M6
  `5852ae48f618f7c4313d2eb478c18afed991d1b1`, tree
  `518bf91a7b96cc2ee4ca18b7575219054bd70a49`, push confirmado em
  `origin/impl/v0.0.1-core`.
- `2026-08-08T03:14:00-03:00`: M7 RED reproduzido com seis specs
  `@readiness|@composition`; todos falharam antes de
  `assets/runtime/readiness.js` existir. GREEN/REFACTOR implementaram o
  collector terminal em `assets/runtime/readiness.js`, sem alterar o schema
  ou o runtime M6: validação detached de cada tarefa, ordem topológica,
  timeout, fontes (`document.fonts.ready` e `check`), evidência de requests
  bloqueados, quantização de métricas em 1/64 CSS px, estabilidade em no
  máximo oito frames, relatórios profundamente congelados e isolamento de
  placements/`ExecutionID`. Os fixtures explícitos estão em
  `testdata/runtime/readiness-vectors.json`; as suítes exercitam evidência
  forjada/ausente, timeout, fonte indisponível, rede bloqueada, layout
  instável, duplicidade e duas composições independentes.
- Gate browser M7 passou 8/8; o gate combinado `@runtime|@readiness|@composition`
  passou 11/11 com Node `v26.5.0`, Chromium rev `1169` e `network=0`. Também
  passaram `GOWORK=off GOFLAGS=-mod=readonly go test ./... -count=1`,
  `go vet ./...`, `node --check assets/runtime/readiness.js`,
  `go tool muamba verify --strict` e
  `go tool muamba generate-go --strict --check --dir assets --package assets
  --output mermaid_gen.go`. A forma literal do plano sem `--package assets`
  falha porque Muamba não consegue inferir o package a partir do próprio
  arquivo gerado; o gate executável usa o argumento explícito e não altera
  bytes gerados.
- M7 commit `21a2652`, tree
  `5c73b048ac5cfa902cbd64c18249c60e2acf51f4`, push confirmado em
  `origin/impl/v0.0.1-core`; worktree limpo após o checkpoint. M7 permanece
  sujeito à revisão independente, e I1b/T6/sucessores continuam fora deste
  marco.
- `2026-08-08T04:00:00-03:00`: auditoria executável do checkpoint M7 foi
  repetida no HEAD de documentação `cd3779c`, tree
  `91a1ae8bacc6e18d0a558cb2812e74f3db0d5705`, sem alterações de código.
  `GOWORK=off GOFLAGS=-mod=readonly go test ./... -count=1`, `go vet ./...`,
  `node --check` dos runtimes Mermaid/readiness, Muamba `verify --strict` e
  `generate-go --strict --check --package assets` passaram. O browser offline
  passou 17/17 (`@runtime`, `@readiness`, `@composition`, `@contrast` e
  `@pagination`) com Node `v26.5.0`, Chromium revision `1169` e
  `network=0`; os diretórios temporários do harness foram removidos e o
  worktree permaneceu limpo.
- Os quatro artefatos otimistas atuais foram re-hashados: HTML light
  `334553` bytes, SHA-256
  `f8007a2314a7a0c52c96802569293467bf7af5fb4c6969ba0a762e70a63e8c7f`;
  HTML dark `334556` bytes, SHA-256
  `1a02a23a5954e0486786aa90dd3b9272e17e493550bf5eeabed3f6196ef0c6c9`;
  PDF light A4/PDF 1.4, 17 páginas, `424326` bytes, SHA-256
  `f24b64a0f9cfc3f15c1d39244105557de2fde44bb825ab926abd32e98fe2ebb6`;
  PDF dark A4/PDF 1.4, 17 páginas, `436358` bytes, SHA-256
  `ef69d3e78da15f1af9caaa0aeb910792b759a90a8b7487f2c2a59e5c876e3cf8`.
  O HTML dark foi aberto no painel Codex em
  `output/html/margo-v0.0.1-optimistic-dark.html`.
- A próxima fronteira formal continua sendo I1a/I1b: T6 permanece no fim do
  backlog por decisão explícita, e H/P/D/O/I2-I4 não podem receber um falso
  aceite sem o handoff externo e a prova de origem. Nenhum proxy, tag, release,
  domínio ou pseudo-versão foi inventado durante esta auditoria.

### 2026-08-08 — dark standalone frame checkpoint

- `2026-08-08T03:24:22-03:00`: o feedback visual do dark mode foi convertido
  em contrato de fonte: `html`, `body` e `.goshtoso-document` usam o mesmo
  `--margo-page-background`; header/footer preservam os tokens dark de texto e
  borda; stamps preservam superfície, texto e borda dark. Isso remove a faixa
  clara entre a página e o chrome no HTML/PDF derivados, sem mexer nos tokens
  de Goshtoso.
- TDD RED foi reproduzido no teste focado `@shell uses dark page tokens for
  frame, chrome, and stamps`: antes da alteração, `html` computava `transparent`
  em vez do fundo dark. GREEN passou 1/1 após a alteração. O gate browser
  combinado `@shell|@pagination|@contrast` passou 7/7 com Node `v26.5.0`, npm
  `11.17.0`, Chromium revision `1169`, versão `136.0.7103.25` e
  `network=0`.
- O teste raiz `GOWORK=off GOFLAGS=-mod=readonly go test ./... -count=1` e
  `go vet ./...` foram tentados neste checkpoint, mas o ambiente retornou
  `no space left on device` durante a escrita do build. Isso é bloqueio
  ambiental, não aceite do produto; nenhum cache, worktree ou artefato amplo
  foi removido para mascará-lo. Os artefatos HTML/PDF ignorados não foram
  regenerados após esta mudança porque o renderer de benchmark não está
  versionado como comando reprodutível neste worktree; os hashes anteriores
  permanecem apenas como evidência do checkpoint anterior.
- Arquivos deste checkpoint: `assets/standalone.css` e
  `test/browser/specs/standalone-pagination.spec.mjs`; `GOAL.md` registra o
  estado. Próximo passo seguro: `git diff --check`, commit/push do ajuste e
  reauditoria das fronteiras formais. T6, I1b e os sucessores continuam
  deliberadamente fora deste checkpoint.
- Checkpoint versionado: commit `68e972df8915298aab0a2c7f3397e028becb7eb7`,
  tree `2f19fdd23515e1d15c64a09fdd8358b8b206ed57`, enviado com sucesso para
  `origin/impl/v0.0.1-core`. A árvore ficou limpa após o push.

### 2026-08-08 — HTML dark regenerado

- `2026-08-08T03:30:32-03:00`: o helper local
  `/private/tmp/margo-optimistic-generator` foi executado contra este
  worktree com `GOWORK=off GOFLAGS=-mod=mod`; o HTML standalone foi regenerado
  em light e dark, e os três SVGs Mermaid existentes foram reanexados ao novo
  shell. Os arquivos de revisão são
  `output/html/margo-v0.0.1-optimistic-current.html` (402.663 bytes,
  SHA-256 `69053e26a670301df2b7cba4ad8b66692e0585f782d5813ea96a339b1391fce4`)
  e `output/html/margo-v0.0.1-optimistic-dark-current.html` (402.666 bytes,
  SHA-256 `e5f1ea0458b848e8d3e7c87669e41340ce3b345089e1af66edb82ac6e1fd5fa5`).
  A inspeção estática confirma três figuras Mermaid, 21 elementos SVG (inclui
  ícones), tokens de página dark e o fundo compartilhado do frame.
- O HTML dark regenerado foi aberto no painel Codex. A tentativa de gerar os
  PDFs e executar a validação Chromium falhou antes do browser iniciar:
  `ENOSPC: no space left on device` ao criar o diretório temporário do
  Playwright. Portanto não há novo hash PDF nem nova evidência de impressão
  neste checkpoint; os PDFs anteriores permanecem somente como evidência do
  checkpoint anterior. Nenhum cache amplo, worktree ou artefato foi removido
  para mascarar a falta de espaço.
- Checkpoint versionado: commit `8f386f2de5a7bae92834a0e8487e39ab1749408b`,
  tree `1a1fabc74031b85d2d7853b314acba4a9ffe269e`, enviado para
  `origin/impl/v0.0.1-core`; worktree limpo.
- Os dois HTMLs `output/html/margo-v0.0.1-optimistic.html` e
  `output/html/margo-v0.0.1-optimistic-dark.html` agora apontam para esses
  bytes regenerados (respectivamente `69053e26a670301df2b7cba4ad8b66692e0585f782d5813ea96a339b1391fce4`
  e `e5f1ea0458b848e8d3e7c87669e41340ce3b345089e1af66edb82ac6e1fd5fa5`).
- Checkpoint versionado: commit `1c0f889a63a353043e9523ebc6a813939ac56f03`,
  tree `a85f46225278d8f7d9dfe35d2a9a10e0315a5263`, enviado para
  `origin/impl/v0.0.1-core`; worktree limpo.

### 2026-08-08 — PDF dark/light atualizado

- `2026-08-08T03:35:53-03:00`: como o Playwright não conseguia criar seu
  diretório temporário por `ENOSPC`, a impressão foi executada diretamente com
  o Chromium pinado local (`136.0.7103.25`) e os HTMLs inline/offline. Os PDFs
  A4/PDF 1.4 canônicos agora são `output/pdf/margo-v0.0.1-optimistic.pdf`
  (19 páginas, 549.140 bytes, SHA-256
  `bb5c58a16d10e8eb095a1937c5b7cc055cfa34ee8d3156e89a1fd0fc2134fd1e`) e
  `output/pdf/margo-v0.0.1-optimistic-dark.pdf` (19 páginas, 555.125 bytes,
  SHA-256 `e90af9b7991ec4745fb8ce1c1350f1b0fa89d5eb840b213d30080235382ddae9`).
  `pdfinfo` confirma A4, tagged, sem JavaScript e sem criptografia; `pdftotext`
  recupera Contents, Mermaid source, edge cases e furniture nos dois modos.
  Os bytes anteriores foram apenas renomeados para os backups ignorados
  `*.previous.pdf`, sem destruição.
- Esta é evidência de impressão direta, não substitui o gate completo do
  harness M0 (que segue impedido por espaço em disco). A validação visual
  rasterizada e a comparação WCAG dos PDFs ficam pendentes até o ambiente
  liberar espaço; não há aceite formal novo de P1-P7.
- Checkpoint versionado: commit `66c22c5395c9256bbc268220b4f3935cc7013f5d`,
  tree `8ce170d61e3bf542961b0450fa0bc3b298a44888`, enviado para
  `origin/impl/v0.0.1-core`; worktree limpo.

### 2026-08-08 — auditoria de fechamento parcial

- `2026-08-08T03:38:18-03:00`: o branch está limpo e sincronizado em
  `8a58d57f970ae214047ef8364a1abc1c1f732107`, tree
  `fa91863d9f407e822b58e26e7ca8b6f1537c5fea`; `git diff --check`,
  `git show --check` e `git ls-remote origin/impl/v0.0.1-core` passaram.
  Os quatro artefatos canônicos têm os hashes registrados acima.
- O objetivo ainda não está concluído: C0-C8, T0-T5 e M0-M7 possuem
  implementação/evidência local; T6 continua deliberadamente deferred sem
  `release/table-handoff.json`; I1a/I1b, H/P/D/O e I2-I4 não podem ser
  promovidos sem o handoff externo e os gates independentes. A validação PDF
  completa/WCAG continua ambientalmente pendente por `ENOSPC`.
- Portanto `GOAL.md` permanece `IN_PROGRESS`; nenhum `update_goal complete` é
  permitido neste estado e nenhum release/tag/push de publicação foi criado.
- Checkpoint versionado: commit `7e15ac6e13156e7bf1bd77fbcc4694e7bb66f0dc`,
  tree `d453807864fc78caab2f7c0d48cfd98f21346c29`, enviado para
  `origin/impl/v0.0.1-core`; worktree limpo.

### 2026-08-08 — auditoria de pressão de disco e limpeza Margo

- `2026-08-08T04:22:29-03:00`: a auditoria foi limitada aos recursos Margo
  em `/private/tmp`, sem tocar nos worktrees de implementação, nos snapshots
  de revisão R4-R17, nos worktrees T0-T6, no cache Node/Chromium pinado, nem
  nas evidências de Mermaid, PDF e revisão humana.
- Não havia processo ativo com caminho, comando ou ambiente Margo; não houve
  renderer, Playwright, Chromium, `go test`, `go build`, `templ generate` ou
  `npm ci` órfão desta lane. O PID `23028` continua sendo o servidor Manja em
  `/private/tmp/manja-management-demo-ui` e foi preservado. Um Chromium
  `chromedp-runner` observado anteriormente pertencia ao worktree Xisnove;
  também não foi tocado.
- Foram removidos somente sete recursos temporários, todos pertencentes ao
  usuário, sem handles abertos, sem referência no `GOAL.md` e fora de Git:
  `/private/tmp/margo-go-cache` e seis diretórios
  `/private/tmp/margo-pagination-*`. O `du` registrou `446038016` bytes
  (`~425,3 MiB`) recuperáveis. O espaço livre observado passou de `119 GiB`
  para `156 GiB`; a diferença APFS inclui compactação/reclaim assíncrono e
  não é atribuída integralmente aos diretórios removidos.
- A causa imediata foi acúmulo de um `GOCACHE` descartável e harnesses de
  paginação antigos. O helper temporário
  `/private/tmp/margo-optimistic-generator/render-artifacts.mjs` também fecha
  o browser apenas no caminho de sucesso, portanto uma exceção pode deixar
  um Chromium órfão; ele não é fonte versionada deste worktree e nenhum
  processo correspondente estava vivo. O runner versionado
  `test/browser/lint-contrast.mjs` já usa `try/finally` para `browser.close()`;
  não foi necessário alterar código de produto nesta limpeza.
- Recursos intencionais permanecem preservados: M0/M5 e os quatro arquivos
  Node, o ZIP Chromium, `margo-r5-module.qSyEnw`, `margo-mermaid-audit`,
  `margo-human-review-latest`, `margo-pdf-review-table-spacing-r17`, todos os
  worktrees/snapshots Git e os quatro artefatos HTML/PDF canônicos.
- Após a limpeza: o worktree `impl/v0.0.1-core` permaneceu limpo e alinhado ao
  remoto; nenhum gate amplo foi executado. A validação foi somente de espaço,
  ausência de processos/caminhos abertos e integridade dos recursos
  preservados. O teste unitário focado de contraste passou `4/4` ao ser
  executado sob o `node_modules` já provisionado do harness M5; a invocação
  direta no worktree foi recusada apenas porque ele não contém `node_modules`.
  Os gates completos PDF/WCAG e a aceitação formal de I1a/I1b continuam
  pendentes; T6 segue deferred por decisão explícita do usuário.

### 2026-08-08 — renderer HTML otimista versionado

- `2026-08-08T04:38:18-03:00`: os gates readonly do novo pacote
  `tools/optimistic-renderer` passaram: teste focado, `go test ./...`,
  `go vet ./...`, `charts/go test ./...` e `pdf/go test ./...`, todos com
  `GOWORK=off GOFLAGS=-mod=readonly`. O comando não inicia browser, não baixa
  dependências e consome o stylesheet Goshtoso embutido, tema `modern`,
  `Brand`, TOC, logo, watermark, stamps e modos claro/escuro.
- A implementação é coberta por RED/GREEN: o teste falhava antes da função
  `generateHTML` existir e passou depois da implementação. Os testes verificam
  modo dark, tema, classe dark, título, heading, stylesheet Goshtoso, rejeição
  de modo inválido e ausência de arquivo temporário após a escrita atômica.
- O smoke real gerou, a partir de
  `testdata/markdown/margo-full-feature-set.md`,
  `output/html/margo-v0.0.1-optimistic.html` (334.695 bytes,
  SHA-256 `8594055e5f854ab8b3a9307b7cbccf1174da839ff718d5dcd594983b44939057`)
  e `output/html/margo-v0.0.1-optimistic-dark.html` (334.698 bytes,
  SHA-256 `e66bb703b7c11583b0d36daf07503d38428480941cda766158d850f9733d66ac`).
  Ambos não deixaram `.margo-render-*` temporário; o HTML dark foi aberto no
  painel Codex para inspeção humana.
- O renderer agora é fonte versionada para HTML, mas a impressão PDF e a
  validação visual/browser continuam pertencendo ao harness M0 e devem consumir
  o ambiente verificado. O helper externo
  `/private/tmp/margo-optimistic-generator/render-artifacts.mjs` permanece
  somente uma fonte histórica não versionada para o pipeline PDF/browser.
- T6 continua deliberadamente deferred por decisão explícita do usuário;
  `release/table-handoff.json`, I1a/I1b e os sucessores formais não foram
  inventados nem promovidos neste checkpoint.
- Checkpoint versionado: commit `914be1508c3f9bfc191241b430ad8da40f91f7cd`,
  tree `ed826990bd44fa881024529745f6f3709641c5f9`, enviado para
  `origin/impl/v0.0.1-core`; a árvore ficou limpa após o push.

### 2026-08-08 — disclosure Mermaid e quebra de página de blocos protegidos

- `2026-08-08T04:53:34-03:00`: o caso RED foi registrado em
  `render_test.go`: a fonte Mermaid não deveria começar aberta no HTML. O
  GREEN removeu `open` de `renderRuntimeFence`, preservando a fonte como um
  disclosure acessível (`details/summary`) que o leitor pode abrir sob
  demanda.
- A preparação de impressão em `standalone.go` agora abre os disclosures
  Mermaid apenas durante `beforeprint`/`margoPreparePrintTOC`, marca blocos
  protegidos que atravessam uma fronteira de página com
  `data-margo-print-break-before="page"`, e restaura estado e marcadores em
  `afterprint`/`margoRestorePrintState`. `assets/document.css` traduz o
  marcador para `break-before: page` e `page-break-before: always`, mantendo
  listas, tabelas, figuras, código, disclosures e Mermaid legíveis como
  unidades de impressão.
- Os testes estáticos e unitários passaram com
  `GOWORK=off GOFLAGS=-mod=readonly`; `go test ./...`, `go vet ./...`,
  `charts/go test ./...`, `pdf/go test ./...` e `git diff --check` também
  passaram. O spec Playwright acrescenta os casos de disclosure
  print-only/restore e bloco que cruza a página.
- Prova focada no Chromium pinado M0 (sem rede/download e sem mutar fonte),
  com `assets/document.css` e `assets/standalone.css` reais embutidos:
  `before={open:false,marker:null}`, `prepared={open:true,marker:page,
  breakBefore:page}`, `restored={open:false,marker:null}`. Isso é evidência
  focada do contrato, não aceite formal M0; o runner completo permanece
  indisponível neste checkout porque o cache/receipt verificado anterior
  aponta para um harness temporário já removido.
- O smoke do renderer versionado foi regenerado após a mudança:
  `output/html/margo-v0.0.1-optimistic.html` (337.382 bytes,
  SHA-256 `d8301779ab49798399b3438c342116a3f25367e96cda2e9f9cd5ae04c175ac18`)
  e `output/html/margo-v0.0.1-optimistic-dark.html` (337.385 bytes,
  SHA-256 `12b8438c4dac8385f80b258a4de362d360d717f1af899f78047a40fcd5174800`).
  Ambos mantêm a fonte Mermaid fechada no HTML; a alteração ainda não foi
  promovida como nova evidência PDF formal.
- Arquivos alterados neste checkpoint: `render.go`, `render_test.go`,
  `standalone.go`, `standalone_test.go`, `assets/document.css` e
  `test/browser/specs/standalone-pagination.spec.mjs`. T6, `release/table-handoff.json`,
  I1a/I1b e os sucessores formais continuam deliberadamente pendentes.
- Checkpoint de código: commit `20de90cccff2fb16408d1df3ef556ee679c8b3f1`,
  tree `24ba7cb06a9e2e754d5d755ecc204bf8114fc30d`, enviado para
  `origin/impl/v0.0.1-core`; o commit documental deste registro será criado
  em seguida.

### 2026-08-08 — PDFs Mermaid regenerados e prova de impressão tagged

- `2026-08-08T05:02:39-03:00`: a prova focada usou o Chromium pinado
  `136.0.7103.25` em
  `/private/tmp/margo-m0-generated.adPbmQ/.cache/playwright/darwin-arm64/1169/chrome-mac/Chromium.app/Contents/MacOS/Chromium`, sem rede externa. O runtime Mermaid local carregou os três diagramas do benchmark, com `3/3` sucessos: flowchart SHA-256
  `6ce2d8cb22b882a18cad0a7b45203ded625f7203da68e2a4a2c5437196b6ffba`,
  sequence SHA-256
  `25f6bad267d86185dbb02225376eda02485292e73a324e1f1d2e3fc5e5c5ed21` e
  readiness-flow SHA-256
  `0b97e9a13a705e9a013bba40cbd5c1ac62d16ac2b3179714940e1bb84a88c807`.
- O HTML permanece com as três fontes Mermaid fechadas na tela; durante a
  preparação de impressão os três disclosures são abertos, os blocos
  protegidos que cruzam uma fronteira recebem `break-before: page`, e o estado
  original é restaurado depois. A prova registrou TOC em uma coluna, `3`
  marcadores de quebra, zero requisições bloqueadas e zero erros de console.
- As margens de impressão seguem o fundo do modo: claro usa
  `rgb(255, 255, 255)` para página, margem e chrome; dark usa
  `oklch(0.145 0 0)` para os três. As evidências JSON são
  `output/review/margo-optimistic-browser-evidence.json` (SHA-256
  `217e7280edc45289a3c25f731986178035a70c0037d8c0ad8cc258eca7ae66aa`) e
  `output/review/margo-optimistic-dark-browser-evidence.json` (SHA-256
  `98462c22d5a4ca1e9f909020d65d302c06fa578b0e4d6f2fb2cda184965ad50c`).
- PDFs regenerados com `tagged: true`, outline, A4, sem JavaScript e sem
  encriptação: `output/pdf/margo-v0.0.1-optimistic.pdf` tem 604.078 bytes,
  21 páginas, SHA-256
  `6f25c552fb6492cc9fe61659bb35f150b8e184df72583939d2b4f9c4c75d7810`;
  `output/pdf/margo-v0.0.1-optimistic-dark.pdf` tem 610.333 bytes, 21 páginas,
  SHA-256 `79cb9e72d8bd31520fd7db2099ff08303529d21ea9764abafedf8fdbacbcfd49`.
  `pdftotext` confirmou Contents, edge cases, Mermaid source e Human
  acceptance record.
- HTMLs correspondentes permanecem:
  `output/html/margo-v0.0.1-optimistic.html` (SHA-256
  `d8301779ab49798399b3438c342116a3f25367e96cda2e9f9cd5ae04c175ac18`) e
  `output/html/margo-v0.0.1-optimistic-dark.html` (SHA-256
  `12b8438c4dac8385f80b258a4de362d360d717f1af899f78047a40fcd5174800`). O
  HTML dark foi aberto no painel local do Codex; a navegação direta `file://`
  do navegador embutido foi recusada pela política de segurança.
- Esta é uma evidência focada do contrato de renderização e impressão, não
  aceite formal M0/I1a/I1b. O runner completo/cache verificável continua
  indisponível neste checkout; T6 e `release/table-handoff.json` continuam
  deliberadamente deferred. O PDF é gerado pelo harness de browser/M0, não pelo
  `tools/optimistic-renderer` isolado.

### 2026-08-08 — chrome de impressão alinhado ao fundo do modo

- RED: o CSS standalone escondia `goshtoso-document__header` e
  `goshtoso-document__footer` em `@media print`, embora o contrato exigisse
  logo, cabeçalho, rodapé e watermark também no PDF. Isso fazia o PDF parecer
  ter uma margem/cromado de outra superfície, especialmente no dark.
- GREEN: o print CSS agora mantém header/footer visíveis, aplica os tokens
  `--margo-print-chrome-background`, `--margo-print-chrome-foreground` e
  `--margo-print-chrome-outline`, usa `print-color-adjust: exact` e impede
  fragmentação interna. Os testes Go e o spec browser verificam display,
  fundo, cor e bordas nos modos claro/escuro.
- Prova focada no Chromium pinado `136.0.7103.25` confirmou em ambos os modos:
  página, body, documento, TOC, header e footer compartilham a mesma
  superfície; `3/3` diagramas Mermaid renderizam; TOC permanece em uma coluna;
  há `5` marcadores de quebra protegida; zero requests bloqueadas e zero erros
  de console. O claro usa `rgb(255, 255, 255)` na superfície; o dark usa
  `oklch(0.145 0 0)`. A área cinza/preta fora da folha que aparece no Preview
  é chrome do visualizador, não margem desenhada no PDF.
- Artefatos atuais: HTML claro
  `output/html/margo-v0.0.1-optimistic.html` (337.780 bytes,
  SHA-256 `532306807949e077baed9adc4b04df41690719e19aa7a3886ca62cc605955034`);
  HTML dark `output/html/margo-v0.0.1-optimistic-dark.html` (337.783 bytes,
  SHA-256 `8580e43c3f35a7d33727f837e70a4b68912bf0a5cfabf9b5838b5493bef262e7`);
  PDF claro A4 tagged com 22 páginas e 611.191 bytes, SHA-256
  `c42f5605d1e8ff4ac139a5ab43e888e07ef2ca1fe7791d385688bf5f01b4150f`;
  PDF dark A4 tagged com 22 páginas e 617.551 bytes, SHA-256
  `972f2f670e151118d41a42d16d985e10f15fa9197a3bc0a762c43e461ae53377`.
  Evidências JSON: claro `08ae561bfd02f512e0037a54a05278d94e13ad20a01582a47fa705d0a8f27e28`;
  dark `a014579be31677e99b0215becca14a61e10da55dce8f63f10dd18550e3fd2989`.
- Checkpoint de código: commit `6a585c00daa7a4cbbd13f780df3567fde1f78f04`,
  tree `1eadbe3c1911f4e019d0a5b4962863679aa96206`, enviado para
  `origin/impl/v0.0.1-core`. O trabalho segue limpo após o push; a prova é
  focada e não substitui a aceitação formal M0/I1a/I1b.

### 2026-08-08 — margem de página e superfície dark corrigidas

- RED: a margem CSS anterior era `20mm 18mm 22mm`; no PDF dark o texto
  começava exatamente no início dessa área útil e a área externa da folha não
  recebia explicitamente o mesmo fundo. O heading parecia tocar a borda e a
  diferença entre a superfície da página e o chrome do Preview ficava mais
  evidente no dark.
- GREEN: `@page` agora usa `margin: 24mm 22mm 26mm` e
  `background: var(--margo-print-page-background)`. Isso mantém o fundo do
  modo também na área de margem e cria espaço de leitura real para headings,
  parágrafos, listas e blocos longos. O teste
  `TestStandalonePrintPageLeavesReadableBreathingRoom` fixa tamanho A4,
  margem e background no CSS embutido.
- A prova focada regenerou os dois PDFs em A4. O claro tem 23 páginas e
  615.640 bytes, SHA-256
  `197d6aeda33f8d300abea19fc6ce894fe727a9ff83053e89e22febb89c4b949d`;
  o dark tem 23 páginas e 620.310 bytes, SHA-256
  `c33cc54a0c2d774426181ad9de4a7508fb54393b09728f8f78cc08fdda34d987`.
  Na página 4, o heading passa a iniciar aproximadamente em 22mm da borda
  lateral e 24mm do topo, sem encostar no limite visual.
- HTMLs regenerados: claro
  `output/html/margo-v0.0.1-optimistic.html`, SHA-256
  `fa6f4ba9cd24840e634385777825df8752f572ec666f13016bc4aa7d7cd6dbb2`;
  dark `output/html/margo-v0.0.1-optimistic-dark.html`, SHA-256
  `3c11729637f484f4cb25e421ce18e109726348af26e4ec16cdad7c0bb8ab031c`.
  Evidências JSON: claro `1494009d19540bf357d7f865dbcb76d1336e9fe4a1b4bbf6f3109c523b181278`;
  dark `5a43cf5cdf036f66907ee9e98e6affa45b9918461448ec06695f6cc9cd798c09`.
- Os gates `go test .`, `go test ./...`, `go vet ./...`, `charts/go test
  ./...`, `pdf/go test ./...` e `git diff --check` passaram com
  `GOWORK=off GOFLAGS=-mod=readonly`. Checkpoint de código:
  commit `ac3877c488ef64922ce2665c485d878c554e258c`, tree
  `305cdde968d0aa65d5d6dc1b1aa8d741b25a3b80`, enviado para
  `origin/impl/v0.0.1-core`. A evidência continua focada, não é aceite formal
  M0/I1a/I1b; T6 e `release/table-handoff.json` seguem deferred.

### 2026-08-08 — auditoria visual pós-margem

- `2026-08-08T05:28:40-03:00`: os artefatos atuais foram re-hashados no
  worktree limpo. O HTML claro tem 337.830 bytes e SHA-256
  `fa6f4ba9cd24840e634385777825df8752f572ec666f13016bc4aa7d7cd6dbb2`; o
  HTML dark tem 337.833 bytes e SHA-256
  `3c11729637f484f4cb25e421ce18e109726348af26e4ec16cdad7c0bb8ab031c`.
  Os PDFs A4 correspondentes têm 23 páginas: claro 615.640 bytes,
  `197d6aeda33f8d300abea19fc6ce894fe727a9ff83053e89e22febb89c4b949d`, e
  dark 620.310 bytes,
  `c33cc54a0c2d774426181ad9de4a7508fb54393b09728f8f78cc08fdda34d987`.
- A rasterização independente da página 4 dark confirmou que headings e prosa
  começam dentro da margem respirável, sem tocar as bordas. A página 1 mantém
  TOC em uma coluna porque ela cabe no espaço vertical; a coluna dupla continua
  apenas como fallback responsivo de impressão. `pdfinfo` confirmou A4,
  tagged, sem JavaScript e sem criptografia.
- Nenhum novo defeito de renderização foi aberto nesta auditoria. O branch
  permanece limpo e alinhado ao remoto em `145416fc82c8f0a0ca390ab8415f5af880f4a335`,
  tree `827a16f8977801736fd486b11f5b870e71dffbab`. M0 formal, I1a/I1b,
  sucessores H/P/D/O e T6 continuam pendentes por suas dependências/autoridade;
  `release/table-handoff.json` não foi inventado.

### 2026-08-08 — PDF dark regenerado após comentário de margem

- O comentário apontava para uma versão anterior do PDF. Os PDFs foram
  regenerados a partir dos HTMLs atuais com Chromium pinado
  `136.0.7103.25`, sem requests bloqueadas, erros de console ou erros de
  página. O PDF claro atual tem 23 páginas e SHA-256
  `0bbf4c2fc2735b46411933c46c8feb750bffd88dab2691be0bf37c5f5d4701ec`;
  o PDF dark atual tem 23 páginas e SHA-256
  `a7e78d7900e73080e14288877b8ad29a7fb0745eb30a67dc54eaf46fd96066d9`.
- A medição `pdftotext -bbox` da página 4 dark coloca o primeiro heading em
  `xMin=62.25pt` e `yMin=71.70pt`, coerente com `@page` de 22mm lateral e
  24mm superior. Portanto o texto não toca a borda no artefato atual; a
  captura revisada era de um PDF anterior à correção de margem.
- O HTML e o PDF permanecem artefatos de trabalho ignorados pelo Git. O
  checkpoint documental deste registro é separado do checkpoint funcional;
  nenhuma aceitação formal M0/I1a/I1b ou T6 foi inferida.

### 2026-08-08 — auditoria readonly de fechamento

- Os gates locais passaram com `GOWORK=off GOFLAGS=-mod=readonly`: `go test
  ./... -count=1`, `go vet ./...`, `charts/go test ./...`, `pdf/go test
  ./...`, `cmd/margo/go test ./...` e a suíte focada de standalone/paginação.
- O CLI `tools/optimistic-renderer` regenerou os dois HTMLs em diretório
  temporário com os mesmos bytes dos artefatos versionados: light 337.830
  bytes (`fa6f4ba9cd24840e634385777825df8752f572ec666f13016bc4aa7d7cd6dbb2`)
  e dark 337.833 bytes
  (`3c11729637f484f4cb25e421ce18e109726348af26e4ec16cdad7c0bb8ab031c`).
- A tentativa de repetir `node --test test/browser/lint-contrast.test.mjs`
  diretamente falhou antes dos testes porque este checkout não contém o
  `node_modules` do M0; isso confirma que o comando bare Node não é evidência
  formal. A prova anterior via Playwright absoluto permanece candidata, e o
  runner M0/receipt deve ser restaurado pelo predecessor formal antes de uma
  aceitação independente.
- Nenhum arquivo de módulo foi alterado nesta auditoria. T6 continua no fim
  do backlog por decisão explícita; I1a/I1b e os sucessores formais seguem
  pendentes.

### 2026-08-08 — M0 reprovisionado e correções visuais pós-review

- O RED do instalador M0 foi reproduzido com um ZIP Chromium local mínimo:
  quando o `--receipt` ficava dentro da raiz de extração ainda inexistente,
  `install-browser.sh` criava essa raiz antes do teste `-d` e pulava a
  extração, retornando `margo.browser_executable_missing`. O teste de contrato
  é `test/browser/harness/install-browser-contract.mjs` e não usa rede.
- O GREEN move a criação do diretório do receipt para depois da extração e da
  validação do executável. Checkpoint publicado: commit `331c320`, enviado
  para `origin/impl/v0.0.1-core`.
- A sequência POSIX M0 foi executada neste worktree: receipt Chromium
  `test/browser/.cache/playwright/darwin-arm64/1169/browser-receipt.json`,
  SHA-256 `f50de873dc047443cb96206760448b3c56892f086a5eee915fea8bdf7e8679bd`;
  checked env `test/browser/.cache/node-env.checked.sh`, SHA-256
  `1591fb686c189f5fb38cdf4f9c31731cc1b8e508594cea1d5e8381ec4bdaaecd`;
  npm cache receipt `test/browser/.cache/npm/v11.17.0/Darwin-arm64/receipt.json`,
  SHA-256 `46e043e0dbaa8e28de6bfd0a94c2f1dac2bf82bea6ebcabe9b530e8c7e63cff9`.
  O runner validou Node `v26.5.0`, npm `11.17.0`, Chromium revision `1169`,
  versão `136.0.7103.25`, cache imutável e `network=0`.
- O conjunto M0 executado pelo runner checked passou `14/14`: cinco testes
  `@margo-harness`, contraste claro/escuro, oito testes de paginação/shell,
  além do contrato local do instalador. Os gates readonly Go (`go test ./...`,
  `go vet ./...`, `charts`, `pdf` e `cmd/margo`) também passaram. Esta é uma
  evidência candidata local, não aceite formal independente M0/I1a/I1b.
- O mesmo checkpoint corrige o comentário visual de blocos divididos: a
  preparação de impressão aplica `break-before: page` inline nos blocos
  protegidos e restaura o estilo original após o print. Corrige também o
  stamp dark, que deixava de ser transparente e passa a usar
  `--color-surface-dark-alt`. Checkpoint publicado: commit `d9d2ede`, tree
  `2ce4f48295ad9a23fa68f84bb9b1a0d1ff2224a4`.
- HTMLs regenerados a partir de `testdata/markdown/margo-full-feature-set.md`:
  [light](output/html/margo-v0.0.1-optimistic.html), 338.448 bytes, SHA-256
  `e367c35ce9a2701ae8b18f3a33c12af09bf81cac4af7505115bcbb99d59d9da0`, e
  [dark](output/html/margo-v0.0.1-optimistic-dark.html), 338.451 bytes, SHA-256
  `24eb3b4bbf8d691914c094821ecbcf9d60e19a97ecd90bd06504f1698a80aa04`.
- PDFs A4 tagged regenerados com três diagramas Mermaid renderizados e sem
  requests externas: [light](output/pdf/margo-v0.0.1-optimistic.pdf), 23
  páginas, 615.705 bytes, SHA-256
  `f5610e45854803b769b5a886c09bfb40dae8ec5ba829d59da71a1d2bd3716a5c`; e
  [dark](output/pdf/margo-v0.0.1-optimistic-dark.pdf), 23 páginas, 620.376
  bytes, SHA-256
  `278e3e85dcd44491c2195ac9f8403a6877ad0b76d962164100384060acd4aefa`.
  A página 4 dark rasterizada inicia o conteúdo em margem respirável; a página
  1 mantém TOC em uma coluna porque cabe verticalmente, usando duas colunas
  somente como fallback de altura.
- Os diretórios gerados `node_modules` e `test-results` foram preservados fora
  do worktree em `/tmp/margo-m0-test-artifacts-d9d2ede`; o cache M0 permanece
  em `test/browser/.cache` como artefato efêmero não versionado. T6,
  `release/table-handoff.json`, I1a/I1b e os sucessores formais continuam
  deliberadamente pendentes.

### 2026-08-08 — M0 checked runner reexecutado

- Com 138 GiB livres, o comando checked foi reexecutado sem alterar código:
  `./test/browser/run-playwright.sh --check --env-file
  "$PWD/test/browser/.cache/node-env.checked.sh" --grep
  '@margo-harness|@contrast|@pagination|@shell'`.
- Resultado: `14 passed (3.9s)`; receipt reportou Node `v26.5.0`, npm
  `11.17.0`, Chromium revision `1169`, versão `136.0.7103.25` e
  `network=0`. O `npm ci` consumiu o cache local e instalou oito pacotes;
  nenhuma resolução remota foi usada.
- `node_modules` e `test-results` foram movidos, de forma recuperável, para
  `/tmp/margo-m0-test-artifacts-0616e7e`; o cache checked permanece em
  `test/browser/.cache` como artefato efêmero não versionado. O worktree tem
  apenas esse cache não rastreado; nenhuma fonte de módulo foi alterada.
- Evidência continua candidata local, não aceite formal independente M0.
  T6, `release/table-handoff.json`, I1a/I1b e H/P/D/O continuam fora da
  fronteira executável até o handoff/proveniência correspondente.

### 2026-08-08 — readonly Go gates no HEAD atual

- No HEAD `3dcf52f` os gates passaram sem alteração de módulo:
  `GOWORK=off GOFLAGS=-mod=readonly go test ./... -count=1`, `go vet ./...`,
  `charts/go test ./... -count=1`, `pdf/go test ./... -count=1` e
  `cmd/margo/go test ./... -count=1`.
- Root, `charts`, `pdf` e `cmd/margo` permanecem compiláveis; `deck` continua
  pacote estático sem testes. Isso confirma o estado candidato atual, mas não
  substitui o proxy/handoff I1a/I1b nem a revisão independente.

### 2026-08-08 — identidade do checkpoint funcional atual

- O checkpoint de implementação que contém a correção de paginação é o commit
  `4d1234560d5390b252abdd960546f5268628b73e`, tree
  `b4829d50301e39e24cac2ce3cfa06ff507cca2c7`, já enviado para
  `origin/impl/v0.0.1-core`. O próximo commit, se houver, será apenas
  documental e não deve ser confundido com mudança de runtime.

### 2026-08-08 — paginação: heading junto da lista protegida

- O comentário visual da página 5 dark foi reproduzido: o heading `Ordered,
  restarted, and mixed lists` ficava no fim da página, enquanto o `<ol>` era
  movido sozinho para a página seguinte por `break-inside: avoid-page`.
- O script `standalonePrintPaginationScript` agora identifica pares de
  heading direto + próximo bloco protegido, decide primeiro a quebra do par e
  só aplica `break-before: page` ao bloco quando não há heading associado a
  promover. Os marcadores e estilos inline originais continuam sendo
  restaurados por `margoRestorePrintState`.
- RED: o primeiro teste revelou que manter os dois marcadores ainda deixava o
  heading isolado. GREEN: a ordem heading-first passou a manter o heading e o
  `<ol>` na mesma página; o teste browser verifica o mesmo índice de página e
  a restauração dos dois elementos.
- Gates: `GOWORK=off GOFLAGS=-mod=readonly go test ./... -count=1` passou;
  runner M0 checked com
  `./test/browser/run-playwright.sh --check --env-file
  "$PWD/test/browser/.cache/node-env.checked.sh" --grep '@pagination'`
  passou `7/7`; a rodada M0 completa com
  `--grep '@margo-harness|@contrast|@pagination|@shell'` passou `15/15`,
  com Node `v26.5.0`, Chromium `136.0.7103.25` e `network=0`.
- HTMLs regenerados: light 340.055 bytes,
  SHA-256 `328d9141fcb03ca509755c03f9e1918b6bebffe6c95b8a30564d3a9c57f37543`;
  dark 340.058 bytes,
  SHA-256 `441e5cd46b226bac5294a4d92040b8a178f0741af9f5fdba56ff75c3e8b4dbe8`.
- PDFs A4 tagged regenerados sem requests bloqueadas, erros de console ou
  erros de página: light 23 páginas, SHA-256
  `fda87b63db2d750f79db5d703ea152041dc9f0b085776ebc307d7bfb116d8dc1`;
  dark 23 páginas, SHA-256
  `6445213f57944901cd368fe370a49a889fc53515baecc8b975fa5be23ffcbe21`.
  A inspeção raster da página 6 dark mostra o heading seguido integralmente
  pela lista ordenada; o espaço restante na página 5 é intencional para não
  fragmentar o bloco protegido.
- Alterados: `standalone.go`, `standalone_test.go`,
  `test/browser/specs/standalone-pagination.spec.mjs` e este registro.
  `node_modules`/`test-results` do runner foram preservados em
  `/tmp/margo-m0-test-artifacts-heading-full-fM2J0K`; apenas
  `test/browser/.cache` permanece como cache M0 não versionado. Nenhuma
  aceitação formal I1a/I1b/T6 foi inferida.

### 2026-08-08 — auditoria da captura de página 7

- A captura anotada como página 7 mostra apenas o watermark, mas não coincide
  com o artefato atual regenerado após `4d12345`: `pdftotext` da página 7
  contém `Loose list with paragraphs`, seus itens e o bloco de código. A
  rasterização atual está em `/tmp/margo-pdf-check-heading-current/page-07.png`.
- Varredura determinística das 23 páginas dark, removendo somente o footer
  `OPTIMISTIC BENCHMARK`, encontrou `blank_after_footer_removal=0`; os tamanhos
  de texto das páginas 6, 7 e 8 são respectivamente 733, 409 e 1011 bytes.
- O espaço inferior da página 7 é consequência do contrato deliberado de
  manter a lista com parágrafos e seu código no mesmo bloco. Relaxar
  `break-inside: avoid-page` para preencher espaço criaria exatamente a
  fragmentação que a revisão pediu para evitar. Nenhuma mudança de runtime foi
  necessária nesta captura porque o estado atual já não contém a página vazia.
- O lint checked de contraste também passou para ambos os modos: `500` nós
  auditados por modo, `0` falhas, `0` requests bloqueadas, Node `v26.5.0` e
  Chromium `136.0.7103.25`.

### 2026-08-08 — tabelas longas no print e auditoria da página 10

- O comentário visual de página 10 dark foi comparado com uma regeneração
  atual. A captura anotada era de um estado anterior; o PDF autoritativo atual
  mostra as duas tabelas completas, com cabeçalhos, linhas e células dentro da
  área imprimível. A rasterização usada para a conferência está em
  `/tmp/margo-pdf-check-table-current-4lILxk/page-10.png` e a extração
  `pdftotext -layout` conserva toda a tabela.
- Foi adicionado um caso RED/GREEN em
  `test/browser/specs/standalone-pagination.spec.mjs`: uma célula longa com
  `.whitespace-nowrap` estourava a tabela para `tableRight=2125.703125`,
  enquanto a área do artigo terminava em `520.5`. A correção em
  `assets/document.css` só no contexto de impressão limita wrapper e tabela a
  `100%`, usa `table-layout: fixed` e permite quebra segura em `th`/`td` com
  `white-space: normal`, `overflow-wrap: anywhere` e `word-break: break-word`.
- O teste focado passou e a rodada M0 checked passou `16/16`, incluindo
  harness, contraste, paginação e shell, com Node `v26.5.0`, npm `11.17.0`,
  Chromium revision `1169`/versão `136.0.7103.25` e `network=0`. O lint de
  contraste passou em light e dark com `500` nós por modo e `0` falhas.
- `GOWORK=off GOFLAGS=-mod=readonly go test ./... -count=1` passou. Os HTMLs
  foram regenerados: light 340.514 bytes, SHA-256
  `daddf32bcbcc17580469597a575a4f8c5289276ec69eaffcc259efd9cc6c3d06`; dark
  340.517 bytes, SHA-256
  `eab6fec0769969699fc0a0e3e4404941b05457750e9abe4c8780aac55869e0a5`.
  Os PDFs A4 tagged têm 23 páginas: light SHA-256
  `bc20989a5dc68fd15d36c389baf9634a8f3c95ef4673e4eafa5575746e234730`; dark
  SHA-256 `163dde4500f28518270cdff89fa0417a795c6c0865b9c675061d3aa5a8846e53`.
  As evidências do render não registraram erros de console, erros de página ou
  requests bloqueadas.
- Os artefatos temporários `node_modules`/`test-results` do runner foram
  movidos de forma recuperável para
  `/tmp/margo-m0-test-artifacts-table-current-a4R2b2`; somente
  `test/browser/.cache` permanece como cache M0 intencional e não versionado.
  Este checkpoint continua candidato local: não infere aceitação formal de
  I1a/I1b/T6 nem altera a posição de `release/table-handoff.json` no backlog.

### 2026-08-08 — O1: spool privado antes da publicação

- O próximo predecessor local escolhido foi O1, a fronteira comum para CLI,
  PDF e demais saídas: nenhum sink deve tornar bytes visíveis antes de a
  compilação, runtime e relatório estarem validados. O RED inicial confirmou a
  ausência de `Spool`, `ArtifactSink`, `CommitOutcome` e `CommitResult`.
- O GREEN adicionou `spool.go` e `artifact_sink.go`. `Spool` mantém bytes em
  memória até `MemoryLimit`, cruza para arquivo temporário privado `0600`,
  impõe `MaximumBytes` sem anexar o chunk que estoura, calcula
  `ArtifactDigest` dos bytes exatos e oferece replay por `Reader()`. `Close()`
  é idempotente e remove o estágio privado; cancelamento é observado antes de
  criar ou mutar o estágio.
- `CommitOutcome` agora preserva os quatro estados necessários para sinks
  futuros: `not_committed`, `committed`, `durability_uncertain` e `unknown`.
  `ArtifactSink` recebe contexto, `io.Reader` e digest esperado, e devolve
  `CommitResult` com alvo, bytes e identidade do artefato sem prometer
  atomicidade antecipadamente.
- Testes O1: cruzamento de limiar/permissão `0600`, overflow sem mutação,
  cancelamento com limpeza, replay/digest e contrato de sink. Gates passaram:
  `GOWORK=off GOFLAGS=-mod=readonly go test . -run
  'Test(Spool|CommitOutcome|ArtifactSink)' -count=1`, race focado `TestSpool`
  `-count=20`, suíte root `go test ./... -count=1`, `go vet ./...` e
  `go test -race ./... -count=1`.
- Arquivos deste checkpoint: `spool.go`, `spool_test.go`,
  `artifact_sink.go`, `artifact_sink_test.go` e este registro. O1 é candidato
  local; O2/O3/O4, I3 e a aceitação independente ainda não foram inferidos.
  O handoff T6/I1a/I1b continua inalterado e Goshtoso não foi editado nem
  integrado por este checkpoint.

### 2026-08-08 — O2: publicação atômica no-replace

- O RED de O2 foi reproduzido com os quatro testes de no-replace e o teste de
  operações Unix: antes da implementação, `AtomicFileSink` e
  `defaultAtomicOps` falharam na compilação (`undefined`). O teste foi mantido
  como contrato de pre-publicação, digest e preservação do destino existente.
- O GREEN adicionou `atomic_file_sink.go` e `atomic_unix.go`. O sink calcula o
  digest enquanto escreve um arquivo temporário no mesmo diretório, força
  modo `0600`, faz `Sync`/`Close`, recusa `Force` neste predecessor, verifica o
  digest esperado e só então usa um hard link Unix atômico no-replace como
  ponto de visibilidade. O estágio é removido após a publicação ou em toda
  falha pré-linearização; leitura posterior e sincronização do diretório
  classificam `not_committed`, `committed`, `durability_uncertain` ou
  `unknown` sem prometer rollback.
- A cobertura inclui erro de leitura, cancelamento antes da publicação,
  mismatch de digest, destino preexistente sem sobrescrita, limpeza do estágio
  privado, modo `0600`, bytes/digest publicados e repetição com race detector.
  Gates passaram: `GOWORK=off GOFLAGS=-mod=readonly go test . -run
  'TestAtomic(NoReplace|Unix)' -count=1`,
  `GOWORK=off GOFLAGS=-mod=readonly go test -race . -run 'TestAtomic'
  -count=20`, `go test ./... -count=1`, `go vet ./...` e
  `go test -race ./... -count=1`.
- Arquivos deste checkpoint: `atomic_file_sink.go`, `atomic_unix.go`,
  `atomic_file_sink_test.go`, `atomic_unix_test.go` e este registro. O2 é
  candidato local publicado no branch de implementação; O3 ainda é o dono
  serial da extensão `--force`, Windows e da classificação pós-linearização.
  Nenhuma aceitação formal Goshtoso/T6/I1a/I1b foi inferida.
- Identidade publicada do checkpoint O2: commit
  `b0131be2c748f8e4d2441a60d7dc32abd95151e8`, tree
  `4c1535036c83561d7c98a4da3dde383d7a4a9314`, remoto
  `origin/impl/v0.0.1-core`. O único estado não rastreado intencional é
  `test/browser/.cache/`, usado pelo recibo M0; não há arquivos staged ou
  modificados fora desse cache.

### 2026-08-08 — O3: force, Windows e classificação pós-visibilidade

- O RED de O3 adicionou os casos de `Force`, falha de sincronização do pai,
  erro ambíguo do primitivo e cancelamento depois da publicação. Antes da
  implementação, a interface não possuía `publishReplace`, produzindo falha
  de compilação no teste O3.
- O GREEN transferiu a máquina de estados para o sink compartilhado: o caminho
  padrão continua no-replace; `Force` usa replace atômico; o resultado de
  publicação informa se os novos bytes ficaram visíveis; read-back contra o
  snapshot anterior e o digest esperado evita converter erro ambíguo em
  `not_committed`. Cancelamento é adiado durante publicação, read-back e sync,
  retornando o outcome real junto com `context.Canceled`. Unix usa hard-link
  no-replace e `os.Rename` no replace; Windows usa `MoveFileEx` com
  `MOVEFILE_WRITE_THROUGH` e `MOVEFILE_REPLACE_EXISTING` para o caminho force.
- Gates: `GOWORK=off GOFLAGS=-mod=readonly go test . -run
  'TestAtomic(Force|PostLinearization|Cancellation|Windows)' -count=1`, a
  matriz O2/O3 focada com `-count=20`, `go test ./... -count=1`, `go vet
  ./...`, `go test -race ./... -count=1`, e compilação cruzada Windows com
  `GOOS=windows GOARCH=amd64 go test -c` passaram. O3 é candidato local;
  aceitação independente do sink e os sucessores O4/O5 ainda não foram
  inferidos.
- Arquivos deste checkpoint: `atomic_file_sink.go`, `atomic_unix.go`,
  `atomic_windows.go`, `atomic_file_sink_test.go`, `atomic_windows_test.go` e
  este registro. A pequena transferência serial de `atomic_unix.go` adiciona
  apenas o método replace necessário ao estado compartilhado; nenhuma fonte
  Goshtoso foi alterada e T6/I1a/I1b continuam pendentes.
- Identidade publicada do checkpoint O3: commit
  `b1c3281bf6e8181f4e05e62276a26c95ad17ebf1`, tree
  `15debf6efee03e80e3408f2a489c168458036ab7`, remoto
  `origin/impl/v0.0.1-core`. O cache M0 `test/browser/.cache/` segue como o
  único não rastreado intencional.

### 2026-08-08 — O4: stdout somente depois do spool validado

- O RED de O4 confirmou que `StdoutSink` ainda não existia. O contrato cobre
  zero bytes em falha de leitura, digest divergente e cancelamento antes da
  publicação; a saída só é tocada depois que o leitor inteiro foi consumido
  para um `Spool` privado e o digest esperado foi verificado.
- O GREEN adicionou `stdout_sink.go`: a fase de validação usa o mesmo limite de
  documento e limpeza privada do O1; a fase de cópia para o writer é separada
  e nunca promete atomicidade. Short write ou erro depois de um prefixo visível
  retorna `CommitUnknown`, bytes observados e diagnóstico
  `margo.stdout.partial_write`; sucesso completo retorna `CommitCommitted`.
- Testes cobrem erro antes do primeiro byte, publicação exata, mismatch sem
  saída, short write com prefixo observado e cancelamento pré-commit. Gates
  passaram: `GOWORK=off GOFLAGS=-mod=readonly go test . -run 'TestStdout'
  -count=20`, race focado `TestStdout -count=20`, `go test ./... -count=1`,
  `go vet ./...` e `go test -race ./... -count=1`.
- Arquivos deste checkpoint: `stdout_sink.go`, `stdout_sink_test.go` e este
  registro. O4 é candidato local; O5 (CLI/render/publicação) ainda depende da
  revisão dos sinks e das autoridades de runtime. Goshtoso/T6/I1a/I1b seguem
  sem aceitação formal.
