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

## Integração contínua

O workflow `.github/workflows/ci.yml` executa em pull requests e em pushes para
`main`:

- testes, race detector, lint e build do backend em Linux;
- instalação limpa, lint, typecheck e build Wails do desktop no Windows.

O workflow apenas valida o repositório. Ele não publica artefatos nem realiza
deployment.

### Servidor Go

Requisitos:

- Go 1.27 ou superior compatível.

Executar localmente:

```powershell
$env:TALOS_DATABASE_URL = "postgres://talos:change-me@localhost:5432/talos?sslmode=disable"
go run ./cmd/server
```

Copie `.env.example` como referência e forneça os valores pelo ambiente do
processo. O servidor não carrega arquivos `.env` automaticamente.

| Variável | Obrigatória | Padrão |
|---|---:|---|
| `TALOS_SERVER_PORT` | não | `8080` |
| `TALOS_DATABASE_URL` | sim | nenhum |
| `TALOS_DB_MAX_OPEN_CONNS` | não | `10` |
| `TALOS_DB_MAX_IDLE_CONNS` | não | `5` |
| `TALOS_DB_CONN_MAX_LIFETIME` | não | `30m` |
| `TALOS_DB_CONN_MAX_IDLE_TIME` | não | `5m` |
| `TALOS_DB_PING_TIMEOUT` | não | `5s` |
| `TALOS_DATA_DIR` | não | `./data` |
| `TALOS_TRUSTED_LAN` | não | `false` |
| `TALOS_UPLOAD_MAX_BYTES` | não | `104857600` |
| `TALOS_DEFAULT_LOCALE` | não | `pt-BR` |
| `TALOS_DEFAULT_CURRENCY` | não | `BRL` |
| `TALOS_DEFAULT_TIMEZONE` | não | `America/Sao_Paulo` |

Nunca registre `TALOS_DATABASE_URL`, pois ela pode conter credenciais.

### PostgreSQL via Docker Compose

O PostgreSQL de desenvolvimento roda na rede interna do Compose e não publica
a porta `5432` no host por padrão. Contêineres no mesmo projeto Compose acessam
o banco pelo nome de serviço `postgres`:

```text
postgres:5432
```

Antes da primeira execução, copie `.env.example` para `.env` e substitua
`POSTGRES_PASSWORD` por uma senha local. O arquivo `.env` é ignorado pelo Git.

```powershell
Copy-Item .env.example .env
docker compose up -d postgres
docker compose ps
```

Os dados persistem no volume nomeado `postgres_data`. Pare o serviço sem apagar
o volume com `docker compose down`. Não use `docker compose down --volumes` a
menos que queira apagar deliberadamente o banco local.

Até `DEP-002` adicionar o servidor ao Compose, uma execução do servidor
diretamente no host precisa usar uma instância PostgreSQL separada que seja
acessível pelo host. A configuração Compose padrão não enfraquece a fronteira
de segurança publicando o banco apenas para esse fluxo temporário.


## Release 1.1 domain clarifications

- Jobs may be non-commercial.
- Catalog pricing works before quote/order/sale.
- Marketplace fee profiles are manual in Release 1.
- Quotes never become orders implicitly.
- Internal labor cost and billable service price are separate.
