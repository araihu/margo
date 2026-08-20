# Goal: Margo / Marpit slide decks

> Arquivo de acompanhamento local. Intencionalmente não versionado, não deve ser adicionado ao índice Git.

## Natureza iterativa

Este é um **goal vivo**: ele evolui durante a sessão e deve ser atualizado após cada decisão relevante, implementação, evidência ou rodada de revisão. O documento deve:

- manter o objetivo atual e o próximo resultado verificável;
- registrar decisões e evidências novas sem apagar contexto útil;
- mover itens entre aberto, em progresso, bloqueado e concluído;
- atualizar o loop de revisão e os critérios de saída quando o escopo mudar;
- continuar local e não versionado ao longo de todo o trabalho.

## Contexto

- Worktree: `/private/tmp/margo-marpit-decks-v001`
- Branch: `feat/marpit-decks-v001`
- Data do ciclo: 2026-08-19
- Commit publicado: `335d9892deb937b1516304aa5dd65acb9bd53a90`
- Estado atual: slice v0.0.1 implementado, revisado, commitado e publicado na branch; novo ciclo aberto neste task após handoff da sessão `01a01874-d5e8-74a3-9338-191cc438a0f0`.
- Regra de entrega: preservar o worktree isolado. O commit e o push do slice anterior foram autorizados e concluídos. Qualquer novo commit ou push exige autorização própria. Não criar PR, fazer merge, tag, release ou deploy sem autorização explícita.

## Ciclo ativo após o handoff

### Objetivo atual

Definir e executar o próximo incremento dos decks Margo sobre a baseline
publicada, sem enfraquecer compatibilidade Marpit, determinismo, acessibilidade
ou os contratos de runtime/PDF já aprovados.

### Baseline congelada

- [x] perfil Marpit-compatible v0.0.1 implementado;
- [x] layouts, temas, charts, Mermaid, mídia, notas e HTML standalone;
- [x] geometria 16:9/4:3, overflow, print/PDF e runtime reports;
- [x] comparador r6 com 30 pares Margo/Marpit e revisão reproduzível;
- [x] Impeccable R5 e Checklist Design R5 sem P0/P1;
- [x] suíte Go, vet, módulos, formatação, diff e sintaxe JS verdes;
- [x] commit e branch remota verificados no mesmo SHA;
- [x] nenhum PDF, PPTX, asset ou string TOTVS versionado.

### Decisão de escopo

O próximo incremento escolhido é **Composições R1**. A aprovação explícita desta
sessão autorizou a implementação e a verificação do slice; ela não autoriza
commit, push, PR, merge, tag, release ou deploy.

- [x] **Composições R1:** transformar o vocabulário genérico da referência em
   presets Margo (`content`, `agenda`, `media-split`, `media-stage`, `steps`,
   `highlight`, `compare-grid`, `hero`, `image-grid`) sem copiar layouts ou
   assets da origem.
- [ ] **Dívida de evidência P2:** executar zoom real de navegador a 200%, testes
   VoiceOver/NVDA, corrigir o idioma do placeholder PT-BR e registrar corpus
   e sidecars reproduzíveis.
- [ ] **Proveniência do comparador:** substituir identidade/status sintetizados
   em `make-compare.mjs` por evidência produzida pelo runtime real e validada
   no envelope canônico.
- [ ] **Storytelling de dados R2:** especificar `chart-story`, `data-table`,
   `cycle` e `timeline-fit`; só implementar depois de fechar o contrato de
   composição R1.

### Decisão arquitetural aprovada

Composições R1 terão uma diretiva versionada `composition`, separada de
`class` e `layout/slot`. A diretiva expressa intenção; o catálogo interno
resolve a intenção para layout estrutural, contrato de slots, variante visual,
regras de conteúdo e requisitos de acessibilidade. O DOM receberá
`data-margo-composition` e continuará determinístico.

Entradas v0.0.1 sem `composition` permanecem compatíveis. A superfície R1
acrescenta apenas a diretiva versionada, não aceita HTML/CSS arbitrário, não
copia layouts TOTVS e não importa assets da referência. A spec fixa nomes,
aliases, cardinalidades, fallback, diagnósticos, versão do perfil e comportamento
de print/PDF.

