# Fluxo Completo Tecnofer — ESI-500 / ESI-750

> Cenário real executado pelo `scripts/test-flow.sh` (153/153 checks — 0 falhas — 0 avisos).  
> Empresa: **TECNOFER** · CNPJ `52454668000102` · IE `9103144679` · Mandaguari–PR  
> Ambiente: **homologação** SEFAZ (FocusNFE token `YvCRephh2ZiwEmpawutMj83uPJxAYMD9`)

---

## Estrutura do Produto (BOM 3 Níveis)

```
LLC 0  PA-001 · 50001 ESI-500 Estante Industrial 500 kg
         ├─ PI-001 · 51001 Coluna 3m         × 2 UN
         │    ├─ MP-001 · 52001 Chapa Aço 3mm   × 12 KG  (+8% perda)
         │    └─ MP-002 · 52002 Perf.Tubular 100 × 3 M   (+5%)
         ├─ PI-002 · 51002 Viga 2m ★COMPARTILHADA × 2 UN
         │    ├─ MP-001 · 52001 Chapa Aço 3mm   × 8 KG   (+6%)
         │    └─ MP-003 · 52003 Perfil I 200mm  × 2 M    (+5%)
         ├─ PI-003 · 51003 Base Padrão           × 1 UN
         │    ├─ MP-001 · 52001 Chapa Aço 3mm   × 6 KG   (+5%)
         │    └─ MP-004 · 52004 Parafuso M8×25  × 8 UN   (+2%)
         ├─ MP-004 · 52004 Parafuso M8×25        × 16 UN  (+2%)  ← montagem PA
         └─ MP-005 · 52005 Porca M8              × 16 UN  (+2%)

LLC 0  PA-002 · 50002 ESI-750 Estante Industrial 750 kg
         ├─ PI-004 · 51004 Coluna 4m             × 2 UN
         │    ├─ MP-001 · 52001 Chapa Aço 3mm   × 16 KG  (+8%)
         │    └─ MP-002 · 52002 Perf.Tubular 100 × 4 M   (+5%)
         ├─ PI-002 · 51002 Viga 2m ★COMPARTILHADA × 3 UN  ← mesma peça!
         ├─ PI-005 · 51005 Base Reforçada         × 1 UN
         │    ├─ MP-001 · 52001 Chapa Aço 3mm   × 10 KG  (+5%)
         │    └─ MP-006 · 52006 Parafuso M10×30 × 12 UN  (+2%)
         ├─ MP-006 · 52006 Parafuso M10×30        × 20 UN  (+2%)  ← montagem PA
         └─ MP-007 · 52007 Porca M10              × 20 UN  (+2%)
```

**Peça compartilhada**: PI-002 Viga 2m é filha tanto do ESI-500 (×2) quanto do ESI-750 (×3).  
O MRP agrega a demanda: `5×2 + 3×3 = 19 UN` numa única sugestão.

---

## Passo 1 — Pedido de Venda (19/06/2026)

| Campo | Valor |
|-------|-------|
| Código | SO-1 |
| Cliente | Construtora GrandeObra SP Ltda · CNPJ `55444333000177` |
| Status | `P` (Confirmado) |
| Emissão | 2026-06-19 |
| Entrega | 2026-08-15 (ESI-500) / 2026-08-20 (ESI-750) |

**Itens:**

| Seq | Código | Produto | Qtd | Preço Unit. | Total | ICMS |
|-----|--------|---------|-----|-------------|-------|------|
| 1 | 50001 | ESI-500 | 5 UN | R$ 4.500,00 | **R$ 22.500,00** | 12% |
| 2 | 50002 | ESI-750 | 3 UN | R$ 6.800,00 | **R$ 20.400,00** | 12% |
| | | | | **TOTAL** | **R$ 42.900,00** | |

---

## Passo 2 — Planejamento da Demanda

### Demandas Independentes (Seção 13)

| Código | Item | Qtd | Data |
|--------|------|-----|------|
| DI-5001 | 50001 ESI-500 | 5 UN | 2026-08-15 |
| DI-5002 | 50002 ESI-750 | 3 UN | 2026-08-20 |

### Previsão de Vendas (Seção 12)

| Item | Semana | Ano | Qtd Prevista |
|------|--------|-----|--------------|
| ESI-500 | 32 | 2026 | 8 UN |
| ESI-750 | 32 | 2026 | 4 UN |
| ESI-500 | 34 | 2026 | 10 UN |
| ESI-750 | 34 | 2026 | 5 UN |

