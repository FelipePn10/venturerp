# Módulo de Romaneio (Expedição) — Documentação

Cobre o módulo de **romaneio de expedição/carregamento** para pedidos de venda,
compra e produção. Modela a logística de saída no padrão de um *outbound
delivery* de ERP de ponta (SAP/Oracle): **separação → conferência → packing em
volumes → despacho**, com **reserva de estoque**, **dados de transporte**,
**vínculo com a NF-e** e **exportação profissional** (PDF/Excel).

> Índice geral da documentação: **visao-geral.md** (mesma pasta `docs/dev/`).
> Branding dos documentos: motor `pdfkit` (ver `internal/infrastructure/export/pdfkit`).

---

## 1. O que é o Romaneio

O romaneio (*packing list* / *delivery note*) acompanha a mercadoria no
transporte: lista itens, quantidades, **pesos líquido e bruto**, **volumes**
(embalagens) com dimensões e cubagem, dados da transportadora e serve de
documento de conferência entre emitente, transportador e destinatário.

É um documento **logístico/operacional** — não substitui a NF-e. No nosso
desenho a **baixa de estoque é fiscal** (ocorre na autorização da NF-e de
saída); o romaneio apenas **reserva** o estoque na separação. Isso espelha o
fluxo SAP *Outbound Delivery → Post Goods Issue*: a delivery reserva/aloca e o
PGI baixa o estoque.

### Tipos suportados (referência polimórfica)

| Tipo | Referência | Uso |
|------|-----------|-----|
| `SALES_ORDER` | Pedido de Venda | Expedição ao cliente |
| `PURCHASE_ORDER` | Pedido de Compra | Devolução/retorno a fornecedor |
| `PRODUCTION_ORDER` | Ordem de Produção | Movimentação de acabados |

---

## 2. Modelo de Dados

```
shipment_sequences   ← gerador de código sequencial
shipments            ← cabeçalho (status, pesos, transporte, NF-e)
  ├─ shipment_items   ← itens (qtd planejada, conferida, peso unitário)
  ├─ shipment_volumes ← volumes / handling units (packing)
  └─ shipment_events  ← trilha de auditoria das transições
```

Migrations: `000146` (base), `000167` (referência polimórfica),
**`000169` (modelo profissional: pesos líq/bruto, cubagem, transporte, NF-e,
volumes, eventos)**.

### `shipments` — cabeçalho

| Campo | Tipo | Descrição |
|-------|------|-----------|
| `code` | BIGINT | Código sequencial (PK lógica) |
| `reference_type` / `*_order_code` | — | Pedido de origem (venda/compra/produção) |
| `carrier_code` | BIGINT | Transportadora (mestre) |
| `status` | VARCHAR | `OPEN` `SEPARATED` `CONFERRED` `SHIPPED` `CANCELLED` |
| `total_volumes` | INT | Qtde de volumes (recalculada dos volumes) |
| `total_net_weight` / `total_gross_weight` | NUMERIC | **Peso líquido e bruto** (distintos) |
| `total_cubage_m3` | NUMERIC | Cubagem total (m³) |
| `freight_modality` | VARCHAR | `CIF` `FOB` `TERCEIROS` `SEM_FRETE` |
| `freight_value` / `insurance_value` | NUMERIC | Frete e seguro (R$) |
| `vehicle_plate` / `driver_name` / `driver_document` / `antt_code` | — | Dados da viagem |
| `seals` | VARCHAR | Lacres (separados por vírgula) |
| `estimated_delivery` | DATE | Previsão de entrega |
| `fiscal_exit_id` / `nfe_number` / `nfe_key` | — | Vínculo com a NF-e do carregamento |
| `separated_at` / `conferred_at` / `shipped_at` / `cancelled_at` | TIMESTAMPTZ | Carimbos das transições |
| `created_by` / `updated_by` | UUID | Autoria (vem do usuário autenticado) |

### `shipment_items` — itens

