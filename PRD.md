# PRD 1.1 — Plataforma de Gestão de Oficina 3D

> **Nome do produto:** a definir  
> **Status:** Draft consolidado para revisão final antes da implementação  
> **Versão:** 1.1  
> **Data:** 01/09/2026  
> **Estratégia:** greenfield, com reutilização seletiva de referências/código MIT quando vantajoso  
> **Aplicação principal da Release 1:** desktop Windows  
> **Servidor da Release 1:** local, via Docker, acessível pela LAN  
> **Objetivo documental:** servir simultaneamente como PRD humano e fonte de verdade para agentes de IA

---

## 0.0. Alterações da versão 1.1

Esta revisão esclarece a independência entre operação física e comercial.

Mudanças principais:

- Job pode existir sem cliente, orçamento, pedido ou venda;
- adicionado `internal` ao purpose de Job;
- Jobs não comerciais consomem estoque, energia, horas de máquina e mão de obra normalmente;
- custo de Job não comercial é classificado como consumo operacional, não como prejuízo comercial;
- precificação passa a existir diretamente no item de catálogo, antes de qualquer orçamento;
- comparação de preço por canal/marketplace usa perfis manuais de taxas, sem integração de marketplace;
- orçamento é independente e nunca cria pedido ou receita automaticamente;
- conversão de orçamento para pedido exige ação explícita;
- mão de obra passa a separar custo interno da hora e preço cobrado ao cliente;
- serviços de modelagem, personalização, acabamento e outras atividades humanas podem ser precificados separadamente.

---

## 0. Como agentes devem usar este documento

Este PRD define **produto, arquitetura, limites e critérios de aceite**.

Agentes de implementação **não devem transformar seções inteiras deste PRD em uma única task**.

A execução deve ocorrer pelas tasks pequenas definidas em `IMPLEMENTATION_TASKS.md`.

### Ordem de autoridade

Em caso de conflito:

1. task ativa;
2. este PRD;
3. ADRs aprovados;
4. `AGENTS.md`;
5. código existente;
6. preferências implícitas do agente.

Uma task não pode alterar uma decisão fechada deste PRD sem criar um ADR separado e receber aprovação.

### Regra de escopo

Cada task deve:

- resolver um único objetivo principal;
- possuir critérios de aceite verificáveis;
- declarar dependências;
- evitar refatorações não relacionadas;
- adicionar ou atualizar testes;
- atualizar documentação afetada;
- terminar com o repositório em estado executável.

---

# 1. Visão do produto

Criar uma plataforma para administrar uma microoficina de fabricação aditiva, começando por impressão 3D FDM.

A plataforma possui dois fluxos independentes que se conectam quando necessário.

### Fluxo operacional

```text
Compra / estoque
      ↓
Material / bobina / suprimento
      ↓
Catálogo / design / versão
      ↓
Job físico
      ↓
Consumo / energia / mão de obra / horas de máquina
      ↓
Resultado / qualidade
      ↓
Custo real / consumo operacional
```

### Fluxo comercial

```text
Catálogo + Cost Engine
      ↓
Precificação por canal
      ↓
Preço de referência / tabela
      ↓
Orçamento opcional
      ↓
Pedido opcional
      ↓
Jobs opcionais vinculados
      ↓
Rentabilidade comercial
```

Regras:

- nem todo Job é comercial;
- nem toda precificação gera orçamento;
- nem todo orçamento gera pedido;
- um orçamento aceito não gera pedido ou receita automaticamente;
- o sistema deve permitir estudar preço e viabilidade comercial antes de existir cliente ou venda.

O sistema não será apenas uma calculadora de impressão nem apenas um catálogo de arquivos.

Ele deverá ser a fonte de verdade operacional e econômica da oficina.

---

# 2. Problemas que o produto resolve

A oficina precisa responder com dados auditáveis:

1. Qual arquivo e versão foram usados?
2. Qual impressora executou a peça?
3. Qual bobina foi usada?
4. Quanto material foi planejado?
5. Quanto material foi realmente consumido?
6. Quanto virou modelo, suporte, purga ou perda?
7. Quanto a impressão consumiu de energia?
8. Quanto tempo de máquina foi utilizado?
9. Quanto tempo humano foi utilizado?
10. A impressão foi aprovada ou falhou?
11. Quanto custou fabricar?
12. Quanto custa vender em determinado canal?
13. Qual é o menor preço sustentável?
14. Qual preço atinge a margem desejada?
15. Qual preço estava vigente quando a venda ocorreu?
16. Qual produto, cliente, canal ou lote é mais rentável?
17. Quais arquivos podem ser comercializados legalmente?
18. Quais pedidos ainda precisam ser produzidos?
19. Qual é o saldo físico estimado de cada bobina?
20. Quais pendências operacionais precisam de atenção?
21. Quanto um Job interno, pessoal, de teste ou protótipo consumiu sem gerar receita?
22. Quanto preciso cobrar pelo mesmo item em Pix, cartão, B2B, Mercado Livre ou Shopee segundo as taxas cadastradas?
23. Qual canal oferece melhor contribuição para determinado item e quantidade?
24. Quanto custa uma hora efetivamente produtiva de trabalho humano?
25. Quanto devo cobrar por modelagem, personalização, acabamento ou outro serviço humano?
26. Um orçamento pode ser criado, enviado, aceito, rejeitado ou expirar sem gerar pedido ou venda?

---

# 3. Princípios do produto

## P-001 — Dados centralizados; controle físico local

Os dados da oficina são centralizados.

O controle físico da impressora não é.

O servidor:

- armazena dados;
- autentica usuários;
- aplica RBAC;
- armazena arquivos;
- recebe telemetria;
- fornece catálogo, Jobs e informações comerciais.

O servidor **não**:

- inicia impressão;
- pausa impressão;
- continua impressão;
- cancela impressão;
- envia comandos à impressora;
- acessa remotamente o computador do operador.

## P-002 — O Bambu Handy/Bambu Studio continuam sendo ferramentas de controle

Se o usuário precisar controlar a impressora remotamente, utiliza o Bambu Handy.

Se precisar preparar/slicar/imprimir, utiliza o Bambu Studio.

A aplicação integra-se a essas ferramentas sem tentar substituí-las.

## P-003 — Job representa execução física

Um Job não é um produto e não é um arquivo.

Um Job representa uma execução física relevante da impressora, comercial ou não comercial.

## P-004 — Histórico não deve desaparecer

Sempre que houver valor operacional em conhecer a evolução, usar eventos/medições em vez de sobrescrever silenciosamente o passado.

Exemplos:

- pesagens de bobina;
- mudanças de preço;
- movimentos de estoque;
- versões de design;
- snapshots de custo;
- eventos relevantes de Job.

## P-005 — Custo não é preço

O sistema deverá tratar separadamente:

1. custo de fabricação;
2. custo comercial;
3. estratégia de preço.

## P-006 — Automação sem esconder origem

O sistema pode automatizar cálculos, mas todo resultado financeiro relevante deve ser decomponível e auditável.

## P-007 — Simplicidade de operação

A complexidade deve ficar no motor, não no formulário diário.

O sistema deverá herdar automaticamente dados de:

- oficina;
- máquina;
- material;
- bobina;
- design;
- produto;
- canal;
- parâmetros financeiros.