### Plano de Produção

| Campo | Valor |
|-------|-------|
| Código | **2001** |
| Nome | Plano Tecnofer Q3-2026 |
| Modo demanda | `ALL` (independente + forecast) |
| Tipo | MRP |

---

## Passo 3 — Explosão MRP (Seção 14)

`POST /api/mrp-calculation/run` · `plan_code=2001` · `generate_llc=true`

O MRP gerou **16 sugestões** na tabela `mrp_planned_suggestions`:

### Sugestões FABRICACAO (7 itens — PA + PI)

| Código Sugestão | Item | Tipo | LLC | Qtd |
|-----------------|------|------|-----|-----|
| 1 | 50002 ESI-750 | FABRICACAO | 0 | 15 |
| 2 | 50001 ESI-500 | FABRICACAO | 0 | 28 |
| 3¹ | 51001 Coluna 3m | FABRICACAO | 1 | — |
| 4 | 51002 Viga 2m ★ | FABRICACAO | 1 | 19 ← agregada |
| 5 | 51003 Base Padrão | FABRICACAO | 1 | — |
| 6 | 51004 Coluna 4m | FABRICACAO | 1 | — |
| 7 | 51005 Base Reforçada | FABRICACAO | 1 | — |

¹ Inclusão da perda no cálculo eleva quantidades acima do nominal.

### Sugestões COMPRA (9 entradas — MPs)

| Código Sugestão | Item | Qtd MRP | UOM | Necessidade |
|-----------------|------|---------|-----|-------------|
| (sugg 3) | 52007 Porca M10 | **306,12** | UN | 2026-08-03 |
| (sugg 7) | 52005 Porca M8 | **457,14** | UN | 2026-08-03 |
| (sugg 9) | 52006 Parafuso M10 | **306,12** | UN | 2026-08-03 |
| (sugg 10) | 52004 Parafuso M8 | **457,14** | UN | 2026-08-03 |
| (sugg 12) | 52002 Perfil Tubular | **303,16** | M | 2026-08-03 |
| (sugg 13) | 52004 Parafuso M8 ² | **228,57** | UN | 2026-08-03 |
| (sugg 14) | 52006 Parafuso M10 ² | **183,67** | UN | 2026-08-03 |
| (sugg 15) | 52001 Chapa Aço 3mm | **2.446,49** | KG | 2026-08-03 |
| (sugg 16) | 52003 Perfil I 200mm | **212,63** | M | 2026-08-03 |

² MP usada em múltiplos PIs e diretamente no PA — MRP gera sugestão separada por cada caminho de demanda.

---

## Passo 4 — Firmar Sugestões MRP (Seção 16)

`POST /api/mrp-calculation/suggestions/{code}/firm`

As 9 sugestões COMPRA foram convertidas em **ordens planejadas** na tabela `planned_orders` com `order_type=PURCHASE`:

| planned_code | order_number | item_code | Qtd | Tipo |
|--------------|--------------|-----------|-----|------|
| 1 | 1 | 52007 Porca M10 | 306,12 UN | PURCHASE |
| 2 | 2 | 52005 Porca M8 | 457,14 UN | PURCHASE |
| 3 | 3 | 52006 Parafuso M10 | 306,12 UN | PURCHASE |
| 4 | 4 | 52004 Parafuso M8 | 457,14 UN | PURCHASE |
| 5 | 5 | 52002 Perfil Tubular | 303,16 M | PURCHASE |
| 6 | 6 | 52004 Parafuso M8 (2ª) | 228,57 UN | PURCHASE |
| 7 | 7 | 52006 Parafuso M10 (2ª) | 183,67 UN | PURCHASE |
| 8 | 8 | 52001 Chapa Aço | 2.446,49 KG | PURCHASE |
| 9 | 9 | 52003 Perfil I | 212,63 M | PURCHASE |

> **Fix aplicado**: `FirmarSugestaoMRPUseCase` precisava mapear `"DEPENDENTE"→REPLENISHMENT`  
> e `"COMPRA"→PURCHASE` antes de gravar no enum `demand_type_enum` / `order_type_enum`.

---

## Passo 5 — Pedidos de Compra (Seção 17)

### PO-1 — Siderúrgica ParanaAco (Perfis e Chapas)