| Campo | Descrição |
|-------|-----------|
| `quantity` | Quantidade **planejada** a expedir |
| `conferred_qty` / `is_conferred` | Quantidade **conferida** (real) e flag |
| `unit_net_weight` / `unit_gross_weight` | Peso unitário (para recalcular totais) |
| `warehouse_id` | Depósito de origem (base da reserva) |

> **Divergência** = `is_conferred && conferred_qty ≠ quantity` (sobra/falta).
> Exposta em `has_divergence` na API e **bloqueia o despacho** salvo aceite
> explícito.

### `shipment_volumes` — volumes / handling units

| Campo | Descrição |
|-------|-----------|
| `volume_number` | Número do volume |
| `package_type` | `CAIXA` `PALLET` `FARDO` `ENGRADADO` `BOBINA` `SACO` `TAMBOR` `AMARRADO` |
| `net_weight` / `gross_weight` | Pesos do volume |
| `length_cm` / `width_cm` / `height_cm` | Dimensões |
| `cubage_m3` | Cubagem (calculada de LxAxC se não informada) |
| `marking` / `contents` | Marca/identificação e conteúdo |

### `shipment_events` — auditoria

Uma linha por transição: `event` (`CREATED`/`SEPARATED`/`CONFERRED`/`SHIPPED`/
`CANCELLED`/`TRANSPORT`/`NFE_LINKED`), `note`, `created_by`, `created_at`.

---

## 3. Ciclo de Vida (máquina de estado)

```
            separar              conferir            despachar
  OPEN ─────────────► SEPARATED ──────────► CONFERRED ──────────► SHIPPED
   │   (reserva estoque)   │   (todos itens     │  (sem divergência    (terminal)
   │                       │     conferidos)    │     ou aceite)
   └───────────────────────┴────────────────────┴──► CANCELLED (libera reservas)
```

Transições válidas (recusadas fora disso, com erro claro):

| De | Para |
|----|------|
| `OPEN` | `SEPARATED`, `CANCELLED` |
| `SEPARATED` | `CONFERRED`, `OPEN`, `CANCELLED` |
| `CONFERRED` | `SHIPPED`, `SEPARATED`, `CANCELLED` |
| `SHIPPED` / `CANCELLED` | — (terminais) |

Regras de cada passo:
- **Separar** (`OPEN→SEPARATED`): exige itens; **reserva o estoque** (uma
  `stock_reservation` ACTIVE por item, referência `SHIPMENT`/código).
- **Conferir** (`SEPARATED→CONFERRED`): exige **todos** os itens conferidos.
- **Despachar** (`CONFERRED→SHIPPED`): valida conferência e **divergências**;
  **consome** as reservas (a NF-e faz a baixa real). Carimba `shipped_at`.
- **Cancelar**: só antes de despachado; **cancela** as reservas.

`AddItem` só é aceito em `OPEN`/`SEPARATED`. `ConferItem` é bloqueado em
`SHIPPED`/`CANCELLED`. A máquina de estado vive em
`entity.ShipmentStatus.CanTransitionTo`.

---

## 4. Integração com Estoque (reserva, não baixa)

| Evento | Efeito no estoque |
|--------|-------------------|
| Separar romaneio | Cria **reserva** ACTIVE por item (`StockReserver.CreateReservation`) |
| Despachar | **Consome** as reservas do romaneio |
| Cancelar | **Cancela** as reservas do romaneio |
| Emitir NF-e (saída) | **Baixa real** do estoque (movimento `OUT`) — fora deste módulo |

A reserva reduz o **disponível** (ATP) sem mexer no físico; a NF-e reduz o
**físico**. Isso evita estoque vendido em duplicidade. O acoplamento usa uma
interface estreita (`shipment_uc.StockReserver`) — o `StockRepository`
compartilhado não é alterado. Reservas são *best-effort*: uma falha de estoque
não trava a transição do romaneio.

---

## 5. API — Endpoints

Base: `/api/shipments` (papéis `ADMIN`/`USER`). `created_by`/`updated_by` vêm do
**usuário autenticado** (JWT), nunca do corpo do request.