### Próximo resultado verificável

- [x] escolher uma única frente principal;
- [x] registrar contrato, arquivos afetados, compatibilidade e não objetivos em
      `docs/superpowers/specs/2026-08-19-margo-compositions-r1-design.md`;
- [x] revisar e aprovar a spec arquitetural;
- [x] escrever o plano de implementação em
      `docs/superpowers/plans/2026-08-19-margo-compositions-r1.md`;
- [x] escrever testes que falham para o comportamento escolhido;
- [x] implementar o menor incremento que fecha esses testes;
- [x] repetir gates Go, runtime/PDF e review visual proporcionais ao risco;
- [x] atualizar este documento com evidências e pendências;
- [x] parar antes de commit, push, PR, merge, tag, release ou deploy sem nova
      autorização explícita.

### Critério de saída do ciclo ativo

O ciclo fecha quando a frente escolhida tem contrato documentado, testes
reproduzíveis, evidência visual ou de runtime correspondente, nenhuma
regressão P0/P1 e um inventário claro do que permanece fora de escopo. Gates
verdes não autorizam publicação.

### Fronteira de sincronização auxiliar

Há uma linha de trabalho externa inventariando um template TOTVS (PDF/PPTX,
118 slides, 67 layouts, 5 masters e 174 mídias). Essa entrada serve apenas
como vocabulário e referência de composição/layout. Não importar, copiar ou
versionar arquivos, PDFs, PPTX, mídias ou assets TOTVS neste worktree.

O sync final confirmou a regra de generalização: não replicar os 67 layouts.
Para este slice, as famílias de composição ficam organizadas em oito presets
R1 (`content`, `agenda`, `media-split`/`media-stage`, `steps`, `highlight`,
`compare-grid`, `hero`, `image-grid`). O storytelling de dados (`chart-story`,
`data-table`, `cycle`, `timeline-fit`) fica como R2. `theme` continua sendo
tokens/fontes/chrome global; `layout` é geometria/slots; `composition` é
intenção + layout + regras; `variant` é tratamento visual; `asset` é mídia ou
decoração; `motion` é runtime separado. A superfície pública v0.0.1 permanece
`class` + `<!-- layout/slot -->`, sem nova diretiva pública neste ciclo.

Agendas, recortes/focal point de mídia, grids de 3–4 imagens, steps,
highlights, compare de 3–4 colunas, heróis, multi-chart, tabelas/merges,
gauges radiais e outros gaps identificados são roadmap, não devem ser
interpretados como suporte já entregue. Timelines de objetos, transitions,
posicionamento absoluto, máscaras/vetores compostos, device mockups, ribbons
de marca e round-trip PPTX permanecem fora do Markdown/v0.0.1. Nenhum logo,
foto, ícone, fonte ou asset foi extraído/importado dessa referência.

Também existe uma guia visual autocontida em
`/private/tmp/margo-totvs-deck-audit-20260819/margo-decks-visual-reference.html`.
Ela pode orientar documentação e vocabulário (25 presets, matriz, tokens,
boundaries e roadmap), mas permanece local/temp: não versionar, não promover
por inferência e não importar strings, imagens, SVGs ou elementos TOTVS no
produto.

## Objetivos

1. Implementar decks de slides Margo com perfil compatível e explicitamente versionado para Marpit/Marp.
2. Suportar o conjunto aprovado de layouts e temas Margo, incluindo colunas, sidebar, compare, métricas, timeline e demo.
3. Suportar as extensões de conteúdo do slice: charts, Mermaid, código, imagens/backgrounds, notas do apresentador e diretivas locais/globais.
4. Produzir HTML standalone e PDF determinísticos, com geometria fixa, validação de overflow, evidência de print/PDF e identidade de runtime.
5. Manter acessibilidade semântica: ordem de leitura estável, labels localizados, foco visível, navegação por teclado, alternativas textuais e contraste AA.
6. Comparar visualmente a saída Margo com uma saída equivalente em Marpit, em 16:9 e 4:3, desktop e viewport estreita.
7. Preservar contratos de runtime existentes: descriptors/reports raiz, IDs de tarefas, diagnósticos e hashes canônicos.