## P-008 — Job independe de contexto comercial

Todo uso físico relevante da impressora pode ser registrado como Job sem cliente, orçamento, pedido ou venda.

Jobs não comerciais:

- consomem material e supplies;
- registram energia;
- acumulam horas de máquina;
- registram mão de obra;
- podem gerar Cost Snapshot;
- entram no histórico de manutenção e utilização;
- não geram receita;
- não devem ser classificados como prejuízo comercial.

## P-009 — Comercial é opt-in

Precificação, orçamento, pedido e produção comercial são conceitos separados.

Nenhuma transição comercial ocorre implicitamente.

Em especial:

- calcular preço não cria orçamento;
- criar orçamento não cria pedido;
- marcar orçamento como aceito não cria pedido;
- criar pedido não inicia impressão;
- Job só é vinculado ao pedido quando houver relação comercial real.

## P-010 — Precificação deve existir antes da venda

Um item de catálogo deve poder ser analisado economicamente sem cliente e sem venda.

A área de precificação deve permitir comparar canais com perfis manuais de taxas e responder onde vale a pena focar comercialmente antes de implementar integrações de marketplace.

---

# 4. Decisões arquiteturais fechadas

| Tema | Decisão |
|---|---|
| Fork ou greenfield | **Greenfield** |
| Daedalus | referência/source donor seletivo, não base do repositório |
| Backend servidor | **Go** |
| API | HTTP JSON |
| Router | Chi ou equivalente leve; troca exige ADR |
| Desktop | **Wails + React + TypeScript** |
| SO desktop Release 1 | **Windows-only** |
| Banco | **PostgreSQL** |
| ORM | **não introduzir ORM sem ADR** |
| Acesso ao banco | somente servidor |
| Desktop → banco | proibido |
| Desktop → servidor | HTTP API na LAN |
| Servidor → desktop | nenhum mecanismo de execução remota |
| Servidor → impressora | proibido |
| Desktop → impressora | somente integração local, inicialmente telemetria |
| Controle remoto da impressora | fora do produto; usar Bambu Handy |
| Controle local da impressora | usar Bambu Studio/Handy; nosso app não implementa Start/Pause/Cancel na Release 1 |
| Arquivos | storage no servidor |
| Deploy servidor | Docker Compose |
| Deploy desktop | instalador/binário Windows |
| Web app | fora da Release 1 |
| Internet pública | não necessária |
| TLS na Release 1 | opcional em LAN confiável; obrigatório antes de exposição fora da LAN |
| Offline/sync | fora de escopo |
| Moeda inicial | BRL |
| Locale inicial | pt-BR |
| Timezone inicial | America/Sao_Paulo |
| Datas no backend | UTC |
| Dinheiro | inteiros em centavos |
| Medidas físicas | decimal/NUMERIC |
| Tema | claro, escuro ou sistema |
| Branding | nome + logo; sem sistema de temas customizados |

---

# 5. Fronteiras de segurança

## SEC-BOUNDARY-01 — Impressora

Não deve existir caminho:

```text
Servidor → Impressora
```

## SEC-BOUNDARY-02 — Desktop

Não deve existir mecanismo em que o servidor envie comandos arbitrários ao desktop para execução.

O desktop é um cliente que inicia conexões para o servidor.

## SEC-BOUNDARY-03 — Credenciais Bambu

Dados necessários para integração local com a impressora, como access code, devem permanecer no dispositivo autorizado.

Armazenamento preferencial no Windows Credential Manager.

Não persistir access code da impressora no servidor.

## SEC-BOUNDARY-04 — PostgreSQL

PostgreSQL não deve ser publicado diretamente para clientes desktop.

Somente o servidor acessa o banco.

## SEC-BOUNDARY-05 — Arquivos

Arquivos do catálogo não devem ser expostos por diretório de rede público.

Downloads passam pela camada de autorização do servidor.

---

# 6. Topologia da Release 1

```text
                       LAN DA OFICINA

              ┌─────────────────────────┐
              │     SERVER DOCKER       │
              │                         │
              │ API Go                  │
              │ Auth / RBAC             │
              │ PostgreSQL              │
              │ File Storage            │
              │ Backup                  │
              └────────────▲────────────┘
                           │
                         HTTP
                           │
          ┌────────────────┴────────────────┐
          │                                 │
┌─────────┴──────────┐           ┌──────────┴─────────┐
│ Desktop Windows A │           │ Desktop Windows B   │
│ Wails             │           │ Wails               │
│ React + TS        │           │ React + TS          │
│ Go local layer    │           │ Go local layer      │
└─────────┬──────────┘           └────────────────────┘
          │
          │ apenas local
          ├──────────────► Bambu Studio
          │
          └──────────────► Bambu A1 Mini
                           telemetria LAN
```

## 6.1. Fronteira interna do desktop

O frontend React não deve possuir credenciais do servidor nem implementar acesso HTTP autenticado diretamente.

Fluxo preferido:

```text
React / TypeScript
      ↓
Wails bindings
      ↓
Go desktop application layer
      ↓
Go remote API client
      ↓
Workshop Server
```

O Go local:

- lê/grava sessão no Windows Credential Manager;
- adiciona credencial à requisição HTTP;
- aplica timeout/retry apropriado;
- redige secrets em logs;
- expõe ao React apenas DTOs necessários.

Isso reduz exposição de token no WebView e elimina a necessidade de CORS para o cliente desktop oficial.

---

# 7. Estrutura sugerida do repositório

```text
/
├── cmd/
│   └── server/
├── internal/
│   ├── domain/
│   │   ├── auth/
│   │   ├── catalog/
│   │   ├── inventory/
│   │   ├── jobs/
│   │   ├── costing/
│   │   ├── pricing/
│   │   ├── commercial/
│   │   └── printers/
│   ├── application/
│   ├── infrastructure/
│   │   ├── postgres/
│   │   ├── filestorage/
│   │   ├── auth/
│   │   └── backup/
│   └── platform/
│       └── http/
├── desktop/
│   ├── app/
│   ├── internal/
│   │   ├── remote/
│   │   ├── bambu/
│   │   ├── bambustudio/
│   │   ├── cache/
│   │   └── credentials/
│   └── frontend/
├── migrations/
├── docs/
│   ├── adr/
│   └── architecture/
├── scripts/
├── docker-compose.yml
├── AGENTS.md
├── IMPLEMENTATION_TASKS.md
└── PRD.md
```

A estrutura poderá ser ajustada durante BOOT-001 se houver ganho claro, sem alterar as fronteiras de domínio.

---

# 8. Autenticação e RBAC

## 8.1. Usuários

Usuários existem no servidor.

Campos mínimos:

```text
id
name
email_or_username
password_hash
status
created_at
updated_at
last_login_at
```

Senha:

- Argon2id;
- parâmetros documentados e atualizáveis;
- nunca armazenar senha pura.

## 8.2. Sessões desktop

Fluxo:

```text
Desktop
  ↓
login
  ↓
Servidor valida senha
  ↓
token opaco aleatório
  ↓
desktop salva token no Credential Manager
```

No servidor:

```text
sessions
- id
- user_id
- device_id
- token_hash
- created_at
- expires_at
- last_used_at
- revoked_at
```

Token puro não é armazenado no banco.

## 8.3. Devices

Registrar instalações autorizadas:

```text
client_devices
- id
- display_name
- os
- app_version
- created_at
- last_seen_at
```

O registro serve para auditoria e revogação de sessão.

Não permite controle remoto.

## 8.4. Permissões

O backend autoriza permissões, não nomes de roles.

Permissões iniciais:

```text
catalog.read
catalog.write
files.read
files.upload

inventory.read
inventory.write

jobs.read
jobs.create
jobs.update
jobs.evaluate

costing.read
costing.manage
pricing.read
pricing.manage

commercial.read
commercial.write

telemetry.read
telemetry.publish

users.manage
settings.manage
backup.manage
```

## 8.5. Roles iniciais

### Owner/Admin

Todas as permissões.

### Operator

Operação, Jobs, catálogo de leitura, inventário e telemetria.

### Designer

Catálogo, arquivos, designs e versões.

### Commercial

Clientes, orçamentos, pedidos, preços e leitura de catálogo/custos permitidos.

### Viewer

Somente leitura autorizada.

Roles poderão ser editáveis posteriormente; a Release 1 pode usar perfis fixos com vínculo `role → permissions`.

## 8.6. Primeiro usuário

Se não houver usuário:

```text
GET /api/setup/status
```

retorna `needs_setup=true`.

Somente durante esse estado:

```text
POST /api/setup/admin
```

pode criar o primeiro Owner/Admin.

Após criação, endpoint de bootstrap fica permanentemente indisponível.

---

# 9. Configurações e personalização

## SET-001 — Workshop Settings

Campos iniciais:

```text
workshop_name
logo_file_id
default_locale
default_currency
display_timezone
default_theme
```

`default_theme`:

```text
light
dark
system
```

Sem:

- custom CSS;
- cor primária configurável;
- marketplace de temas;
- tokens de design editáveis.

Na Release 1 o branding afeta:

- login;
- cabeçalho;
- janela desktop;
- relatórios/orçamentos;
- documentos exportados.

Não é requisito alterar dinamicamente ícone do executável Windows.

---

# 10. File Storage

## 10.1. Modelo

Arquivos são objetos imutáveis.

```text
files
- id
- sha256
- original_name
- content_type
- size_bytes
- storage_key
- uploaded_by
- created_at
```

Deduplicação por SHA-256 é permitida.

## 10.2. Storage

Implementação inicial:

```text
LocalFilesystemStorage
```

dentro do volume Docker.

Estrutura física não deve depender do nome original.

Exemplo:

```text
/data/objects/ab/abcdef...
```

## 10.3. Download

Download exige permissão apropriada.

Desktop baixa para cache local quando necessário.

## 10.4. Cache desktop

Cache não é fonte de verdade.

Pode ser apagado sem perda de dados.

Diretório sugerido:

```text
%LOCALAPPDATA%\<app>\cache
```

---

# 11. Catálogo

## 11.1. Catalog Item

O catálogo contém itens da oficina, não apenas produtos vendidos.

```text
catalog_items
- id
- name
- sku
- description
- purpose
- sellable
- tags
- status
- created_at
- updated_at
```

`purpose`:

```text
product
prototype
tooling
test
internal
personal
```

## 11.2. Partes

Um item pode possuir uma ou mais partes impressas.

```text
catalog_parts
- id
- catalog_item_id
- name
- quantity
- notes
```

## 11.3. Designs e versões

Cada parte possui Designs versionados.

```text
design_versions
- id
- catalog_part_id
- version
- notes
- created_by
- created_at
```

Versões são imutáveis.

Alteração significativa gera nova versão.

## 11.4. Arquivos de design

Uma versão pode possuir:

```text
source
mesh
print
preview
documentation
other
```

Exemplos:

```text
STEP
FCStd
blend
STL
3MF
PNG
PDF
```

## 11.5. Procedência e licença

Campos:

```text
origin
source_url
original_author
license_name
commercial_use_allowed
attribution_required
attribution_text
notes
```

`origin`:

```text
original
customer
remix
third_party
unknown
```

Não bloquear uso interno por licença desconhecida.

Para itens `sellable=true`, a UI deve indicar quando a permissão comercial está desconhecida ou negada.

---

# 12. Inventário

## 12.1. Material

Representa o tipo de material:

```text
Voolt3D PLA Velvet Branco
```

Campos:

```text
materials
- id
- manufacturer
- name
- material_type
- color_name
- color_hex
- nominal_density
- default_replacement_cost_per_kg_cents
- notes
```

## 12.2. Bobina física

Cada bobina é um registro separado.

```text
material_spools
- id
- code
- material_id
- nominal_net_weight_g
- tare_weight_g
- gross_weight_at_open_g
- current_remaining_weight_g
- purchase_cost_cents
- replacement_cost_per_kg_cents
- opened_at
- last_weighed_at
- last_dried_at
- storage_location
- storage_status
- lot_number
- status
- created_at
- updated_at
```

Status:

```text
sealed
open
stored
drying
empty
retired
```

## 12.3. Pesagens

Não sobrescrever histórico.

```text
spool_measurements
- id
- spool_id
- measured_at
- gross_weight_g
- derived_remaining_weight_g
- source
- notes
- recorded_by
```

`source`:

```text
manual
imported
other
```

`current_remaining_weight_g` é estado derivado/cache atual.

## 12.4. Supplies

Itens não-filamento:

```text
NFC
argola
ímã
parafuso
cola
verniz
lixa
embalagem
etiqueta
```

```text
supplies
- id
- name
- sku
- unit
- current_quantity
- replacement_unit_cost_cents
- minimum_quantity
- notes
```

## 12.5. Movimentos

Estoque de supply deve ser auditável.

```text
supply_movements
- id
- supply_id
- type
- quantity
- unit_cost_cents
- reference_type
- reference_id
- occurred_at
- recorded_by
- notes
```

Tipos:

```text
purchase
consume
adjustment
return
discard
```

Não depender apenas de `current_quantity`.

---

# 13. BOM do catálogo

Um item de catálogo pode possuir componentes não impressos, seja para uso comercial ou interno.

```text
catalog_bom_items
- id
- catalog_item_id
- supply_id
- quantity_per_unit
- waste_percent
- notes
```

Exemplo Smart Pet Tag:

```text
1 × NFC
1 × argola
1 × embalagem
```

O filamento é calculado pela produção/Job e não precisa ser duplicado na BOM como supply.

---

# 14. Jobs

## 14.1. Conceito

Um Job representa uma execução física.

Campos mínimos:

```text
print_jobs
- id
- code
- catalog_item_id
- design_version_id
- printer_id
- purpose
- status
- planned_quantity
- good_quantity
- scrap_quantity
- hypothesis
- result_notes
- quality_status
- planned_seconds
- actual_seconds
- labor_minutes
- created_by
- created_at
- started_at
- completed_at
```

`purpose`:

```text
test
prototype
production
maintenance
internal
personal
```

`purpose` descreve por que a máquina foi usada, não se houve venda.

`production` pode representar produção para pedido, produção para estoque ou outra produção destinada a uso comercial futuro.

## 14.2. Status do Job

```text
draft
prepared
printing
awaiting_review
completed
failed
cancelled
```

Status de Job não deve ser igual ao estado da impressora.

## 14.3. Resultado de qualidade

```text
pending
approved
partial
failed
```

A impressora terminar não implica aprovação.