| Item | Descrição | Qtd | Preço Unit. | Total |
|------|-----------|-----|-------------|-------|
| 52001 | Chapa Aço 3mm SAE1020 | 600 KG | R$ 7,80 | **R$ 4.680,00** |
| 52002 | Perfil Tubular 100×100mm | 60 M | R$ 42,50 | **R$ 2.550,00** |
| 52003 | Perfil I 200mm | 50 M | R$ 68,00 | **R$ 3.400,00** |
| | | | **PO-1 Total** | **R$ 10.630,00** |

### PO-2 — MetalFix (Fixadores M8 e M10)  
CNPJ: `22334455000186`

| Item | Descrição | Qtd | Preço Unit. | Total |
|------|-----------|-----|-------------|-------|
| 52004 | Parafuso M8×25 Zincado | 300 UN | R$ 0,45 | **R$ 135,00** |
| 52005 | Porca M8 Zincada | 300 UN | R$ 0,18 | **R$ 54,00** |
| 52006 | Parafuso M10×30 Galvanizado | 300 UN | R$ 0,65 | **R$ 195,00** |
| 52007 | Porca M10 Galvanizada | 300 UN | R$ 0,25 | **R$ 75,00** |
| | | | **PO-2 Total** | **R$ 459,00** |

---

## Passo 6 — Recebimento de Matéria Prima — GRE (Seção 18)

7 entradas de estoque no Almoxarifado MP (depósito 1):

| Mov | Item | Tipo | Qtd | Valor Unit. | Total | NF Ref. |
|-----|------|------|-----|-------------|-------|---------|
| MV-1 | 52001 Chapa Aço | IN | 600 KG | R$ 7,80 | R$ 4.680,00 | NF-PAC-001 |
| MV-2 | 52002 Perfil Tubular | IN | 60 M | R$ 42,50 | R$ 2.550,00 | NF-PAC-001 |
| MV-3 | 52003 Perfil I | IN | 50 M | R$ 68,00 | R$ 3.400,00 | NF-PAC-001 |
| MV-4 | 52004 Parafuso M8 | IN | 300 UN | R$ 0,45 | R$ 135,00 | NF-FIX-001 |
| MV-5 | 52005 Porca M8 | IN | 300 UN | R$ 0,18 | R$ 54,00 | NF-FIX-001 |
| MV-6 | 52006 Parafuso M10 | IN | 300 UN | R$ 0,65 | R$ 195,00 | NF-FIX-001 |
| MV-7 | 52007 Porca M10 | IN | 300 UN | R$ 0,25 | R$ 75,00 | NF-FIX-001 |
| | | | | **Total GRE** | **R$ 11.089,00** | |

---

## Passo 7 — Produção PI — Semi-Acabados (Seção 19)

Roteiro de cada PI: **Seq 10 Corte/Preparação (0,25h) → Seq 20 Soldagem MIG (0,5h)**

| OF | Item | Descrição | Qtd | Início | Fim | Lote | Backflush MPs |
|----|------|-----------|-----|--------|-----|------|---------------|
| 1 | 51001 | Coluna 3m | 10 UN | 23/06 | 25/06 | LOTE-COL3-001 | 52001 ×120 KG · 52002 ×30 M |
| 2 | 51002 | Viga 2m ★ | **19 UN** | 23/06 | 25/06 | LOTE-VIG2-001 | 52001 ×152 KG · 52003 ×38 M |
| 3 | 51003 | Base Padrão | 5 UN | 23/06 | 24/06 | LOTE-BASE-001 | 52001 ×30 KG · 52004 ×40 UN |
| 4 | 51004 | Coluna 4m | 6 UN | 23/06 | 26/06 | LOTE-COL4-001 | 52001 ×96 KG · 52002 ×24 M |
| 5 | 51005 | Base Reforçada | 3 UN | 24/06 | 25/06 | LOTE-BREF-001 | 52001 ×30 KG · 52006 ×36 UN |

★ Viga 2m: demanda agregada de ambas as variantes (`5×2 + 3×3 = 19`).  
Todas as OFs PI passaram pelos estados: **OPEN → IN_PROGRESS → COMPLETED → CLOSED**.

---

## Passo 8 — Montagem Final — PA (Seção 20)

Roteiro dos PAs: **Seq 10 Corte (0,5h) → Seq 20 Soldagem (1,5h) → Seq 30 Montagem (2,0h)**

### OF-6 — PA-001 ESI-500 × 5