## Resultado da revisão da especificação

Foi executado um loop de revisão independente de especificação com subagentes de desenvolvimento e design. Os bloqueadores foram iterados até o marco de especificação ser considerado aceitável.

Pontos fechados no contrato:

- semântica correta de `headingDivider` escalar e por array;
- separadores CommonMark com 0–3 espaços e precedência de Setext H2;
- reset/clear de diretivas herdadas e associação atômica de background + alternativa;
- geometria lógica/visual, escala, print reset e espaço de medição normalizado;
- catálogo estrutural, DOM, ordem de leitura e combinações por layout;
- contraste exaustivo por tema/mode;
- bundle de fontes canônico, pesos, licenças e digest com preimage reproduzível;
- descriptor/report raiz profile-neutral, schemas v2 estritos e tarefas screen/print válidas;
- envelope de evidência PDF não recursivo e hashes conhecidos;
- diagnóstico separado para `deck.font_bundle_mismatch` e `deck.validator_profile_mismatch`.

## Resultado da revisão da implementação visual

### Veredito

**Parcial — 15/32 (Poor).** Duas heurísticas ficaram N/A (#5 e #9); não houve P0, mas existem três P1.

O conteúdo é reconhecivelmente Margo — acento teal editorial, runtime/PDF, charts e evidências — porém o comparador ainda se comporta como uma galeria genérica. A principal falha funcional é que, em 390 px, cada par permanece lado a lado em thumbnails de aproximadamente 158×90, tornando o texto das lâminas ilegível.

### Evidência objetiva

- 30 cards de comparação e 60/60 imagens carregadas;
- imagens com 960×540;
- testes em 1280×720 e 390×844;
- sem erros de console, IDs duplicados ou overflow horizontal;
- `lang=pt-BR`, um H1, 30 H2, `header/main/article` e `alt` não vazio;
- contraste observado: corpo 13,91:1, parágrafo 5,53:1, labels 4,80:1;
- detector Impeccable: exit 2, 2 warnings, 0 errors — ambos relativos a `Inter` no harness de comparação e de baixa relevância para o produto;
- overlay não foi injetado porque a superfície do navegador era somente leitura; foram usados DOM, métricas, screenshots e console.

### Bloqueadores prioritários

Os três P1 abaixo são o registro histórico da rodada R1; as correções foram
implementadas no ciclo atual e reavaliadas em R2.

#### P1 — inspeção em mobile

Adicionar modo empilhado, zoom/1:1, fullscreen, navegação por slide e sincronização de pan/zoom. Validar também 200% de zoom.

#### P1 — tipografia e fidelidade de tema

Embebedar WOFF2 versionado, validar a fonte computada e resolver o risco de fallback serifado nas lâminas “modern”, que deveriam usar Margo Sans/UI sans.

#### P1 — proveniência e significado das diferenças

Exibir SHA da fonte, renderer/versão, viewport e geometria PDF, timestamp, legenda de diferenças, status por slide, notas e modo delta.

#### P2 — arquitetura de revisão

Agrupar o corpus em Foundations, Layouts, Media, Charts e Runtime/PDF; fornecer índice, âncoras, progresso e filtro de itens não revisados; reduzir sombras e chrome.

#### P2 — semântica das capturas

Usar `figure/figcaption`, resumos de diferença e links para o HTML/PDF de origem, em vez de depender apenas de `alt` genérico.

### Review R2 após a primeira correção

- Impeccable Assessment A: **26/40 — Acceptable**, contra 15/32 na R1. O
  comparador agora é reconhecivelmente uma ferramenta Margo/Marpit, com
  proveniência, grupos, filtros, estado e zoom.
- Impeccable Assessment B: detector executado uma vez; o único warning foi um
  falso positivo de imagem do diálogo fechado. A superfície in-app perdeu o
  browser durante a rodada, então a evidência viva R2 foi complementada por
  CDP local reproduzível.
- Checklist Design: a responsividade R1 foi corrigida, mas a auditoria apontou
  foco/bordas com contraste insuficiente, idioma misto sem associação `lang`,
  estado local sem identidade de corpus e manifesto sem identidade de fonte/runtime.

Correções aplicadas depois de R2:

- [x] estado local agora usa a identidade dos manifests + corpus; `needs
  attention` é contado separadamente e reset tem confirmação + undo;
- [x] zoom 1:1 abre Margo e Marpit juntos, empilhando em viewport estreita;
- [x] bordas e focus rings usam tokens com contraste >= 3:1 e controles têm
  alvo mínimo de 44px;
- [x] documento comparador usa `lang="en"` e marca o lede/legenda em
  `lang="pt-BR"`;
- [x] manifesto publica expected/observed font bundle, checks e validation
  identity `margo/runtime-report/v2`;
- [x] capturas Margo foram regeneradas com o bundle WOFF2 real em
  `/tmp/margo-pdf-compare/margo-r3-final4` (960×540), removendo o fallback
  serifado observado na rodada anterior;
- [x] handler de teclado do deck não sequestra setas dentro de controls, links
  ou widgets de chart;
- [x] reflow extremo simulado via CDP (390px e equivalente a 200%) não
  apresenta overflow horizontal além da scrollbar vertical.

## Loop de revisão executado

### 1. Especificação e contrato

- Brainstorm inicial de layouts e compatibilidade Marpit.
- Worktree isolado criado para o slice.
- Revisão paralela da spec por subagentes de desenvolvimento e design.
- Iteração sobre semântica Marpit, geometria, acessibilidade, fontes, runtime reports, IDs e evidência PDF.
- Marco de spec encerrado como aceitável antes da implementação.

### 2. Implementação do slice

- Parser/renderizador de decks, temas, layouts e diretivas.
- Charts, Mermaid e demais extensões previstas.
- Runtime de navegação, controles localizados, foco e overflow.
- HTML standalone, PDF e evidência de validação.
- Corpus de compatibilidade e comparação Margo/Marpit.

### 3. Comparação visual

- Geração de saída equivalente em Marpit.
- Página de comparação em `http://127.0.0.1:8765/margo-pdf-compare/index.html?cb=20260819v3`.
- Inspeção em desktop e viewport estreita.

### 4. Review com `$impeccable`

- Preflight de contexto com `context.mjs`.
- Assessment A em subagente: especificidade, heurísticas de Nielsen, carga cognitiva, jornada emocional, personas e prioridades.
- Assessment B em subagente: detector, DOM, métricas, screenshots, console e comportamento responsivo.
- Critique persistido em:
  `/private/tmp/margo-marpit-decks-v001/.impeccable/critique/2026-08-19T16-52-25Z__index-html.md`
- Trend: primeiro registro desse alvo; ainda sem tendência comparável.

### 5. Review com `$using-checklist-design`

Auditoria independente baseada nos registros do Checklist Design para:

- [Carousel](https://www.checklist.design/design-system/carousel): parcial;
- [Button](https://www.checklist.design/design-system/button): parcial;
- [Card](https://www.checklist.design/design-system/card): desktop consistente, mobile reprovado por legibilidade;
- [Typography](https://www.checklist.design/design-system/typography): escala presente, zoom/viewport estreito insuficiente;
- [Accessibility](https://www.checklist.design/design-system/accessibility): semântica e contraste bons, matriz AT/foco ainda incompleta.

## Evolução do ciclo atual

Implementado após o review 15/32:

- [x] gerador reproduzível em `tools/marpit-compare/make-compare.mjs`;
- [x] grupos Foundations, Layouts, Media, Charts e Runtime/PDF;
- [x] índice, busca, filtro por categoria e estado local de revisão;
- [x] visualização Margo-only/Marpit-only e diálogo de captura 1:1;
- [x] provenance com SHA-256 do corpus/manifests, renderer, viewport, geometria PDF e timestamp;
- [x] layout empilhado em viewport estreita; evidência CDP: viewport 390, `scrollWidth=390`, card 366px, captura 336px;
- [x] fallback tipográfico sans/mono explícito para evitar queda acidental em serif em faces interativas ausentes;
- [x] substituir placeholders do font bundle por WOFF2 versionados reais, embutir aliases/licença e validar `document.fonts` + digest observado;
- [x] ligar evidência de fontes ao `RuntimeReport` Chromium v2 (`FontChecks` + `FontBundleDigest`) e falhar com diagnóstico estável quando houver mismatch;
- [x] vincular o estado de revisão à identidade corpus/manifest, separar `reviewed` de `needs attention` e oferecer confirmação + undo;
- [x] abrir o zoom como comparação pareada Margo/Marpit, com placeholders válidos, captions e alt contextual;
- [x] elevar bordas/foco para contraste de componentes e targets de 44px;
- [x] regenerar as 30 capturas Margo com WOFF2 real e publicar no manifesto a identidade de fonte/runtime observada;
- [x] proteger o handler de teclado contra controls, links e widgets interativos;
- [x] recortar cada captura no canvas lógico `.margo-deck__slide`, removendo
  stage/sombra/borda e congelando 30 PNGs 960×540; `capture-report.json`
  registra 0 overflow em 16:9;
- [x] ajustar charts para o canvas fixo e validar 30/30 slides sem overflow em
  16:9 e 4:3 (`/tmp/margo-exhaustive-4by3-final3`);
- [x] adicionar manifests por slide, `capture-report.json` e
  `runtime-report-v2.json` com browser/engine/platform/font identity e hashes
  reproduzíveis;
- [x] adicionar modo `fit pair` (padrão) + alternância 1:1 no diálogo pareado;
- [x] adicionar export JSON do review com identidade, contagens, estados,
  timestamp e sidecars de evidência;
- [x] alinhar a taxonomia ao corpus real: Media 13–19, Diagrams/Charts 20–26
  e Runtime/PDF 27–30;
- [x] repetir Assessment A, Assessment B e Checklist Design após a nova página.

### 6. Próximo ciclo obrigatório

1. Concluído: reexecutar Assessment A + Assessment B e a auditoria Checklist
   Design sobre a página r6.
2. Concluído: revalidar 16:9, 4:3, 1280×720, 390×844, equivalente a 200%,
   print/PDF, modal, export e acessibilidade estrutural; manter evidência CDP
   quando o browser in-app não estiver disponível.
3. Concluído: registrar aprovação visual/funcional e fechar o goal sem merge,
   push, release ou deploy.

## Critério de saída

O objetivo só estará concluído quando o comparador permitir inspeção legível em mobile e desktop, registrar diferenças/proveniência de forma explícita, preservar os contratos determinísticos de HTML/PDF/runtime e passar novamente pelos dois subagentes Impeccable mais o Checklist Design sem P1 aberto. O goal permanece aberto até esse checkpoint, mesmo com os gates de código e CDP verdes.

## Checkpoint R3/R4 — correções pós-review

- O P0 visual foi fechado: os 30 PNGs Margo em `margo-r3-final4` têm 960×540
  e foram capturados apenas do slide lógico; `capture-report.json` mostra
  `overflow=false` para todos os slides em 16:9.
- A mesma matriz de overflow passou em 4:3 com 30/30 slides em
  `/tmp/margo-exhaustive-4by3-final3`.
- A auditoria CDP do comparador atual (`cb=20260819r5`) encontrou 30 cards,
  60 figuras/60 imagens, 60/60 imagens carregadas após scroll individual,
  zero quebradas, zero erros de console e zero overflow horizontal em 1280 e
  390; no equivalente a 200%, `scrollWidth=204` contra `clientWidth=195`,
  apenas a scrollbar vertical esperada.
- O diálogo pareado abre em `fit-pair` com as duas imagens simultâneas e
  alterna para 1:1; foco retorna ao botão de origem. Reset + undo restaura o
  estado. O export JSON foi exercitado por CDP e baixou
  `margo-marpit-review-337ca30e59b9.json`, com `total=30`,
  `needsAttention=1`, sidecars e `runtimeEvidenceSHA256`.
- O checkpoint Checklist R4 encontrou um P1 de seletor: o atributo
  `data-zoom-title` também existia nos botões. O gerador agora usa `#zoom-title`,
  preserva as 60 imagens após abrir o modal e atualiza o label associado ao
  diálogo.
- O equivalente a 200% tinha 9px de largura documental por campos flexíveis e
  hashes longos. Inputs/selects/toolbar e notas agora têm limites e wrapping;
  no r6 o documento mede `195/195` sem overflow horizontal.
- Impeccable R4 aprovou o comparador sem P0/P1 após a correção de captura,
  fit-pair e export. Checklist R4 anterior foi invalidado pelo seletor stale;
  Checklist R5 e Impeccable R5 agora aprovam o r6 sem P0/P1.
- `GOWORK=off go test ./... -count=1` passou depois das alterações de chart e
  comparação. O navegador serviu o `index.html` com HTTP 200 e bytes idênticos
  ao arquivo local.

## Checkpoint R5 — encerramento do slice

- Impeccable R5: **aprovado; nenhum P0/P1**. O r6 servido corresponde ao
  `index.html` local, SHA-256
  `8974ffd145e5071e387bc37856b156b5e14355c69c5c0d63c8c973837b0a60ea`.
- Checklist Design R5: **nenhum P1 restante**. Carousel, Button e Card
  passaram; Typography, Accessibility e Provenance ficaram parciais apenas
  por lacunas P2 de evidência.
- O defeito do seletor foi fechado com `#zoom-title`: abertura do modal mantém
  60/60 imagens e atualiza `aria-labelledby` para o slide aberto.
- CDP fresco confirmou fit-pair (~586/586 px), 1:1 (960/960 px), foco
  restaurado, zero erros/exceções e equivalente a 195 px sem overflow
  documental (`inner/client/scroll/body = 195`).
- Export real validado: JSON parseável, contagens/identidade/sidecars
  consistentes, SHA-256
  `7d4e7e53daa1c8c009e18e5f6e3b892010a9cf51777415bfcb395139db54d576`.
- Evidência Margo: 30 PNGs 960×540, 30/30 slides sem overflow em 16:9 e 4:3;
  60 assets, tamanhos e hashes íntegros; manifests e runtime report
  recomputáveis dentro do envelope do comparador.
- Limitações P2 explícitas: não houve execução literal de zoom de navegador a
  200% nem teste VoiceOver/NVDA; bytes do corpus não acompanham os sidecars;
  `make-compare.mjs` sintetiza identidade/status do relatório comparativo a
  partir dos argumentos; placeholder em português ainda herda `lang="en"`.
  Nenhuma dessas lacunas bloqueia o slice, mas permanecem backlog de evidência.

## Checkpoint Composições R1 — implementação aprovada

O usuário aprovou a execução depois da revisão da spec e do plano. A
implementação foi concluída neste worktree isolado, sem promover nenhum arquivo
TOTVS e sem criar commit ou push novo.

### Contrato entregue

- catálogo fechado `r1` com `content`, `agenda`, `media-split`, `media-stage`,
  `steps`, `highlight`, `compare-grid`, `hero` e `image-grid`; `none` limpa a
  composição herdada;
- diretiva em frontmatter, corpo e spot, com herança/clear e diagnósticos
  `deck.composition_*` estáveis;
- layouts implícitos ou marcadores explícitos compatíveis, slots semânticos em
  ordem de origem e família `grid` controlada somente por composição;
- HTML com identidade de catálogo, composição, variante, família e slots,
  labels localizados e CSS limitado a tokens/famílias R1;
- digest de tarefa e envelope de validação canônicos incluem catálogo e inputs
  de composição; evidência PDF preserva o caminho legado e registra identidade
  R1 quando usada;
- fixture Markdown autocontida cobrindo todos os nove nomes, manifesto
  comparável, documentação pública e categoria `Compositions R1` no gerador.

### Gates executados

Todos passaram:

```text
GOWORK=off go test ./... -count=1
GOWORK=off go vet ./...
GOWORK=off go mod verify              # all modules verified
gofmt -l .                             # saída vazia
git diff --check
node --check tools/marpit-compare/make-compare.mjs
GOWORK=off go test ./deck ./pdf ./pdf/chromium ./cmd/margo -count=1
```

### Evidência visual e de comparação

Os artefatos são temporários e permanecem fora do repositório em
`/tmp/margo-compositions-r1-eNjPpk/`:

- seis HTML (modern, goshtoso e minimal em 16:9 e 4:3) e seis PDFs Chromium;
  cada PDF tem 10 páginas; 16:9 mede `960 × 540 pt` e 4:3 mede `720 × 540 pt`;
- PNGs revisados de conteúdo, compare-grid e 4:3 minimal em `png/`;
- comparador gerado em `compare/`, com `catalogVersion: r1`, SHA-256 do
  manifesto, nove cards da categoria R1 e nenhuma inclusão da lâmina
  deliberadamente descomposta;
- Playwright/CDP verificou desktop `1280/1280` e mobile `390/390` de largura
  documental, 9 cards R1, 18 labels de composição, filtro de categoria,
  Margo-only, zoom pareado, alternância 1:1, retorno de foco e fechamento do
  diálogo.

### Estado e pendências

O ciclo R1 está concluído para fins de implementação e revisão proporcional,
sem P0/P1 aberto. Permanecem como P2: zoom literal a 200%, VoiceOver/NVDA,
sidecars completos do corpus, captura de identidade do runtime real no
comparador e placeholder PT-BR dedicado. O próximo incremento de storytelling
de dados continua R2 e não foi inferido como suporte atual.

O commit publicado da baseline continua sendo
`335d9892deb937b1516304aa5dd65acb9bd53a90`; qualquer novo commit/push, PR,
merge, tag, release ou deploy exige autorização própria.

## Checkpoint visual — chrome e capa Margo

- [x] Paginação visível reduzida a inteiro (`1`, `2`, …), sempre ancorada no
  canto inferior direito do canvas lógico;
- [x] Header/footer retirados do fluxo de conteúdo, aproximados dos cantos e
  alinhados à esquerda, sem sobreposição no screen ou print;
- [x] Capa/lead/hero usa container vertical de três linhas, com bloco e texto
  centralizados no canvas;
- [x] PDF regenerado e conferido: 42 páginas, `960 × 540 pt`, Tagged; páginas
  1, 2, 10, 13 e 42 revisadas visualmente;
- [x] `GOWORK=off go test ./... -count=1` e `git diff --check` verdes;
- [ ] Commit/push continuam fora do escopo sem autorização nova.

## Checkpoint Marpit equivalent — corpus exhaustivo

- [x] Gerada saída equivalente do mesmo corpus narrativo de 42 slides usando
  `@marp-team/marp-core` 4.4.0 sobre `@marp-team/marpit` 3.2.2;
- [x] Frontmatter convertido para `marp: true`, tema standard, paginação,
  header/footer e `size: 16:9`; classes Margo projetadas para `_class` Marpit;
- [x] Conteúdo, tabelas, listas, código, notas, imagens e roadmap preservados;
  `layout/slot/composition` sem equivalente nativo ficam achatados em ordem de
  origem, sem CSS arbitrário ou posicionamento absoluto;
- [x] Mermaid e Goshtoso permanecem fences de código no baseline puro, com
  limite explicitado no deck; nenhum plugin externo foi fingido como suporte;
- [x] HTML, fonte Markdown e PDF temporários em
  `/tmp/margo-marpit-equivalent/`; somente assets Margo já existentes foram
  staged localmente, sem copiar ou versionar material TOTVS;
- [x] PDF Chromium verificado: 42 páginas, `960 × 540 pt`, Tagged, 1.973
  palavras dentro do MediaBox, 42 seções HTML e quatro imagens com `alt`;
- [x] Amostras visuais revisadas: capa centralizada, agenda, media stage,
  metrics, timeline, Mermaid, chart fence, matrizes, tema invertido e slide
  final;
- [ ] Publicação em commit/push permanece fora deste checkpoint e exige nova
  autorização explícita.

## Estado do slice base

Objetivos de implementação, comparação visual, runtime/PDF, acessibilidade
estrutural e revisão independente concluídos neste worktree isolado. O slice
foi publicado no commit `335d9892deb937b1516304aa5dd65acb9bd53a90` da branch
`feat/marpit-decks-v001`. `goal.md` permanece local e não versionado. O ciclo
ativo está descrito no início deste documento; não há autorização implícita
para novo commit ou push, PR, merge, tag, release, deploy ou importação do
material TOTVS.