## 14.4. Uso de material

```text
print_job_material_usage
- id
- print_job_id
- material_id
- spool_id
- role
- planned_grams
- actual_grams
- planned_meters
- actual_meters
- measurement_source
- historical_material_cost_cents
- replacement_material_cost_cents
- created_at
- updated_at
```

`role`:

```text
model
support
purge
other
```

A mesma bobina pode aparecer mais de uma vez no mesmo Job para roles diferentes.

`measurement_source`:

```text
slicer
spool_weight_delta
manual
printer
estimated
```

## 14.5. Eventos relevantes

```text
job_events
- id
- job_id
- event_type
- occurred_at
- actor_user_id
- source_device_id
- metadata
```

Eventos úteis:

```text
created
prepared
printing_detected
printing_started_manual
finished_detected
reviewed
failed
cancelled
```

Não registrar telemetria de alta frequência aqui.

## 14.6. Jobs sem contexto comercial

Cliente, orçamento, pedido e venda são opcionais para um Job.

Exemplos válidos:

```text
purpose=test       → Benchy / calibração
purpose=prototype  → V1 de um produto
purpose=internal   → porta-sílica / ferramenta da oficina
purpose=personal   → mimo / projeto pessoal
purpose=maintenance→ peça produzida para manutenção
purpose=production → produção para estoque ou pedido
```

Um Job não comercial deve executar normalmente:

```text
consumo de bobina
+ movimentos de supplies
+ energia
+ horas de máquina
+ mão de obra
+ qualidade
+ custo real
+ Cost Snapshot opcional/fechamento
```

Não executar:

```text
receita
margem de venda
comissão de canal
resultado comercial
```

Dashboards financeiros devem classificar esse valor como `operational_consumption` ou equivalente, segmentado por `purpose`, e não como venda com receita zero/prejuízo.

Vínculo com `order_item` é nullable e só deve ser adicionado quando o módulo comercial realizar uma associação explícita.

---

# 15. Energia

## 15.1. Medições

```text
energy_measurements
- id
- job_id
- source
- meter_start_kwh
- meter_end_kwh
- measured_kwh
- estimated_average_power_w
- energy_rate_cents_per_kwh
- occurred_at
- recorded_by
- notes
```

`source`:

```text
manual_meter
smart_plug
estimated
imported
```

## 15.2. Regra de preferência

Para custo real:

1. medição real;
2. leitura inicial/final;
3. estimativa explícita.

Nunca esconder fatores como `× 0,5` dentro da fórmula.

Se estimativa usar fator de utilização, ele deve ser configuração explícita.

---

# 16. Impressoras e manutenção

## 16.1. Printer

Dados centrais não sensíveis:

```text
printers
- id
- name
- manufacturer
- model
- nozzle_diameter
- location
- acquisition_cost_cents
- residual_value_cents
- useful_life_hours
- maintenance_reserve_per_hour_cents
- status
- notes
```

Não armazenar access code Bambu no servidor.

## 16.2. Manutenção

```text
maintenance_events
- id
- printer_id
- type
- performed_at
- printer_hours
- description
- cost_cents
- downtime_minutes
- notes
- created_by
```

Tipos:

```text
cleaning
preventive
corrective
replacement
upgrade
inspection
```

---

# 17. Cost Engine

O Cost Engine responde:

> Quanto custa fabricar?

## 17.1. Componentes

```text
material
energy
machine
labor
supplies
consumables
post_processing
production_packaging
failure_risk
overhead
```

Cada componente permanece identificável.

## 17.2. Custo histórico vs. reposição

### Histórico

Usado para rentabilidade do que efetivamente aconteceu.

Exemplo:

> Bobina comprada por R$ 80.

### Reposição

Usado como referência para decisões futuras.

Exemplo:

> Bobina equivalente custa hoje R$ 110.

Ambos podem coexistir no cálculo.

## 17.3. Hora-máquina

Fórmula inicial:

```text
depreciation_per_hour =
(acquisition_cost - residual_value)
/
useful_life_hours
```

```text
machine_hour_cost =
depreciation_per_hour
+
maintenance_reserve_per_hour
```

Energia permanece separada.

Permitir override explícito de `machine_hour_cost` no futuro somente via parâmetro documentado.

## 17.4. Mão de obra — custo interno

O Cost Engine registra **quanto o tempo humano custa para a oficina**.

Isso é diferente do preço cobrado ao cliente pelo serviço.

Cobrar/custear tempo humano, não tempo total de impressão.

Categorias iniciais:

```text
setup
material_handling
support_removal
finishing
painting
assembly
packaging
modeling
customization
consulting
other
```

Modelo:

```text
labor_rates
- id
- name
- activity_type
- cost_hourly_rate_cents
- active
```

Job pode registrar minutos por atividade ou minutos totais + rate padrão.

### Assistente de custo da hora humana

A Release 1 deve ajudar a obter uma taxa interna realista sem exigir contabilidade avançada.

Entradas opcionais:

```text
target_monthly_compensation_cents
monthly_labor_overhead_cents
available_hours_per_month
productive_utilization_bps
```

Horas produtivas estimadas:

```text
productive_hours =
available_hours_per_month × productive_utilization
```

Custo interno sugerido da hora:

```text
internal_hourly_cost =
(target_monthly_compensation + monthly_labor_overhead)
/ productive_hours
```

`productive_utilization` existe porque nem toda hora disponível é faturável/produtiva.

A UI deve mostrar as premissas e permitir override manual.

A taxa efetivamente usada no Job deve ficar preservada no snapshot.

## 17.5. Custos por lote

Distinguir:

```text
per_batch
per_unit
```

Exemplo:

```text
setup: 10 min por lote
embalagem: 2 min por unidade
```

Custo unitário realizado:

```text
total_batch_cost / good_quantity
```

Não usar quantidade planejada quando houver refugos.

## 17.6. Risco de falha

Não aplicar percentual genérico sobre custos não expostos.

Separar:

```text
vulnerable_cost
non_vulnerable_cost
```

Se:

```text
p = probabilidade de falha
f = fração média do custo vulnerável consumida em uma falha
Cv = custo vulnerável de uma tentativa bem-sucedida
```

custo esperado vulnerável por unidade boa:

```text
Cv × [1 + f × p / (1 - p)]
```

quando `0 <= p < 1`.

Se uma falha média consome 100% do custo vulnerável:

```text
f = 1
```

e o resultado equivale a:

```text
Cv / (1 - p)
```

Componentes adicionados apenas depois da impressão não entram no risco da etapa de impressão.

Na Release 1, `p` e `f` são parâmetros configuráveis por contexto.

Automação estatística por histórico é futura.

## 17.7. Planned Cost

Antes da produção:

```text
planned material
planned time
estimated energy
planned labor
BOM
risk parameters
current replacement costs
```

## 17.8. Actual Cost

Após produção:

```text
actual material
actual machine time
actual energy
actual labor
consumed supplies
good quantity
actual failures/scrap
historical acquisition costs
```

## 17.9. Cost Snapshot

Ao fechar financeiramente um Job:

```text
cost_snapshots
- id
- job_id
- calculated_at
- currency
- material_cost_cents
- energy_cost_cents
- machine_cost_cents
- labor_cost_cents
- supplies_cost_cents
- packaging_cost_cents
- failure_cost_cents
- overhead_cost_cents
- total_cost_cents
- cost_per_good_unit_cents
- parameters_json
```

