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
| `TALOS_WORKSHOP_NAME` | não | `Workshop` |

Nunca registre `TALOS_DATABASE_URL`, pois ela pode conter credenciais.

### Health

`GET /health/live` responde sempre com HTTP `200` e o JSON estável abaixo
enquanto o processo HTTP estiver vivo:

```json
{"status":"ok"}
```

Liveness não consulta PostgreSQL, migrations, file storage ou impressoras. Use
esse endpoint apenas para detectar se o processo deve ser reiniciado; o estado
das dependências é exposto por `GET /health/ready`.

Readiness retorna HTTP `200` com `{"status":"ok"}` quando PostgreSQL responde,
as migrations estão na versão incorporada ao servidor e o diretório de storage
permite criar e remover um arquivo de teste. Caso contrário, retorna HTTP `503`
com `{"status":"unavailable"}` sem expor detalhes internos. Estado de
impressora não participa da readiness.

### API v1

Endpoints de produto são registrados sob `/api/v1`. Erros da API usam sempre
`Content-Type: application/json` e o envelope:

```json
{
  "error": {
    "code": "route_not_found",
    "message": "Route not found",
    "details": {}
  }
}
```

Cada requisição da API recebe `X-Request-ID` na resposta e o mesmo valor no
contexto do handler. Um identificador seguro enviado pelo cliente é preservado;
valores ausentes ou inválidos são substituídos por um UUID aleatório.

`GET /api/v1/meta` retorna `api_version`, `server_version`,
`minimum_desktop_version` e `workshop_name`. Enquanto SET-001 não persistir as
configurações da oficina, o nome vem de `TALOS_WORKSHOP_NAME`. Builds de release
podem definir a versão do servidor com o argumento Docker `SERVER_VERSION`;
builds locais retornam `dev`.

### Servidor e PostgreSQL via Docker Compose

O servidor e o PostgreSQL rodam juntos via Docker Compose. O banco permanece na
rede interna e não publica a porta `5432` no host. O servidor acessa o banco
pelo nome de serviço `postgres`:

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

Para iniciar a pilha completa, use:

```powershell
docker compose up -d --build
docker compose logs -f server
```

Por padrão, a API fica disponível apenas no host local em
`http://127.0.0.1:8080`. Para disponibilizá-la na LAN confiável da oficina,
defina simultaneamente no `.env`:

```dotenv
TALOS_SERVER_BIND_ADDRESS=0.0.0.0
TALOS_TRUSTED_LAN=true
```

Não publique a API fora de uma LAN confiável sem antes adicionar TLS.

O banco persiste em `postgres_data` e os objetos do servidor em `server_data`.
Pare a pilha sem apagar os volumes com `docker compose down`. Não use
`docker compose down --volumes` a menos que queira apagar deliberadamente todos
os dados locais.


## Release 1.1 domain clarifications

- Jobs may be non-commercial.
- Catalog pricing works before quote/order/sale.
- Marketplace fee profiles are manual in Release 1.
- Quotes never become orders implicitly.
- Internal labor cost and billable service price are separate.
