# Oficina 3D — Pacote de Especificação v1.0

Arquivos:

- `PRD.md` — produto, arquitetura, regras e critérios de aceite.
- `AGENTS.md` — regras operacionais para agentes de IA.
- `IMPLEMENTATION_TASKS.md` — backlog da Release 1 quebrado em tasks pequenas e dependências.

O produto ainda não possui nome definitivo.

## Estado atual

O repositório está na fase de bootstrap. O servidor Go mínimo e o shell
desktop Wails estão disponíveis.

## Entradas de desenvolvimento

- `cmd/server/` — entrypoint do servidor Go;
- `internal/` — código interno do servidor, organizado por domínio e camada;
- `desktop/` — aplicação Windows Wails + React + TypeScript;
- `migrations/` — migrations PostgreSQL numeradas e unidirecionais;
- `docs/adr/` — registros de decisões arquiteturais aprovadas;
- `docs/architecture/` — documentação da arquitetura;
- `scripts/` — scripts operacionais e de desenvolvimento.

## Verificações padronizadas

Execute os comandos a partir da raiz do repositório.

Backend:

```powershell
go test ./...
go vet ./...
go build -o ./bin/server.exe ./cmd/server
```

Desktop frontend:

```powershell
npm.cmd --prefix ./desktop/frontend ci
npm.cmd --prefix ./desktop/frontend run lint
npm.cmd --prefix ./desktop/frontend run typecheck
npm.cmd --prefix ./desktop/frontend run build
```

Desktop Windows completo:

```powershell
Push-Location ./desktop
wails build
Pop-Location
```

`BOOT-005` executará essas verificações automaticamente em pull requests.

### Servidor Go

Requisitos:

- Go 1.27 ou superior compatível.

Executar localmente:

```powershell
$env:TALOS_SERVER_ADDRESS = ":8080"
go run ./cmd/server
```

`TALOS_SERVER_ADDRESS` é opcional e usa `:8080` por padrão. A configuração
centralizada será implementada em `CFG-001`.


## Release 1.1 domain clarifications

- Jobs may be non-commercial.
- Catalog pricing works before quote/order/sale.
- Marketplace fee profiles are manual in Release 1.
- Quotes never become orders implicitly.
- Internal labor cost and billable service price are separate.
