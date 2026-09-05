# Fila de integração dos Work Packages

Registro verificado em 2026-09-05 UTC (2026-09-04 em America/Sao_Paulo).
Base integrada: `main` em `c7abe0c`. Reconciliação iniciada em `92cd5b8`.

Os parágrafos e a tabela de preparação abaixo são um registro histórico.

## Execução autorizada dos merges

Após o registro inicial, o usuário autorizou explicitamente integrar todos os
11 PRs. Os dez predecessores foram confirmados como MERGED em `main`, na ordem
abaixo, após checks aprovados. O PR #31 entrega manutenção e este registro final;
quando este documento estiver em `main`, a fila inteira estará integrada.

| PR | Merge commit | Data/hora UTC |
|---|---|---|
| #32 | `cdea7fe` | 2026-09-05T00:41:47Z |
| #33 | `8926e95` | 2026-09-05T00:42:48Z |
| #34 | `20203ad` | 2026-09-05T00:42:59Z |
| #35 | `a54eb7c` | 2026-09-05T00:43:09Z |
| #36 | `8ba4eb3` | 2026-09-05T00:43:19Z |
| #37 | `202fb97` | 2026-09-05T00:44:00Z |
| #38 | `3446ab5` | 2026-09-05T00:44:11Z |
| #39 | `d605e44` | 2026-09-05T00:44:21Z |
| #40 | `9b86292` | 2026-09-05T00:44:32Z |
| #41 | `3192309` | 2026-09-05T00:44:43Z |
| [#31](https://github.com/Wilian-N-Silva/talos-3d-workshop-management/pull/31) | PR de entrega deste registro | Consultar merge confirmado no GitHub |

As bases históricas da tabela original abaixo documentam a preparação da fila.
Durante a execução, cada PR foi redirecionado para `main` antes do merge.
A conferência verificou que o conteúdo de main correspondia ao predecessor e
que o SHA de head permanecia o esperado. O CI da integração final deve ser
observado no GitHub; o registro não presume resultados ainda pendentes.


Os 11 pacotes abaixo já estavam implementados em uma cadeia local. Foram
publicados com um PR por pacote; apenas o primeiro está pronto para revisão.
Os demais permanecem em draft até a integração do predecessor. O PR #31 foi
preservado e ocupa a última posição; número de PR não determina ordem de merge.

| Ordem | Pacote | Tasks | PR | Base de revisão original | Migration | Commit verificado |
|---|---|---|---|---|---|---|
| 1 | WP-CAT-01 | CAT-001, CAT-002, CAT-003 | [#32](https://github.com/Wilian-N-Silva/talos-3d-workshop-management/pull/32) | `main` | 00009 | `f447cb5` |
| 2 | WP-CAT-02 | CAT-004, CAT-005, CAT-006, CAT-007, CAT-008, CAT-009 | [#33](https://github.com/Wilian-N-Silva/talos-3d-workshop-management/pull/33) | `work/wp-cat-01-catalog-items` | 00010 | `69c6947` |
| 3 | WP-INV-01 | INV-001, INV-002, INV-003 | [#34](https://github.com/Wilian-N-Silva/talos-3d-workshop-management/pull/34) | `work/wp-cat-02-design-versions` | 00011 | `93c2353` |
| 4 | WP-INV-02 | INV-004, INV-005, INV-006 | [#35](https://github.com/Wilian-N-Silva/talos-3d-workshop-management/pull/35) | `work/wp-inv-01-filament-inventory` | 00012 | `32e3b32` |
| 5 | WP-CAT-03 | BOM-001 | [#36](https://github.com/Wilian-N-Silva/talos-3d-workshop-management/pull/36) | `work/wp-inv-02-supplies-stock` | 00013 | `4284c92` |
| 6 | WP-PRN-01 | PRN-001 | [#37](https://github.com/Wilian-N-Silva/talos-3d-workshop-management/pull/37) | `work/wp-cat-03-supply-bom` | 00014 | `969b92b` |
| 7 | WP-JOB-01 | JOB-001, JOB-002, JOB-005, JOB-006 | [#38](https://github.com/Wilian-N-Silva/talos-3d-workshop-management/pull/38) | `work/wp-prn-01-printer-registry` | 00015 | `2ea049b` |
| 8 | WP-JOB-02 | JOB-003, JOB-004 | [#39](https://github.com/Wilian-N-Silva/talos-3d-workshop-management/pull/39) | `work/wp-job-01-lifecycle` | 00016 | `502e01f` |
| 9 | WP-ENERGY-01 | ENERGY-001 | [#40](https://github.com/Wilian-N-Silva/talos-3d-workshop-management/pull/40) | `work/wp-job-02-material-usage` | 00017 | `6bdfc6a` |
| 10 | WP-LABOR-01 | LABOR-001, LABOR-002 | [#41](https://github.com/Wilian-N-Silva/talos-3d-workshop-management/pull/41) | `work/wp-energy-01-measurements` | 00018 | `389ee27` |
| 11 | WP-MAINT-01 | MAINT-001 | [#31](https://github.com/Wilian-N-Silva/talos-3d-workshop-management/pull/31) | `work/wp-labor-01-internal-time` | 00019 | `b0ec097` |

## Procedimento de integração

1. Revisar o primeiro PR e aguardar todos os checks obrigatórios.
2. Integrá-lo em `main` com **Create a merge commit**.
3. Alterar a base do PR seguinte para `main` antes de apagar a branch usada
   anteriormente como base. Não mesclar o PR seguinte na branch do predecessor.
4. Conferir o diff, as migrations e o CI após a mudança de base; retirar o draft
   somente quando a dependência estiver em `main`.
5. Repetir a sequência até #31. Nenhum merge está autorizado por este registro;
   a execução desta tarefa publicou e ordenou os PRs, sem integrar código em main.

O repositório permite merge commits. Para esta cadeia já existente, preservar
os ancestrais evita reescrever branches e reapresentar commits dos predecessores.
É uma exceção pontual à preferência por squash de `GIT_WORKFLOW.md`.
Se for escolhido squash ou rebase merge, é necessário reconstruir as bases dos
PRs descendentes e repetir a verificação; não basta trocar a base no GitHub.

A ordem acima foi simulada com `git merge-tree --write-tree` e commits temporários
sem atualização de refs de integração: os 11 merges terminaram sem conflitos.
Isso vale para os commits listados e a base indicada; alterações posteriores em
main ou nos pacotes exigem nova verificação.

## Verificação e correções de dependências

A revisão conferiu as tasks, migrations, documentação, rotas e testes dos
pacotes. Cada versão foi testada em um checkout isolado com PostgreSQL 18:

- `go test ./...` com `TALOS_TEST_DATABASE_URL` definido, incluindo migrations
  e repositórios;
- `go vet ./...` e `go build -o bin/server.exe ./cmd/server`;
- nos seis pacotes que alteram desktop (CAT-01, CAT-02, INV-01, INV-02, CAT-03,
  JOB-02): testes e vet nativos, lint e typecheck do frontend e
  `wails build -m -nosyncgomod` no Windows;
- `git diff --check` em cada diff de pacote.

Três ajustes já existentes em `fb4b840` (Jobs) foram antecipados para que as
versões anteriores também passem isoladamente:

| Commit | Pacote | Ajuste |
|---|---|---|
| `69c6947` | WP-CAT-02 | Reconhecer SQLSTATE 23001 nas restrições de exclusão do catálogo. |
| `93c2353` | WP-INV-01 | Reconhecer SQLSTATE 23001 nas restrições de exclusão do inventário. |
| `969b92b` | WP-PRN-01 | Limpar a tabela printers na fixture que recria o schema para testar migrations. |

Os ajustes foram propagados com merges sem force-push. Um conflito na fixture
foi resolvido mantendo a limpeza de job_events, print_jobs e printers, nesta
ordem. As merges de propagação posteriores não mudaram o conteúdo já corrigido
do pacote de Jobs em diante. Antes deste registro documental, o conteúdo final
em `b0ec097` era idêntico ao de `92cd5b8`.

Não houve nova migration, alteração de fórmula financeira ou mudança de escopo
de produto. As saídas regeneradas pelo Wails ficaram na cópia temporária.
O CI de cada PR continua sendo obrigatório; passar localmente não substitui CI
nem revisão. Os links dos PRs acima mostram o estado atualizado dos checks.