### Cadastro e itens
| Método | Rota | Descrição |
|--------|------|-----------|
| `POST` | `/` | Cria romaneio (`reference_type`, pesos líq/bruto, volumes, notas) |
| `GET` | `/` | Lista com filtros `?status=&carrier_code=&from=&to=&limit=&offset=` |
| `GET` | `/{code}` | Detalhe (itens + volumes) |
| `POST` | `/{code}/items` | Adiciona item (com `unit_net_weight`/`unit_gross_weight`) |
| `POST` | `/{code}/items/confer` | Confere item (`item_id`, `conferred_qty`) |

### Ciclo de vida
| Método | Rota | Corpo |
|--------|------|-------|
| `POST` | `/{code}/separate` | — (reserva estoque) |
| `POST` | `/{code}/confer` | — |
| `POST` | `/{code}/ship` | `{ "accept_divergences": false }` |
| `POST` | `/{code}/cancel` | `{ "reason": "..." }` |

### Transporte, volumes, NF-e, auditoria
| Método | Rota | Descrição |
|--------|------|-----------|
| `PUT` | `/{code}/transport` | Frete/modalidade/placa/motorista/ANTT/lacres/`estimated_delivery` |
| `POST` | `/{code}/volumes` | Adiciona volume (cubagem auto de LxAxC) |
| `GET` | `/{code}/volumes` | Lista volumes |
| `DELETE` | `/{code}/volumes/{volumeID}` | Remove volume |
| `POST` | `/{code}/nfe-link` | Liga NF-e (`fiscal_exit_id`, `nfe_number`, `nfe_key`) |
| `GET` | `/{code}/events` | Trilha de auditoria |

### Auto-fill e exportação
| Método | Rota | Descrição |
|--------|------|-----------|
| `POST` | `/auto-fill/sales-order` | Cria romaneio do pedido de venda (`{sales_order_code}`) |
| `POST` | `/auto-fill/purchase-order` | Idem compra (`{purchase_order_code}`) |
| `POST` | `/auto-fill/production-order` | Idem produção (`{production_order_code}`) |
| `GET` | `/{code}/export/pdf` | Romaneio em PDF profissional |
| `GET` | `/{code}/export/xlsx` | Romaneio em Excel |

> Os payloads de auto-fill **não** levam mais `created_by` — a autoria vem do JWT.

### Exemplos

```bash
# Despachar aceitando divergência de conferência
curl -X POST http://localhost:5072/api/shipments/1042/ship \
  -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d '{"accept_divergences": true}'

# Registrar dados da viagem
curl -X PUT http://localhost:5072/api/shipments/1042/transport \
  -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d '{"freight_modality":"CIF","freight_value":450,"insurance_value":120,
       "vehicle_plate":"ABC1D23","driver_name":"José da Silva","antt_code":"123456789",
       "seals":"LCR-0091, LCR-0092","estimated_delivery":"2026-06-28"}'

# Adicionar um volume (pallet)
curl -X POST http://localhost:5072/api/shipments/1042/volumes \
  -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d '{"volume_number":1,"package_type":"PALLET","net_weight":520,"gross_weight":545.5,
       "length_cm":120,"width_cm":100,"height_cm":80,"marking":"M-01"}'
```

---

## 6. Exportação (PDF / Excel)

O PDF usa o motor **`pdfkit`** (mesmo padrão dos demais relatórios): letterhead
com logo + cor da marca da empresa, seções emolduradas (**Dados / Destinatário /
Transportadora**, com NF-e e lacres), **tabela de itens** (header colorido,
zebra), **tabela de VOLUMES** (espécie, peso líq/bruto, dimensões, cubagem,
marca), bloco de **totais** (peso líquido ≠ bruto, frete, seguro, previsão),
**assinaturas** (Emitente/Transportadora/Destinatário) e rodapé paginado.

Os dados são enriquecidos pelo `RomaneioEnricherAdapter` (empresa+logo+cor do
`fiscal_configs`; destinatário/transportadora/itens dos respectivos mestres) e
sobrepostos pelos **dados de viagem persistidos** no romaneio (placa, motorista,
frete, lacres, previsão). Fontes base-14 Helvetica com **WinAnsiEncoding** (acentos).