Snapshot é imutável.

Recalcular cria novo snapshot, preservando o anterior como histórico, ou exige ação explícita de reabertura.

## 17.10. Consumo operacional não comercial

Todo Job pode possuir custo real independentemente de venda.

Para Jobs sem vínculo comercial:

```text
Cost Snapshot = custo operacional real
Revenue = não aplicável
Commercial profit/loss = não aplicável
```

Relatórios devem permitir agrupar custos por `purpose`, por exemplo:

```text
produção comercial
protótipos
testes
uso interno
manutenção
uso pessoal
```

Não calcular margem comercial usando receita zero para esses Jobs.

---

# 18. Pricing Engine

O Pricing Engine responde, antes de qualquer venda:

> Quanto precisamos cobrar, em qual canal e sob quais premissas?

Ele pode ser usado diretamente a partir do catálogo.

Não exige cliente, orçamento, pedido ou venda.

## 18.1. Perfis de canal

A Release 1 não integra APIs de marketplace, mas deve permitir representar manualmente a estrutura econômica de cada canal.

Exemplos:

```text
Pix direto
Cartão direto
B2B
Revenda
Mercado Livre — perfil manual A
Mercado Livre — perfil manual B
Shopee — perfil manual
Outro marketplace
```

Um perfil representa uma combinação concreta de regras de cobrança.

Isso permite criar perfis diferentes quando um marketplace variar por categoria, modalidade de anúncio ou estratégia de frete.

Modelo inicial:

```text
sales_channel_profiles
- id
- name
- channel_type
- percent_fee_bps
- payment_fee_bps
- tax_bps
- sales_commission_bps
- fixed_fee_per_order_cents
- fixed_fee_per_item_cents
- shipping_subsidy_per_order_cents
- active
- effective_from
- effective_until
- source_note
- notes
```

Usar basis points para percentuais financeiros quando conveniente:

```text
100 bps = 1%
```

As taxas são cadastradas manualmente e identificadas como **premissas de precificação**, não como dados oficiais em tempo real.

Alterar uma estrutura relevante de taxa deve criar uma nova versão/perfil efetivo, preservando cálculos históricos por snapshot.

Sem integração de marketplace na Release 1.

## 18.2. Markup

```text
price = cost × (1 + markup)
```

A UI deve chamar isso de markup.

## 18.3. Margem sobre venda

```text
margin =
(price - all_variable_costs)
/
price
```

Não chamar markup de margem.

## 18.4. Preço para margem alvo

Se:

```text
C = custos atribuídos à unidade/lote
r = soma das taxas percentuais sobre preço
m = margem alvo sobre preço
F = taxas fixas atribuídas à venda
```

então:

```text
price =
(C + F)
/
(1 - r - m)
```

Somente válido quando:

```text
1 - r - m > 0
```

Caso contrário, retornar configuração economicamente impossível.

Taxas percentuais de marketplace, cartão, imposto ou comissão são calculadas sobre a base configurada de venda; nunca sobre o lucro bruto por conveniência.

## 18.5. Preços apresentados

O sistema deve poder mostrar:

```text
estimated_manufacturing_cost
actual_average_cost
replacement_cost_basis
break_even_price
minimum_price
target_margin_price
list_price
promotional_floor
market_reference
actual_sale_price
```

Nem todos precisam existir para todo item.

## 18.6. Price Versions

```text
product_price_versions
- id
- catalog_item_id
- sales_channel_profile_id
- unit_price_cents
- minimum_price_cents
- effective_from
- effective_until
- created_by
- notes
```

Uma análise de precificação só altera preço oficial quando o usuário escolhe explicitamente **Salvar como preço de tabela/versão**.

Vendas/pedidos preservam snapshot do preço usado.

## 18.7. Referência de mercado

Entrada manual na Release 1.

```text
market_price_references
- catalog_item_id
- source
- price_cents
- observed_at
- notes
```

Não automatizar scraping.

## 18.8. Área de Precificação do Catálogo

Todo `CatalogItem` vendável deve possuir uma área de **Precificação** independente do Comercial.

Objetivo:

> permitir estudar viabilidade e preço antes de existir cliente, orçamento, pedido ou venda.

Entradas principais:

```text
catalog item / design
quantity
cost basis
channel profile(s)
target margin
minimum margin
planned batch assumptions
optional labor/service assumptions
```

`cost basis` pode usar:

```text
planned replacement cost
latest valid planned cost
actual average historical cost
manual scenario override
```

Resultados por canal:

```text
manufacturing_cost
channel_percent_fees
channel_fixed_fees
break_even_price
minimum_price
target_margin_price
contribution_cents
effective_margin_bps
markup_bps
profit_per_machine_hour_when_available
```

A UI deve permitir selecionar vários perfis e comparar lado a lado, por exemplo:

```text
Direto/Pix       R$ ...
Mercado Livre A  R$ ...
Mercado Livre B  R$ ...
Shopee            R$ ...
B2B               R$ ...
```

Isso **não é integração com marketplace**.

É uma aplicação do Pricing Engine aos perfis de taxa cadastrados manualmente para orientar foco comercial.

O cálculo pode ser efêmero. Persistência só é obrigatória quando o usuário salva uma Price Version ou quando o resultado entra em um orçamento/pedido snapshot.

## 18.9. Precificação de mão de obra e serviços

O sistema deve separar:

```text
internal labor cost rate
≠
billable labor/service rate
```

A taxa interna vem do Cost Engine e representa quanto aquela hora custa para a oficina.

A taxa cobrada pode representar:

```text
modelagem CAD
personalização
acabamento
pintura
montagem
consultoria
outro serviço humano
```

Modelo:

```text
labor_pricing_profiles
- id
- name
- labor_rate_id
- billing_hourly_rate_cents
- minimum_billable_minutes
- rounding_increment_minutes
- target_margin_bps
- active
- notes
```

A Release 1 deve oferecer um assistente que mostre:

```text
custo interno por hora
preço mínimo da hora
preço da hora para margem alvo
preço da hora no canal selecionado
valor mínimo por serviço
```

O preço sugerido pode usar o mesmo solver de margem/canal do Pricing Engine.

Exemplo:

```text
Custo interno de modelagem: R$ 24/h
Canal direto, margem alvo 40%
Preço sugerido: R$ 40/h
Cobrança mínima: 30 min
```

Valores efetivamente usados em orçamento devem ser preservados em snapshot.

## 18.10. Serviços faturáveis

A Release 1 deve permitir precificar serviço humano mesmo quando não houver um item impresso separado.

Um orçamento pode conter:

```text
produto impresso
serviço de modelagem
serviço de personalização
acabamento
item customizado
```

Isso evita esconder trabalho criativo/técnico dentro da hora-máquina ou do markup do filamento.

---

# 19. Comercial — Release 1

O objetivo é permitir operação comercial básica sem criar um ERP fiscal.

O Comercial consome resultados do Pricing Engine, mas não é necessário para usar a precificação.

## 19.1. Clientes

```text
customers
- id
- type
- name
- document_optional
- email
- phone
- notes
- created_at
```

`type`:

```text
person
company
```

Documento é opcional e não será validado fiscalmente na Release 1.

