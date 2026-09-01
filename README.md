# Oficina 3D — Pacote de Especificação v1.0

Arquivos:

- `PRD.md` — produto, arquitetura, regras e critérios de aceite.
- `AGENTS.md` — regras operacionais para agentes de IA.
- `IMPLEMENTATION_TASKS.md` — backlog da Release 1 quebrado em tasks pequenas e dependências.

O produto ainda não possui nome definitivo.

## Estado atual

O repositório está na fase de bootstrap. Os diretórios de entrada existem,
mas o servidor e o desktop ainda serão inicializados pelas tasks `BOOT-002` e
`BOOT-003`.

## Entradas de desenvolvimento

- `cmd/server/` — entrypoint do servidor Go;
- `internal/` — código interno do servidor, organizado por domínio e camada;
- `desktop/` — aplicação Windows Wails + React + TypeScript;
- `migrations/` — migrations PostgreSQL numeradas e unidirecionais;
- `docs/adr/` — registros de decisões arquiteturais aprovadas;
- `docs/architecture/` — documentação da arquitetura;
- `scripts/` — scripts operacionais e de desenvolvimento.

Os comandos de build, teste e lint serão documentados em `BOOT-004`, depois
que os projetos Go e Wails existirem.


## Release 1.1 domain clarifications

- Jobs may be non-commercial.
- Catalog pricing works before quote/order/sale.
- Marketplace fee profiles are manual in Release 1.
- Quotes never become orders implicitly.
- Internal labor cost and billable service price are separate.