```bash
curl -o romaneio_1042.pdf http://localhost:5072/api/shipments/1042/export/pdf -H "Authorization: Bearer $TOKEN"
curl -o romaneio_1042.xlsx http://localhost:5072/api/shipments/1042/export/xlsx -H "Authorization: Bearer $TOKEN"
```

---

## 7. Fluxo recomendado (ponta a ponta)

1. `auto-fill/*` (ou `POST /`) cria o romaneio em `OPEN` a partir do pedido.
2. `POST /{code}/separate` → reserva estoque, vai para `SEPARATED`.
3. Conferência física: `POST /{code}/items/confer` por item (divergências ficam
   registradas).
4. `POST /{code}/confer` → `CONFERRED` (exige todos conferidos).
5. Packing: `POST /{code}/volumes` para cada embalagem; `PUT /{code}/transport`
   com a viagem.
6. Emite a NF-e (módulo fiscal) → baixa o estoque; `POST /{code}/nfe-link` amarra
   a nota.
7. `POST /{code}/ship` → consome reservas, `SHIPPED`. Imprime o PDF.

---

## 8. Arquitetura

```
internal/
├── domain/shipment/
│   ├── entity/entity.go            # Shipment, ShipmentItem, ShipmentVolume,
│   │                               #   ShipmentEvent, máquina de estado, divergência
│   └── repository/repository.go    # ShipmentRepository, ShipmentFilter, TransportInput
├── application/usecase/shipment_uc/
│   ├── shipment_uc.go              # estado + reserva de estoque + volumes + NF-e
│   ├── auto_fill_uc.go             # auto-fill de pedidos
│   ├── export_uc.go                # montagem do RomaneioData
│   └── response_mapper.go          # entity → DTO
├── infrastructure/
│   ├── repository/shipment/
│   │   ├── shipment_repository_pg.go   # PostgreSQL (pgx)
│   │   ├── adapters.go                 # SalesOrder/PurchaseOrder/ProductionOrder readers
│   │   └── romaneio_enricher.go        # empresa/branding/parties/itens p/ o PDF
│   └── export/romaneio/                # romaneio_pdf.go / romaneio_xlsx.go / romaneio_data.go
└── interfaces/http/handler/shipment_handler.go
```

Princípios: Clean Architecture; interfaces segregadas (`StockReserver`,
`*Reader`); exportadores zero-dependências; `created_by`/`updated_by` do contexto
autenticado.

---

## 9. Testes

```bash
go test ./internal/domain/shipment/... \
        ./internal/application/usecase/shipment_uc/... \
        ./internal/infrastructure/export/romaneio/... \
        ./internal/infrastructure/repository/shipment/...
```

Cobrem: máquina de estado (transições válidas/inválidas), separação+reserva,
conferência, despacho com bloqueio de divergência (e aceite), cancelamento,
auto-fill e estrutura do PDF/XLSX.

Integração: `make test-romaneio` / `./scripts/test-romaneio.sh`.

---

## 10. Paralelo com SAP / Oracle

| Conceito | SAP | Aqui |
|----------|-----|------|
| Documento de saída | Outbound Delivery | `shipments` |
| Picking/Separação | Picking | `SEPARATED` + reserva |
| Packing | Handling Units (VERP) | `shipment_volumes` |
| Baixa de estoque | Post Goods Issue | NF-e de saída (fiscal) |
| Conferência/Divergência | overdelivery check | `conferred_qty` + `has_divergence` |
| Auditoria | Document flow | `shipment_events` |

---

## 11. Roadmap / Melhorias Futuras

- [ ] Saldo "a expedir" por item do pedido (remessas parciais com bloqueio de sobre-expedição)
- [ ] Código de barras / QR Code no PDF para leitura no depósito
- [ ] Envio do romaneio por e-mail ao cliente/transportadora
- [ ] Integração CT-e (vincular ao Conhecimento de Transporte)
- [ ] Foto do produto (`PhotoURL` já existe em `RomaneioItem`)