## 19.2. Orçamentos são independentes de venda/pedido

Um orçamento é uma proposta comercial.

Ele pode existir até seu encerramento sem nunca virar pedido.

```text
quotes
- id
- code
- customer_id
- status
- valid_until
- currency
- notes
- created_by
- created_at
```

Status:

```text
draft
sent
accepted
rejected
expired
cancelled
```

Regras obrigatórias:

- `accepted` significa que a proposta foi aceita, não que um pedido foi automaticamente criado;
- orçamento aceito pode permanecer como orçamento;
- rejeição/expiração/cancelamento não gera venda ou prejuízo;
- nenhuma receita é registrada por existir ou aceitar orçamento;
- conversão para pedido é ação explícita e auditável;
- conversão não deve recalcular silenciosamente os valores aprovados.

## 19.3. Itens do orçamento

Um orçamento pode conter produto, serviço ou item customizado.

```text
quote_items
- id
- quote_id
- item_type
- catalog_item_id nullable
- labor_pricing_profile_id nullable
- description
- quantity
- billable_minutes nullable
- sales_channel_profile_id nullable
- unit_cost_snapshot_cents
- unit_price_cents
- discount_cents
- total_cents
- pricing_snapshot_json
```

`item_type`:

```text
product
service
custom
```

Para serviço, `billable_minutes` e perfil de mão de obra podem originar o valor sugerido, mas o preço final permanece editável dentro das permissões.

## 19.4. Orçamento exportável

Gerar PDF ou documento imprimível contendo:

- branding;
- cliente;
- itens;
- serviços;
- quantidade;
- preço;
- validade;
- observações;
- condições informadas manualmente.

PDF pode ser implementado após o fluxo funcional, mas faz parte do aceite da Release 1.

## 19.5. Pedidos manuais

Pedido é um compromisso operacional/comercial separado do orçamento.

Pode ser criado:

```text
manualmente
OU
por conversão explícita de orçamento
```

```text
orders
- id
- code
- customer_id
- quote_id nullable
- status
- notes
- created_by
- created_at
- completed_at
```

Status:

```text
draft
confirmed
in_production
ready
completed
cancelled
```

A Release 1 não possui ledger fiscal/financeiro de `sales`; pedido não deve ser tratado como pagamento recebido.

## 19.6. Itens do pedido

```text
order_items
- id
- order_id
- item_type
- catalog_item_id nullable
- description
- quantity
- unit_price_cents
- total_cents
- price_snapshot_json
```

Jobs podem ser vinculados a `order_item_id` quando o item exigir produção física.

Itens de serviço podem existir sem Job de impressão.

## 19.7. Relação entre precificação, orçamento e pedido

```text
Precificação do Catálogo
    ├── pode terminar aqui
    ├── pode salvar Price Version
    └── pode preencher um orçamento
                         ↓
                    Orçamento
                    ├── rejeitado/expirado/cancelado → termina
                    ├── aceito → pode terminar aqui
                    └── Converter explicitamente
                                ↓
                              Pedido
                                ↓
                        Jobs quando necessários
```

Nenhuma seta acima é automática, exceto cálculos internos da própria tela.

## 19.8. Fora do comercial da Release 1

Não implementar:

- cobrança;
- gateway;
- PIX automático;
- conciliação;
- contas a pagar/receber;
- emissão fiscal;
- NF-e;
- NFS-e;
- checkout;
- frete integrado;
- marketplace API;
- sincronização automática de taxas de marketplace;
- WhatsApp API;
- CRM avançado;
- funil automatizado.

---

# 20. Fluxo “Preparar impressão”

A ação preferida do catálogo será:

```text
Preparar impressão
```

Fluxo:

1. selecionar Catalog Item;
2. selecionar parte/design version quando necessário;
3. informar quantidade;
4. criar Job em `draft`;
5. selecionar impressora lógica;
6. selecionar material/bobina;
7. preencher/confirmar consumo planejado;
8. mudar Job para `prepared`;
9. baixar arquivo `print` (3MF preferencial);
10. validar hash;
11. abrir arquivo no Bambu Studio local.

O aplicativo **não** pressiona Print.

O operador continua no Bambu Studio.

Pode haver ações separadas:

```text
Criar Job
Abrir no Bambu Studio
```

mas `Preparar impressão` funciona como atalho seguro.

---

# 21. Bambu Studio Integration

## 21.1. Escopo

Windows-only.

O desktop deve:

- detectar caminho comum do Bambu Studio;
- permitir configuração manual do executável;
- baixar arquivo do servidor;
- manter cache local;
- validar SHA-256;
- abrir o arquivo no Bambu Studio.

## 21.2. Falhas

Se Bambu Studio não estiver instalado:

- não quebrar Job;
- informar claramente;
- permitir baixar/abrir pasta do arquivo.

## 21.3. Não fazer

Não automatizar cliques no Bambu Studio.

Não injetar código.

Não controlar impressão via UI automation.

---

# 22. Telemetria Bambu — final do MVP

## 22.1. Objetivo

Permitir que usuários autorizados saibam se a impressora está:

- online/offline;
- idle;
- imprimindo;
- pausada/estado equivalente quando disponível;
- com erro quando disponível.

E, durante impressão, exibir quando disponível:

- arquivo atual;
- progresso;
- camada atual;
- total de camadas;
- tempo restante;
- temperaturas relevantes.

## 22.2. Local-only collection

A coleta ocorre no desktop autorizado que está na LAN da impressora.

Credenciais permanecem locais.

## 22.3. Publicação

O desktop publica snapshot sanitizado no servidor:

```text
printer_current_state
- printer_id
- state
- current_file
- progress_percent
- current_layer
- total_layers
- remaining_seconds
- nozzle_temp
- bed_temp
- updated_at
- source_device_id
```

Não enviar access code.

## 22.4. Persistência

`printer_current_state` é estado atual mutável.

Histórico persistente apenas para eventos relevantes:

```text
online
offline
print_detected
print_finished
error
```

Não gravar amostra por segundo no PostgreSQL.

## 22.5. Visualização por outros usuários

Clientes desktop consultam o servidor periodicamente.

Polling inicial sugerido:

```text
5–10 segundos
```

Não introduzir WebSocket/SSE apenas para este requisito na Release 1.

## 22.6. Controle

Não existirão endpoints:

```text
/start
/pause
/resume
/cancel
```

para impressora.

Esta ausência é intencional.

---

# 23. Pendências

A UI terá uma visão derivada chamada:

```text
Pendências
```

Não criar tabela própria inicialmente.

Consultas possíveis:

- Jobs `awaiting_review`;
- Jobs sem energia quando exigida;
- Jobs sem Cost Snapshot;
- bobinas abaixo do limite;
- itens vendáveis com licença comercial desconhecida;
- orçamentos próximos do vencimento;
- itens com Price Version abaixo do novo preço mínimo calculado;
- perfis de marketplace com taxa vencida/sem data de revisão quando configurado;
- pedidos confirmados sem Jobs suficientes.

---

# 24. Dashboard da Release 1

Evitar dashboard ornamental.

Exibir apenas informação acionável.

## Agora

- Jobs ativos;
- Jobs aguardando revisão;
- status da impressora quando telemetria estiver disponível.

## Estoque

- bobinas críticas;
- supplies abaixo do mínimo.

## Comercial