| Etapa | Data | Detalhe |
|-------|------|---------|
| Criação | 19/06 | `status: OPEN` |
| Início | 26/06 | `status: IN_PROGRESS` |
| Apontamento | 28/06 | Backflush PIs (Dep. Semi-Acabados) · produzidas 5 UN |
| Consumo fixadores | 28/06 | 52004 Parafuso M8 × 80 UN + 52005 Porca M8 × 80 UN |
| Conclusão | 30/06 | `status: COMPLETED` · Dep. Acabados · Lote LOTE-ESI500-001 |
| Fechamento | 30/06 | `status: CLOSED` |

### OF-7 — PA-002 ESI-750 × 3

| Etapa | Data | Detalhe |
|-------|------|---------|
| Criação | 19/06 | `status: OPEN` |
| Início | 27/06 | `status: IN_PROGRESS` |
| Apontamento | 01/07 | Backflush PIs · produzidas 3 UN |
| Consumo fixadores | 01/07 | 52006 Parafuso M10 × 60 UN + 52007 Porca M10 × 60 UN |
| Conclusão | 02/07 | `status: COMPLETED` · Dep. Acabados · Lote LOTE-ESI750-001 |
| Fechamento | 02/07 | `status: CLOSED` |

---

## Passo 9 — CRP e APS (Seção 15)

Após MRP, **antes** das OFs reais:

- `POST /api/crp/calculate` — calculou carga das máquinas no plano 2001
- `POST /api/aps/sequence` — sequenciou as operações por prioridade
- `GET /api/crp/2001` — retornou carga por centro de trabalho

Centros de trabalho utilizados:
| Máquina | Tipo | Cap./dia | Eficiência |
|---------|------|----------|------------|
| Cortadora CNC-01 (1002) | CUT | 16 UN/dia | 95% |
| Soldadora MIG-01 (1001) | WELD | 8 UN/dia | 90% |
| Mesa Montagem-01 (1003) | ASSEMBLE | 4 UN/dia | 88% |

---

## Passo 10 — Inspeção de Qualidade (Seção 22)

### Plano de Inspeção ESI-500 — ponto EXPEDIÇÃO

| Característica | Nominal | Tolerância | Crítica |
|---------------|---------|------------|---------|
| Altura total | 2.500 mm | ±5 mm | Sim |
| Carga máxima | 2.000 kg | −50 kg | Sim |
| Pintura anticorrosiva | 80 µm | ±10 µm | Não |

**Resultado**: 5/5 unidades **APROVADO**  
**Resultado ESI-750**: 3/3 unidades **APROVADO**

---

## Passo 11 — Rastreabilidade de Lotes (Seção 21)

| Item | Lote | Heat Number | Data Recebimento |
|------|------|-------------|------------------|
| 50001 ESI-500 | LOTE-ESI500-001 | H-2026-001 | 30/06/2026 |
| 50002 ESI-750 | LOTE-ESI750-001 | H-2026-002 | 02/07/2026 |

`GET /api/stock/lots/genealogy/50001/LOTE-ESI500-001` — rastreia MP → PI → PA.

---

## Passo 12 — Romaneio de Expedição (Seção 23)

| Campo | Valor |
|-------|-------|
| Destino | Construtora GrandeObra SP |
| Volumes | 4 |
| Peso total | 760 kg |
| Status | OPEN → CONFERRED → **SHIPPED** |

**Itens expedidos:**
- 50001 ESI-500 × 5 UN (Depósito Acabados)
- 50002 ESI-750 × 3 UN (Depósito Acabados)

---

## Passo 13 — NF-e de Saída (Seção 24)

**TECNOFER (PR) → Construtora GrandeObra SP**  
CFOP `6102` — Venda de produto fabricado por encomenda (interestadual PR→SP)

| Campo | Valor |
|-------|-------|
| NF número | 1001 · Série 1 |
| Data emissão | 30/06/2026 |
| Emitente | TECNOFER · CNPJ `52454668000102` · IE `9103144679` |
| Destinatário | Construtora GrandeObra SP · CNPJ `55444333000177` · IE `123456789012` |
| Valor produtos | R$ 42.900,00 |
| Frete | R$ 850,00 |
| **Valor total NF** | **R$ 43.750,00** |

**Itens NF-e:**

| Seq | NCM | Produto | Qtd | Unit. | Total | ICMS 12% |
|-----|-----|---------|-----|-------|-------|----------|
| 1 | 73089090 | ESI-500 | 5 | R$ 4.500 | R$ 22.500 | R$ 2.700 |
| 2 | 73089090 | ESI-750 | 3 | R$ 6.800 | R$ 20.400 | R$ 2.448 |

→ `POST /api/fiscal/exits/{id}/authorize` → FocusNFE homologação → **AUTORIZADA SEFAZ**

