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
- Último HEAD de código da implementação: `6e0fc7ee06fdd641f2141b31440dbb6c9da7e40a`,
  tree `c916ad92bee97edd6e665c0bd12e907e1e18c50c`. Os commits posteriores
  `cd80dfe`, `aa3eaae`, `3a74358` e este checkpoint alteram somente este
  `GOAL.md`; a árvore de código permanece a mesma e o branch segue limpo.

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

- Auditoria de continuidade: o worktree Margo `impl/v0.0.1-core` e o worktree
  T6 `codex/margo-v001-t6` estão limpos; não existe `release/table-handoff.json`
  nem diretório `internal/releasehandoff` no predecessor externo. O RED do T6
  permanece reproduzível e a fronteira externa continua sem mutação.
- Artefatos HTML verificados e apresentados ao usuário: o preview semântico em
  `/tmp/margo-semantic-preview/margo-semantic-preview.html` tem 3.225 bytes e
  SHA-256 `3ae8855b93dbae1d98c21b1ca5cb11b1d965316237fddf025fb528fe8772d155`;
  o preview de contrato em
  `/tmp/margo-contract-preview/margo-contract-preview.html` tem 3.060 bytes e
  SHA-256 `1ef261dcd62a08642c0405f9be8230ed93bfeb12aa07800ecd29c709fe2bb13d`.
  O helper foi regenerado com `GOWORK=off GOFLAGS=-mod=readonly go run .` e
  agora encapsula o fragmento em um documento HTML válido. Nenhum PDF foi
  produzido nesta etapa; isso não substitui o gate P1-P7.
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

- Bloqueio atual: T6 depende de um handoff externo autorizado e ainda não há
  `release/table-handoff.json` concreto para verificar. O revisor não tem
  autorização para criar tag, publicar módulo ou substituir o dono externo.
- Atenção de reprodutibilidade: o transfer record C0 guarda SHA-256 bruto dos
  arquivos `go.mod`/`go.sum`, mas o comando literal do plano C5 usa
  `git hash-object`, que neste repositório produz SHA-1 de blob e nunca pode
  igualar aqueles campos. O checkpoint C5 usou `shasum -a 256` sobre os bytes
  (os valores continuam `0eb36e99...` e `1c7ae9b8...`); esta divergência do
  texto do plano precisa ser reconciliada antes da aceitação independente do
  gate, sem alterar os módulos ou inventar uma identidade.
- Atenção de invocação: o comando E2E literal do plano, executado da raiz com
  `GOWORK=off`, não alcança o módulo separado `site/`; a prova T5 usou o
  `go.work` temporário do checkout e `-tags='e2e table'`, mantendo
  `GOWORK=off GOFLAGS=-mod=readonly` nos gates root/site que não dependem do
  workspace. O erro da forma literal é um limite de invocação do plano, não
  uma falha do runtime T5.
- Registrar aqui qualquer bloqueio reproduzível antes de alterar a ordem ou o
  contrato do plano.