- orçamentos pendentes;
- pedidos em produção;
- pedidos prontos;
- itens com preço de tabela abaixo do piso atual;
- comparação rápida de preço-alvo por perfis de canal selecionados.

## Financeiro operacional

- custo real médio de produtos com dados;
- margem estimada de pedidos;
- alertas de preço abaixo do mínimo;
- consumo operacional não comercial por purpose;
- horas e custo de uso interno/protótipo/teste separados de rentabilidade comercial.

Gráficos avançados ficam para depois de existir massa de dados.

---

# 25. API — princípios

## 25.1. Versionamento

Prefixo:

```text
/api/v1
```

## 25.2. Erros

Formato consistente:

```json
{
  "error": {
    "code": "catalog_item_not_found",
    "message": "Catalog item not found",
    "details": {}
  }
}
```

## 25.3. IDs

UUIDs.

## 25.4. Paginação

Endpoints de lista devem suportar paginação antes de crescerem indefinidamente.

## 25.5. Metadata

Endpoint:

```text
GET /api/v1/meta
```

retorna:

```text
api_version
server_version
workshop_name
minimum_desktop_version
```

Desktop deve detectar incompatibilidade de versão.

---

# 26. Observabilidade

Servidor:

- logs estruturados;
- request ID;
- user ID quando autenticado;
- sem secrets;
- sem tokens;
- sem access code Bambu.

Health:

```text
/health/live
/health/ready
```

Ready verifica:

- PostgreSQL;
- migrations;
- file storage.

Não depende de impressora.

Desktop:

- logs locais;
- botão/ação para abrir pasta de logs;
- secrets redigidos.

---

# 27. Migrations

Ferramenta preferida:

```text
goose
```

ou equivalente simples escolhido em DB-001.

Regras:

- migrations numeradas;
- migration publicada não é reescrita;
- falha de migration impede servidor de ficar ready;
- nenhuma alteração de schema sem migration;
- migrations testadas em banco vazio.

Aplicar advisory lock PostgreSQL durante migration se a ferramenta não fornecer lock adequado.

---

# 28. Backup e restore

Backup válido precisa cobrir:

```text
PostgreSQL
+
File Storage
```

## 28.1. Database

`pg_dump` em formato custom.

## 28.2. Files

Arquivo compactado ou snapshot do diretório de objetos.

## 28.3. Manifest

Backup contém manifesto com:

```text
created_at
server_version
database_backup
files_backup
object_count
database_file_record_count
checksums
```

## 28.4. Validação

Backup só é considerado funcional depois de:

1. criar backup;
2. restaurar PostgreSQL em ambiente limpo;
3. restaurar storage;
4. validar referências;
5. abrir arquivos amostrados;
6. executar testes de integridade.

## 28.5. Restore

Restore deve ocorrer com aplicação em modo de manutenção/parada.

Não restaurar o banco em uso pelo mesmo servidor.

---

# 29. Segurança da Release 1

Obrigatório:

- Argon2id;
- tokens opacos com hash no banco;
- revogação de sessão;
- RBAC no servidor;
- validação de upload;
- limite de tamanho configurável;
- nomes de arquivo não usados como caminho físico;
- proteção contra path traversal;
- SQL parametrizado;
- secrets fora do Git;
- PostgreSQL não publicado na LAN por padrão;
- CORS fechado por padrão, já que desktop não depende de browser cross-origin;
- rate limit no login;
- logs sem secrets.

LAN HTTP é aceito na Release 1 somente sob declaração de `trusted_lan`.

Antes de permitir deployment externo, TLS passa a ser obrigatório.

---

# 30. Internacionalização e formatação

Release 1 pode iniciar somente com `pt-BR` na UI.

Entretanto:

- strings visíveis devem estar preparadas para i18n;
- não armazenar texto monetário formatado;
- datas persistidas em UTC;
- apresentação usa timezone configurada;
- moeda guardada separadamente.

Não bloquear a Release 1 por tradução `en-US`.

---

# 31. Referências externas / Daedalus

Daedalus não é dependência arquitetural.

Pode ser consultado ou ter código MIT portado seletivamente quando isso reduzir custo de desenvolvimento.

Áreas candidatas:

- parsing 3MF;
- conceitos de design versioning;
- integração Bambu para leitura;
- file hashing/storage;
- padrões de Jobs.

Quando código for portado:

1. registrar origem;
2. registrar commit;
3. preservar aviso de licença aplicável;
4. incluir em `THIRD_PARTY_NOTICES.md`;
5. adaptar aos contratos deste produto;
6. não importar arquitetura desnecessária junto.

---

# 32. Fora de escopo da Release 1

Não implementar:

```text
web app
mobile app
SaaS
multi-tenant
sync offline
SQLite
controle remoto de impressora
controle local Start/Pause/Resume/Cancel
Bambu Cloud
AMS automation
upload automático para impressora
marketplace APIs
Nuvemshop
Mercado Livre API
Shopee API
WhatsApp API
checkout
pagamentos
PIX automático
NF-e
NFS-e
contabilidade completa
integração bancária
scraping de concorrentes
IA de precificação
IA de reparo de modelos
slicer próprio
Blender integration
custom themes
macOS
Linux desktop
```

---

# 33. Gates da implementação

## Gate 0 — Bootstrap

Objetivo: repositório executável e regras para agentes.

Saída:

- estrutura;
- CI;
- AGENTS;
- Docker dev;
- desktop shell.

## Gate 1 — Server Foundation

Objetivo: servidor + PostgreSQL + migrations + storage + health.

## Gate 2 — Auth & Settings

Objetivo: bootstrap admin, login, sessões, RBAC, workshop branding.

## Gate 3 — Desktop Foundation

Objetivo: login desktop, session storage seguro, navegação e client API.

## Gate 4 — Files & Catalog

Objetivo: catálogo, partes, designs, versões, arquivos e licenças.

## Gate 5 — Inventory

Objetivo: materiais, bobinas, pesagens, supplies e movimentos.

## Gate 6 — Jobs

Objetivo: criar/preparar/finalizar Jobs comerciais ou não comerciais e registrar consumo físico completo.

## Gate 7 — Cost Engine

Objetivo: custo planejado, realizado e snapshots.

## Gate 8 — Pricing & Commercial

Objetivo: precificação independente por canal, preço de mão de obra/serviços, clientes, orçamentos independentes e pedidos opcionais.

## Gate 9 — Bambu Studio

Objetivo: abrir arquivo correto localmente a partir do catálogo/Job.

## Gate 10 — Backup & Operational Hardening

Objetivo: backup/restore, integridade, logs e recuperação.

## Gate 11 — Bambu Telemetry

Objetivo: leitura local e visualização compartilhada sem controle remoto.

## Gate 12 — Production Validation

Objetivo: validar com operação real.

---

# 34. Critério de aceite da Release 1

A Release 1 é aceita quando for possível:

