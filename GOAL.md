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
- Próximo marco: C2 (diagnósticos, frontmatter e perfil Markdown), mantendo
  C0 read-only e sem alterar `go.mod`, `go.sum` ou o lock.

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
- Registrar aqui qualquer bloqueio reproduzível antes de alterar a ordem ou o
  contrato do plano.