---

## Passo 14 — NF-e de Entrada (Conferência Fiscal — Seção 24b)

NF emitida pela **Siderúrgica ParanaAco** (CFOP `1101` — compra p/ industrialização):

| Item | NCM | Qtd | Unit. | Total | ICMS 12% |
|------|-----|-----|-------|-------|----------|
| 52001 Chapa Aço | 72254090 | 600 KG | R$ 7,80 | R$ 4.680 | R$ 561,60 |
| 52002 Perfil Tubular | 73063090 | 60 M | R$ 42,50 | R$ 2.550 | R$ 306,00 |
| **Total NF** | | | | **R$ 10.630,00** | R$ 1.275,60 |

Status final: **APPROVED**

---

## Passo 15 — Custo Padrão e Financeiro (Seções 25–28)

### Rollup de Custo Padrão

| Nível | Operação |
|-------|---------|
| MP | Preços de compra registrados para 7 MPs |
| Centro de trabalho | Soldadora R$ 5/h |
| Rollup ESI-500 | `POST /api/standard-cost/rollup` — 3 níveis calculados |
| Rollup ESI-750 | Variante calculada independentemente |

### ATP (Disponível para Promessa)

- `GET /api/stock/balances/atp/50001` — ESI-500: 5 UN disponíveis pós-produção
- `GET /api/stock/balances/atp/50002` — ESI-750: 3 UN disponíveis pós-produção

### Relatórios Financeiros

| Relatório | Endpoint |
|-----------|---------|
| Produtos vendidos | `GET /api/financial/relatorios/produtos-vendidos` |
| Produtos produzidos | `GET /api/financial/relatorios/produtos-produzidos` |
| Curva ABC clientes | `GET /api/financial/relatorios/curva-abc-clientes` |
| Histórico de custos | `GET /api/financial/relatorios/historico-custos` |

---

## Resumo Financeiro do Ciclo

| Categoria | Valor |
|-----------|-------|
| Receita bruta (NF-e) | R$ 42.900,00 |
| Frete cobrado | R$ 850,00 |
| **Faturamento total** | **R$ 43.750,00** |
| Custo MPs compradas | R$ 11.089,00 |
| ICMS a recolher (saída 12%) | R$ 5.148,00 |
| Crédito ICMS (entradas) | R$ 1.275,60 |
| **ICMS líquido** | **R$ 3.872,40** |

---

## Resumo do Teste

| Métrica | Valor |
|---------|-------|
| Total de checks | **153** |
| Passaram | **153 ✓** |
| Falharam | **0** |
| Avisos | **0** |
| Itens cadastrados | 14 (2 PA + 5 PI + 7 MP) |
| Estruturas BOM | 20 vínculos |
| Sugestões MRP geradas | 16 (7 FABRICACAO + 9 COMPRA) |
| Ordens planejadas firmadas | 9 PURCHASE |
| Pedidos de compra | 3 (1 MRP automático + 2 manuais) |
| GREs (entradas de estoque) | 7 |
| Ordens de produção | 7 (5 PI + 2 PA) |
| Inspeções | 2 planos · 8 UN aprovadas |
| NF-e saída (homologação) | 1 · Autorizada SEFAZ |
| NF-e entrada | 1 · Aprovada |

---

## Correções Aplicadas neste Ciclo

| # | Problema | Causa Raiz | Fix |
|---|----------|------------|-----|
| 1 | Items 52004-52007 não persistidos | `"unit_of_measurement":"PC"` inválido | Alterado para `"UN"` |
| 2 | MetalFix (PO-2) com erro FK | CNPJ `98765432000111` inválido | Trocado para `22334455000186` |
| 3 | Check passava em respostas de erro | `grep -qE 'id'` batia em "invalid"/"inválido" | Padrão refinado para `'"code":[0-9]'` |
| 4 | Items MP geravam FABRICACAO no MRP | `"engineering":{"type":0}` = FABRICADO | Alterado para `"type":1` (COMPRADO) |
| 5 | Stock acumulado entre runs | `stock_movements` ausente do RESET_SQL | Adicionadas 7 tabelas de estoque ao TRUNCATE |
| 6 | `FirmarSugestao` erro SQLSTATE 22P02 | `"FABRICACAO"` inválido em `order_type_enum` | Adicionado `mapMRPOrderType()` |
| 7 | `FirmarSugestao` erro DEPENDENTE | `"DEPENDENTE"` inválido em `demand_type_enum` | Adicionado `mapMRPDemandType()` |