1. executar o servidor com `docker compose up -d`;
2. criar banco e aplicar migrations;
3. criar o primeiro administrador;
4. instalar o desktop no Windows;
5. conectar desktop ao servidor local;
6. fazer login;
7. visualizar branding da oficina;
8. criar usuários com níveis diferentes;
9. cadastrar catálogo;
10. cadastrar parte;
11. subir STEP/STL/3MF;
12. criar nova versão de design;
13. registrar procedência/licença;
14. cadastrar impressora lógica;
15. cadastrar material;
16. cadastrar bobina física;
17. registrar pesagem;
18. cadastrar supplies;
19. registrar movimento de estoque;
20. montar BOM simples;
21. criar Job sem cliente/orçamento/pedido;
22. escolher purpose `test`, `prototype`, `production`, `maintenance`, `internal` ou `personal`;
23. relacionar design ao Job;
24. selecionar bobina;
25. registrar consumo planejado;
26. usar “Preparar impressão”;
27. abrir 3MF correto no Bambu Studio;
28. registrar Job como iniciado/concluído manualmente sem telemetria;
29. registrar consumo real;
30. registrar energia;
31. registrar mão de obra;
32. avaliar qualidade;
33. gerar custo planejado;
34. gerar custo real;
35. gerar Cost Snapshot de Job não comercial sem classificá-lo como prejuízo comercial;
36. cadastrar perfil de canal de venda;
37. cadastrar manualmente um perfil de taxas de marketplace;
38. calcular preço de equilíbrio;
39. calcular preço para margem alvo;
40. distinguir margem e markup;
41. comparar o preço necessário do mesmo item em pelo menos dois canais;
42. analisar precificação diretamente pelo catálogo sem cliente/orçamento/pedido;
43. salvar explicitamente um resultado como Price Version;
44. calcular economia de lote;
45. configurar custo interno da hora humana;
46. usar o assistente de custo da hora humana;
47. configurar preço cobrado por modelagem/personalização ou outro serviço;
48. calcular preço de serviço para margem/canal selecionado;
49. cadastrar cliente;
50. gerar orçamento com produto e/ou serviço;
51. aceitar/rejeitar/expirar orçamento sem criar pedido automaticamente;
52. converter orçamento em pedido somente por ação explícita;
53. criar pedido manual sem orçamento;
54. relacionar Jobs ao pedido quando necessário;
55. visualizar rentabilidade do pedido sem misturar Jobs não comerciais;
56. exportar orçamento;
57. criar backup completo;
58. restaurar backup com sucesso;
59. configurar integração local Bambu;
60. coletar telemetria;
61. outro desktop visualizar status recente da impressora;
62. comprovar que o servidor não possui endpoint de controle da impressora.

---

# 35. Validação de produção

Antes de considerar a Release 1 estável:

Registrar pelo menos:

```text
20 Jobs
```

incluindo:

- sucesso;
- falha;
- protótipo;
- uso interno ou pessoal;
- produção sem pedido;
- produção vinculada a pedido;
- PLA;
- PETG;
- purga;
- suporte;
- mais de uma versão de design;
- pesagem real;
- energia real;
- Job de lote;
- pedido comercial;
- orçamento que não virou pedido;
- comparação de precificação entre canais;
- serviço de mão de obra/modelagem precificado.

Comparar:

```text
planned_cost
vs
actual_cost
```

e registrar divergências.

---

# 36. Métricas iniciais

Não criar sistema de analytics complexo.

Métricas úteis:

```text
job_success_rate
material_planned_vs_actual
planned_cost_vs_actual
cost_per_good_unit
machine_hours
material_waste_grams
average_margin_by_product
average_margin_by_channel
operational_consumption_by_job_purpose
internal_machine_hours_by_purpose
channel_target_price_difference
channel_contribution_difference
labor_internal_cost_vs_billable_rate
quote_acceptance_rate
quote_to_order_conversion_rate
open_order_count
```

Métricas que ainda não possuem dados suficientes podem ficar sem dashboard.

---

# 37. Requisitos de qualidade

## Backend

- testes unitários em regras financeiras;
- testes de repository contra PostgreSQL real;
- testes de migrations;
- testes de auth;
- testes de autorização;
- testes de file storage;
- `go test`;
- race detector nos pacotes aplicáveis;
- lint.

## Desktop

- TypeScript strict;
- lint;
- build;
- testes de funções críticas;
- testes manuais documentados para Bambu Studio e telemetry.

## Financeiro

Testes obrigatórios para:

- centavos;
- arredondamento;
- markup;
- margem;
- taxa percentual;
- taxa fixa;
- combinação taxa + margem;
- denominador inválido;
- lote;
- good quantity zero;
- risco de falha;
- custo histórico vs reposição;
- snapshot imutável;
- Job não comercial não gera margem/prejuízo comercial;
- taxa de marketplace aplicada à base correta;
- comparação multi-canal;
- cálculo de custo interno de mão de obra;
- cálculo de preço faturável de mão de obra;
- orçamento aceito não cria pedido implicitamente.

---

# 38. Política de arredondamento

Dinheiro é calculado internamente de forma que evite floating point binário.

Percentuais:

- basis points ou decimal exato;
- arredondamento monetário somente nos limites definidos.

Regra inicial:

- cálculos intermediários preservam precisão;
- valores finais monetários persistidos em centavos;
- método de arredondamento documentado em `ADR-FIN-001`.

---

# 39. Definition of Done de qualquer task

Uma task só está concluída quando:

```text
[ ] objetivo da task implementado
[ ] critérios de aceite cobertos
[ ] migration criada quando necessária
[ ] testes adicionados/atualizados
[ ] testes relevantes passam
[ ] lint passa
[ ] build afetado passa
[ ] nenhuma credencial incluída
[ ] documentação afetada atualizada
[ ] nenhum arquivo não relacionado foi alterado sem justificativa
[ ] nenhuma decisão fechada foi alterada silenciosamente
[ ] task não deixou TODO obrigatório para funcionar
```

---

# 40. Estratégia de baixo custo para agentes

Para reduzir tempo, tokens e regressões:

1. não pedir módulos inteiros em uma task;
2. preferir vertical slices pequenos;
3. fornecer ao agente somente PRD + task + arquivos relevantes;
4. evitar refatoração preventiva;
5. não criar abstração sem segundo consumidor real, exceto fronteiras arquiteturais definidas;
6. criar schema mínimo necessário para a task;
7. reutilizar tipos e helpers existentes;
8. separar regras financeiras puras de I/O para facilitar teste;
9. adicionar integração externa somente depois do fluxo manual existir;
10. validar cada Gate antes de iniciar o seguinte quando o próximo depende estruturalmente dele.

---

# 41. North Star

O produto é considerado tecnicamente saudável quando consegue provar, de ponta a ponta:

```text
arquivo/versionamento
        ↓
material físico
        ↓
Job físico (comercial ou não)
        ↓
consumo real
        ↓
custo real
        ├──────────────► consumo operacional por purpose
        │
        └──► precificação por canal
                  ├──► preço de tabela
                  ├──► orçamento opcional
                  │         └──► pedido opcional
                  └──► decisão de onde vender
```

sem exigir que o usuário mantenha uma planilha paralela para confiar nos números.

---

# 42. Decisão final da Release 1

A Release 1 não tenta substituir:

- Bambu Studio;
- Bambu Handy;
- ERP fiscal;
- marketplace;
- software contábil.

Ela **já calcula preços usando perfis manuais de taxas de marketplaces**, mas não integra, publica anúncios, sincroniza pedidos ou consulta taxas automaticamente.

Ela cria a camada que falta entre essas ferramentas:

> **a fonte de verdade operacional, técnica, financeira e comercial da oficina.**
