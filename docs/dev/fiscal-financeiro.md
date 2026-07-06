# Módulo Fiscal & Financeiro — VentureERP

> **Este é o documento único e completo do módulo Fiscal & Financeiro do ERP.**
> Nenhuma outra documentação detalha fiscal/financeiro — a visão geral da API
> (`visao-geral.md`) apenas aponta para cá.

> **Autenticação:** todos os endpoints exigem `Authorization: Bearer <JWT>`.
> Obtenha o token em `POST /users/login`.
> Todas as requests usam `Content-Type: application/json`.

---

## Índice

1. [Visão geral da arquitetura](#1-visão-geral-da-arquitetura)
2. [Pré-requisito: Configuração Fiscal](#2-pré-requisito-configuração-fiscal)
3. [Motor Tributário](#3-motor-tributário)
   - 3.1 [Gestão de Tabelas Tributárias](#31-gestão-de-tabelas-tributárias)
4. [Módulo Fiscal — NF-e de Saída](#4-módulo-fiscal--nf-e-de-saída) *(inclui DANFE, CC-e, cancelamento, consulta de status)*
5. [Módulo Fiscal — NF-e de Entrada](#5-módulo-fiscal--nf-e-de-entrada) *(inclui importação por chave de acesso via Focus)*
6. [CT-e (Conhecimento de Transporte)](#6-ct-e-conhecimento-de-transporte)
7. [Módulo Financeiro — Cadastros Base](#7-módulo-financeiro--cadastros-base)
8. [Contas a Pagar](#8-contas-a-pagar)
9. [Contas a Receber](#9-contas-a-receber)
10. [Fluxo de Caixa & Saldos](#10-fluxo-de-caixa--saldos)
11. [Apuração de Impostos](#11-apuração-de-impostos)
12. [Relatórios](#12-relatórios)
13. [Conciliação Bancária (OFX)](#13-conciliação-bancária-ofx)
14. [Validação de CNPJ/CPF](#14-validação-de-cnpjcpf)
15. [Limitações conhecidas](#15-limitações-conhecidas)
16. [Parâmetros Fiscais — Cadastros de Apoio](#16-parâmetros-fiscais--cadastros-de-apoio)
17. [Localização — Países e UFs](#17-localização--países-e-ufs)
18. [Classificação de Itens](#18-classificação-de-itens)
19. [Preços da Tabela de Vendas](#19-preços-da-tabela-de-vendas)
20. [Tipos de Movimento de Estoque](#20-tipos-de-movimento-de-estoque)
21. [Tipo de NF Saída — Campos Estendidos](#21-tipo-de-nota-fiscal-saída--campos-estendidos)
22. [Motivos de Transferência DAPI](#22-motivos-de-transferência-dapi)
23. [Códigos de Ajuste ICMS — Tabela 5.1.1](#23-códigos-de-ajuste-icms--tabela-511)
24. [Códigos de Ajuste ICMS — Tabelas 5.2/5.3/5.6/5.7](#24-códigos-de-ajuste-icms--tabelas-52--53--56--57)
25. [Linhas de Apuração de ICMS](#25-linhas-de-apuração-de-icms)
26. [Lançamentos Resumo de ICMS](#26-lançamentos-resumo-de-icms)
27. [Apuração do Simples Nacional](#27-apuração-do-simples-nacional)
28. [Redução / Substituição / Diferimento de ICMS/IPI](#28-cadastro-de-redução--substituição--diferimento-de-icmsipi)
29. [Aba Adicionais do Resumo de ICMS](#29-aba-adicionais-do-resumo-de-icms-c197--processos-judiciais)
30. [Restituição / Ressarcimento / Complementação de ICMS ST](#30-restituição--ressarcimento--complementação-de-icms-st)
31. [Notas Especiais de Ajuste](#31-notas-especiais-de-ajuste)
32. [SPED Contábil — ECD](#32-sped-contábil--ecd-escrituração-contábil-digital)
33. [Cadastro de Fornecedores (integração fiscal)](#33-cadastro-de-fornecedores-integração-fiscal)
34. [Cadastro de Classificações Fiscais](#34-cadastro-de-classificações-fiscais)
35. [Tipos de Operação de Entrada](#35-tipos-de-operação-de-entrada)

---

## 1. Visão geral da arquitetura

```
interfaces/http/handler/
  fiscal_handler.go      ← HTTP + JSON para toda a parte fiscal
  financial_handler.go   ← HTTP + JSON para toda a parte financeira

application/usecase/
  fiscal_uc/             ← regras de negócio fiscal (NF-e, CC-e, CT-e)
  financial_uc/          ← contas a pagar/receber, relatórios, OFX

domain/fiscal/
  engine/tax_engine.go   ← cálculo tributário (ICMS, IPI, PIS, COFINS)
  entity/                ← entidades de domínio
  repository/            ← interface do repositório

infrastructure/
  focusnfe/              ← cliente HTTP para a API Focus NF-e
  repository/fiscal/     ← implementação PostgreSQL
  repository/financial/  ← implementação PostgreSQL
```

**Stack tecnológica:**
- Go 1.25, Clean Architecture
- PostgreSQL (pgx/v5, sem ORM)
- `shopspring/decimal` para todos os valores monetários
- [Focus NF-e](https://focusnfe.com.br) para autorização, cancelamento e CC-e
- Regime tributário: **Lucro Real** (PIS/COFINS não-cumulativo)

---

## 2. Pré-requisito: Configuração Fiscal

Antes de emitir qualquer NF-e, configure a empresa. Sem isso, a autorização retorna `"token Focus NF-e não configurado"`.

### `GET /api/fiscal/config`

Retorna a configuração atual da empresa.

**Resposta esperada:**
```json
{
  "cnpj_empresa": "00000000000000",
  "razao_social": "",
  "regime_tributario": "3",
  "uf_empresa": "PR",
  "icms_interno_aliquota": 0.12,
  "icms_diferimento_percentual": 38.46,
  "focus_nfe_ambiente": "homologacao",
  "juros_mes": 0.01,
  "multa_atraso": 0.02,
  "vencimento_icms_dia": 10,
  "vencimento_ipi_dia": 15,
  "vencimento_pis_cofins_dia": 25
}
```

---

### `PUT /api/fiscal/config`

Atualiza a configuração fiscal da empresa.

**Request (dados reais da Tecnofer para homologação):**
```json
{
  "cnpj_empresa": "52454668000102",
  "razao_social": "TECNOFER FABRICACAO E MONTAGEM DE ESTRUTURAS METALICAS LTDA",
  "ie_empresa": "9103144679",
  "regime_tributario": "3",
  "uf_empresa": "PR",
  "logradouro": "Preencher conforme endereço cadastrado no SEFAZ",
  "numero": "S/N",
  "complemento": "",
  "bairro": "Zona Rural",
  "municipio": "Preencher conforme cadastro",
  "codigo_municipio": "Preencher conforme IBGE",
  "cep": "86975000",
  "telefone": "",
  "icms_interno_aliquota": 0.12,
  "icms_diferimento_percentual": 0.3846,
  "focus_nfe_token": "YvCRephh2ZiwEmpawutMj83uPJxAYMD9",
  "focus_nfe_ambiente": "homologacao",
  "juros_mes": 0.01,
  "multa_atraso": 0.02,
  "vencimento_icms_dia": 10,
  "vencimento_ipi_dia": 15,
  "vencimento_pis_cofins_dia": 25
}
```

> **Empresa cadastrada no FocusNFE:** CNPJ `52.454.668/0001-02`, IE `9103144679`,
> nome fantasia **Tecnofer**, CEP `86975-000` (PR). Token de homologação
> `YvCRephh2ZiwEmpawutMj83uPJxAYMD9` — ambiente **sempre** `"homologacao"` para testes.

| Campo | Obrigatório | Descrição |
|---|---|---|
| `focus_nfe_token` | Sim | Token da Focus NF-e (homologação ou produção) |
| `focus_nfe_ambiente` | Sim | `"homologacao"` ou `"producao"` |
| `logradouro` | Sim | Logradouro do emitente (enviado na NF-e) |
| `numero` | Sim | Número do endereço |
| `complemento` | Não | Complemento |
| `bairro` | Sim | Bairro |
| `municipio` | Sim | Nome do município |
| `codigo_municipio` | Sim | Código IBGE do município (7 dígitos) |
| `cep` | Sim | CEP sem formatação (8 dígitos) |
| `telefone` | Não | Telefone do emitente (DDD + número) |
| `icms_interno_aliquota` | Sim | Alíquota interna (decimal, ex: `0.12` = 12%) |
| `icms_diferimento_percentual` | Sim | Percentual de diferimento CST 51 como ratio (ex: `0.3846` = 38,46%). Esse valor é lido dinamicamente ao autorizar a NF-e — alterar aqui reflete imediatamente no payload Focus NF-e. |
| `regime_tributario` | Sim | `"1"` Simples, `"2"` Lucro Presumido, `"3"` Lucro Real |

> **Importante:** os campos de endereço são obrigatórios para a autorização da NF-e na SEFAZ. Sem eles o Focus NF-e rejeita o payload.

**Resposta esperada:** `200 OK` com a configuração atualizada.

---

## 3. Motor Tributário

O motor em `internal/domain/fiscal/engine/tax_engine.go` é chamado automaticamente ao criar uma NF-e de Saída. As regras implementadas são:

### Cenários de ICMS

| Cenário | Condição | Regra |
|---|---|---|
| **INTERNA_CONTRIBUINTE** | Emitente UF == Destino UF e destinatário é contribuinte | ICMS à alíquota interna configurada + **diferimento parcial CST 51** (percentual lido de `icms_diferimento_percentual` em `fiscal_configs`) |
| **INTERNA_NAO_CONTRIBUINTE** | Emitente UF == Destino UF e não-contribuinte | Base inclui IPI, CST 00, sem diferimento |
| **INTERESTADUAL contribuinte** | UFs diferentes, IE preenchida | Alíquota da tabela interestadual, base sem IPI, **sem DIFAL** |
| **INTERESTADUAL não-contribuinte / PF** | UFs diferentes, IE ausente/ISENTO ou `tipo_pessoa = "F"` | Base inclui IPI, DIFAL calculado (EC 87/2015), FCP quando aplicável |
| **Importada (Res. SF 13/2012)** | `origem_mercadoria` ∈ `{"3","4","5","8"}` | Força alíquota interestadual de **4%**, independente da UF destino |

### PIS/COFINS (Lucro Real)

- Alíquota padrão: **PIS 1,65%** e **COFINS 7,6%** sobre a base (não-cumulativo)
- NCMs com alíquotas específicas na tabela `ncm_taxes` sobrepõem os defaults
- NCMs monofásicos (CST 04) devem ter alíquota `0` na tabela → imposto zerado

### IPI

- Alíquota por NCM configurada na tabela `ncm_taxes`
- Para contribuintes: IPI **não** entra na base de ICMS
- Para não-contribuintes: IPI **entra** na base de ICMS

### DIFAL + FCP

Calculado apenas para vendas interestaduais a não-contribuintes:
```
DIFAL = (alíq. interna do estado destino − alíq. interestadual) × base ICMS
FCP   = alíq. FCP do estado destino × base ICMS
```

### ICMS-ST (Substituição Tributária)

Calculado por item **quando o item informa `mva_pct`** (a MVA pode já ser a "MVA
ajustada" para operações interestaduais — o motor aplica como recebida):
```
BaseST  = (BaseICMS próprio + IPI) × (1 + MVA) × (1 − red_base_st_pct)
ICMS-ST = BaseST × alíq. interna do destino − ICMS próprio   (mínimo 0)
```
- A **alíquota interna do destino** (a empresa é o substituto) é resolvida da tabela
  de ICMS interno para a UF de destino (interestadual) ou da configuração fiscal
  (operação interna). Pode ser sobreposta por `aliq_interna_destino_st`.
- O **CST de ICMS** é promovido para a variante de ST: `00 → 10`, `20 → 70`,
  `51 → 10` (a ST prevalece sobre o diferimento, que é zerado).
- Os valores (`base_icms_st`, `aliq_icms_st`, `valor_icms_st`, `mva`) são
  persistidos por item, somados no cabeçalho da nota (`base_icms_st`,
  `valor_icms_st`) e enviados no payload Focus NF-e. O **ICMS-ST soma ao total
  da NF-e** (cobrado do destinatário), diferentemente do ICMS próprio.

---

## 3.1 Gestão de Tabelas Tributárias

As tabelas de alíquotas são consultadas pelo motor tributário em tempo de cálculo. Gerencie-as via API para refletir mudanças na legislação.

### Tabela NCM — `POST /api/fiscal/tabelas/ncm`

Cria ou atualiza (upsert) as alíquotas de IPI, PIS e COFINS para um NCM específico.

**Request:**
```json
{
  "ncm": "84714900",
  "aliq_ipi": 0.05,
  "aliq_pis": 0.0165,
  "aliq_cofins": 0.076,
  "cst_pis": "01",
  "cst_cofins": "01",
  "cst_ipi": "50",
  "description": "Computadores portáteis"
}
```

> Para NCMs monofásicos (alíquota zero), envie `"aliq_pis": 0, "aliq_cofins": 0, "cst_pis": "04", "cst_cofins": "04"`.

**Resposta esperada (`200 OK`):** objeto `NcmTaxTable` com os dados salvos.

---

### Tabela NCM — `GET /api/fiscal/tabelas/ncm`

Lista todos os NCMs ativos cadastrados.

**Resposta esperada:**
```json
[
  {
    "ncm": "84714900",
    "aliq_ipi": 0.05,
    "aliq_pis": 0.0165,
    "aliq_cofins": 0.076,
    "cst_pis": "01",
    "cst_cofins": "01",
    "cst_ipi": "50",
    "description": "Computadores portáteis",
    "is_active": true
  }
]
```

---

### Tabela NCM — `DELETE /api/fiscal/tabelas/ncm/{ncm}`

Desativa (soft delete) um NCM da tabela. Não remove fisicamente o registro.

**Exemplo:** `DELETE /api/fiscal/tabelas/ncm/84714900`

**Resposta esperada:** `204 No Content`

---

### ICMS Interestadual — `POST /api/fiscal/tabelas/icms-interestadual`

Cria ou atualiza a alíquota de ICMS para uma combinação origem/destino.

**Request:**
```json
{
  "origin_uf": "PR",
  "destination_uf": "SP",
  "aliq_icms": 0.12
}
```

**Resposta esperada:** `200 OK`

---

### ICMS Interestadual — `GET /api/fiscal/tabelas/icms-interestadual`

Retorna todas as alíquotas interestaduais cadastradas no formato `"ORIGEM_DESTINO": aliquota`.

**Resposta esperada:**
```json
{
  "PR_SP": 0.12,
  "PR_RJ": 0.12,
  "PR_AM": 0.07
}
```

---

### ICMS Interno — `POST /api/fiscal/tabelas/icms-interno`

Cria ou atualiza a alíquota interna de ICMS e FCP de um estado.

**Request:**
```json
{
  "uf": "SP",
  "aliq_icms": 0.18,
  "aliq_fcp": 0.02
}
```

**Resposta esperada:** `200 OK`

---

### ICMS Interno — `GET /api/fiscal/tabelas/icms-interno`

Retorna as alíquotas internas de ICMS e FCP por estado.

**Resposta esperada:**
```json
{
  "SP": { "ICMS": 0.18, "FCP": 0.02 },
  "RJ": { "ICMS": 0.20, "FCP": 0.02 },
  "PR": { "ICMS": 0.12, "FCP": 0.00 }
}
```

> **Uso pelo motor tributário:** ao calcular DIFAL para vendas interestaduais a não-contribuintes, o motor busca automaticamente a alíquota interna do estado destino nesta tabela. Se não encontrar, usa o valor configurado em `icms_interno_aliquota` da configuração fiscal.

---

## 4. Módulo Fiscal — NF-e de Saída

### Fluxo completo

```
1. POST /api/fiscal/exits/create      → cria NF-e (status: rascunho, calcula impostos)
2. POST /api/fiscal/exits/{code}/authorize → envia para Focus NF-e → status: autorizada
3. (opcional) POST /exits/{code}/carta-correcao → CC-e para correção de dados
4. (opcional) POST /exits/{code}/cancel → cancela NF-e autorizada
```

---

### `POST /api/fiscal/exits/create`

Cria a NF-e em rascunho e **calcula os impostos automaticamente** (ICMS, IPI, PIS, COFINS, DIFAL).

**Request:**
```json
{
  "numero_nf": 1001,
  "serie": "001",
  "data_emissao": "2024-05-15",
  "data_saida": "2024-05-15",
  "cnpj_destinatario": "98765432000188",
  "razao_social_destinatario": "Cliente Exemplo SA",
  "ie_destinatario": "1234567890",
  "uf_destinatario": "SP",
  "tipo_pessoa": "J",
  "cfop": "6101",
  "natureza_operacao": "Venda de mercadoria",
  "valor_produtos": 10000.00,
  "valor_frete": 150.00,
  "valor_seguro": 0.00,
  "valor_desconto": 0.00,
  "sales_order_code": 42,
  "itens": [
    {
      "sequence": 1,
      "item_code": 10,
      "ncm": "84714900",
      "cfop": "6101",
      "quantidade": 10,
      "unit_price": 1000.00,
      "total_price": 10000.00,
      "origem_mercadoria": "0",
      "description": "Computador Portátil"
    }
  ]
}
```

| Campo | Obrigatório | Descrição |
|---|---|---|
| `numero_nf` | Sim | Número da nota |
| `serie` | Sim | Série da nota (ex: `"001"`) |
| `cfop` | Sim | CFOP principal (ex: `"6101"` = venda estadual, `"5101"` = venda interestadual) |
| `ie_destinatario` | Não | Ausente ou `"ISENTO"` → destinatário não-contribuinte → DIFAL |
| `tipo_pessoa` | Não | `"F"` = Pessoa Física → tratada como não-contribuinte para DIFAL |
| `origem_mercadoria` | Sim (por item) | `"0"` nacional, `"3"/"4"/"5"/"8"` importada → 4% interestadual |
| `mva_pct` | Não (por item) | MVA como ratio (ex: `0.40` = 40%). Quando `> 0`, dispara o cálculo de ICMS-ST do item |
| `aliq_interna_destino_st` | Não (por item) | Sobrepõe a alíquota interna do destino usada na ST (ratio) |
| `red_base_st_pct` | Não (por item) | Redução da base de ST (ratio, ex: `0.20` = 20%) |

**Resposta esperada (`201 Created`):**
```json
{
  "id": 1,
  "numero_nf": 1001,
  "status": "rascunho",
  "valor_produtos": 10000.00,
  "valor_ipi": 500.00,
  "valor_icms": 1200.00,
  "valor_pis": 165.00,
  "valor_cofins": 760.00,
  "valor_total": 10650.00,
  "itens": [
    {
      "sequence": 1,
      "base_icms": 10000.00,
      "aliq_icms": 0.12,
      "valor_icms": 1200.00,
      "valor_icms_diferido": 0.00,
      "base_ipi": 10000.00,
      "aliq_ipi": 0.05,
      "valor_ipi": 500.00,
      "aliq_pis": 0.0165,
      "valor_pis": 165.00,
      "aliq_cofins": 0.076,
      "valor_cofins": 760.00,
      "cst_icms": "00",
      "cst_ipi": "50",
      "cst_pis": "01",
      "cst_cofins": "01"
    }
  ]
}
```

> **Nota:** `valor_total = valor_produtos + valor_ipi + valor_frete + valor_seguro - valor_desconto` (ICMS não soma no total da NF-e).
>
> **`aliq_pis` / `aliq_cofins`** são lidos da tabela `ncm_taxes` para o NCM do item. Se o NCM não estiver cadastrado, usa o default de Lucro Real (1,65% e 7,6%). Itens com CST monofásico (04) devem ter alíquota `0` na tabela para que o payload Focus NF-e seja enviado corretamente com valor zerado. Esses campos são persistidos na tabela `fiscal_exit_items` (colunas adicionadas pela migration `000101`).

---

### `POST /api/fiscal/exits/from-load` — emissão por carga

Cria uma NF-e de saída em rascunho a partir de uma **carga de expedição**
(`shipment_loads`). O endpoint consome os romaneios vinculados à carga, monta
os itens da nota, busca preço no pedido de venda quando o romaneio estiver
ligado a pedido e vincula a nota gerada de volta à carga/romaneios.

**Request:**
```json
{
  "load_code": 9001,
  "serie": "001",
  "data_emissao": "2026-07-06",
  "data_saida": "2026-07-06",
  "cnpj_destinatario": "98765432000188",
  "razao_social_destinatario": "Cliente Exemplo SA",
  "ie_destinatario": "1234567890",
  "uf_destinatario": "SP",
  "tipo_pessoa": "J",
  "cfop": "5102",
  "natureza_operacao": "Venda de mercadoria adquirida de terceiros",
  "valor_frete": 180.00,
  "valor_seguro": 0.00,
  "valor_desconto": 0.00,
  "origem_mercadoria": "0",
  "item_overrides": [
    {
      "shipment_code": 1042,
      "item_code": 1001,
      "unit_price": 125.50,
      "ncm": "73089010",
      "description": "Perfil dobrado galvanizado"
    }
  ]
}
```

| Campo | Obrigatório | Descrição |
|---|---|---|
| `load_code` | Sim | Código da carga criada em `/api/shipments/loads` |
| `serie`, `data_emissao`, `cfop`, `natureza_operacao` | Sim | Dados fiscais principais da NF-e |
| `item_overrides[].unit_price` | Condicional | Obrigatório quando o romaneio não tem pedido de venda com preço de item |
| `item_overrides[].shipment_code` | Não | Restringe o override a um romaneio específico; sem ele, vale para o item em qualquer romaneio da carga |

**Validações principais:**
- Cargas `CANCELLED` ou `SHIPPED` não podem ser faturadas.
- A carga precisa ter ao menos um romaneio e não pode já ter nota fiscal vinculada.
- Romaneios cancelados ou despachados não entram no faturamento.
- Se o item estiver conferido, a NF usa `conferred_qty`; caso contrário usa a quantidade planejada.
- Se não houver preço no pedido nem override, o faturamento é bloqueado para evitar NF-e zerada.

**Efeitos gravados:**
- `fiscal_exits.source_type = "LOAD"`.
- `fiscal_exits.shipment_load_code = load_code`.
- `shipment_load_fiscal_notes` recebe o vínculo `load_code × fiscal_exit_id`.
- Cada romaneio da carga recebe `fiscal_exit_id` e `nfe_number`.

Depois de criada, a autorização continua pelo fluxo normal:
`POST /api/fiscal/exits/{code}/authorize`.

---

### DANFE a partir de cupom fiscal, NFC-e ou CF-e

O cadastro manual de NF-e (`/api/fiscal/exits/create`) agora aceita metadados de
origem para cobrir emissão a partir de cupom fiscal/NFC-e/CF-e, mantendo a
rastreabilidade do documento de venda original:

```json
{
  "numero_nf": 1200,
  "serie": "001",
  "data_emissao": "2026-07-06",
  "cfop": "5929",
  "natureza_operacao": "Operacao tambem registrada em cupom fiscal",
  "valor_produtos": 250.00,
  "source_type": "COUPON",
  "fiscal_coupon_number": "CF-123456",
  "fiscal_coupon_date": "2026-07-06",
  "fiscal_coupon_ecf_serial": "ECF000123456",
  "itens": []
}
```

Para NFC-e/CF-e de consumidor, use `source_type` como `NFCE` ou `CFE`. O
cálculo/autorização segue o fluxo fiscal já existente; os campos de origem
servem para consulta, auditoria, SPED/livros e impressão/observações.

---

### `POST /api/fiscal/exits/{code}/authorize`

Envia a NF-e para a SEFAZ via API Focus NF-e. Só funciona se o status for `"rascunho"`.

**Request:** body vazio `{}`

**O que acontece internamente:**
1. Busca os dados da NF-e e seus itens
2. Lê o token Focus NF-e da configuração fiscal
3. Monta o payload conforme layout Focus NF-e v2
4. Chama `POST https://homologacao.focusnfe.com.br/v2/nfe/{ref}?substituicao=true`
5. Salva a chave de acesso, protocolo e ref retornados pelo Focus
6. Cria automaticamente uma **Conta a Receber** vinculada à NF-e (vencimento em 30 dias)
7. **Baixa o estoque**: posta um movimento **`OUT`** por item (depósito resolvido a
   partir do item do **pedido de venda** vinculado), reduzindo o saldo de acabados
8. **Consome as reservas** ativas do pedido de venda (`SALES_ORDER`)
9. Marca o **pedido de venda como Faturado** (`status = "F"`)
10. Registra o log da requisição/resposta no banco

> Os passos 7–9 são *best-effort* e **não desfazem** uma NF-e já autorizada na SEFAZ;
> exigem `sales_order_code` na NF-e e `warehouse_code` no item do pedido. A
> **expedição/romaneio** (logística) é tratada à parte (ver `manufatura-e-compras.md` §19).

**Resposta esperada (`200 OK`):**
```json
{
  "id": 1,
  "status": "autorizada",
  "chave_nfe": "35240512345678000195550010000010011000000001",
  "protocolo": "135240000000001",
  "focus_ref": "10012345678"
}
```

**Erros comuns:**
- `"NF-e deve estar em rascunho para autorizar"` → status já é `autorizada` ou `cancelada`
- `"token Focus NF-e não configurado"` → execute `PUT /api/fiscal/config` primeiro
- `"Focus NF-e: ..."` → rejeição da SEFAZ, o erro da SEFAZ é encaminhado

---

### `POST /api/fiscal/exits/{code}/cancel`

Cancela uma NF-e autorizada. Só pode ser cancelada dentro do prazo legal (24h em homologação, até 30 dias em produção dependendo do estado).

**Request:**
```json
{
  "justificativa": "Erro na emissão da nota fiscal, produto não expedido."
}
```

> A justificativa deve ter no mínimo 15 caracteres.

**Resposta esperada (`200 OK`):**
```json
{
  "id": 1,
  "status": "cancelada",
  "mensagem": "Cancelamento autorizado"
}
```

---

### `POST /api/fiscal/exits/{code}/carta-correcao`

Emite uma CC-e (Carta de Correção Eletrônica). Só pode ser emitida para NF-e com status `"autorizada"`.

**Quando usar:** corrigir dados do destinatário, natureza da operação, CFOP, descrição dos itens. **Não corrige:** valor dos impostos, quantidade, data de emissão.

**Request:**
```json
{
  "texto_correcao": "Corrigir razão social do destinatário para: Cliente Exemplo Comercial SA"
}
```

> O texto deve ter no mínimo 15 caracteres.

**Resposta esperada (`200 OK`):**
```json
{
  "fiscal_exit_id": 1,
  "texto_correcao": "Corrigir razão social do destinatário para: Cliente Exemplo Comercial SA",
  "status": "autorizado",
  "numero_seq": 1
}
```

---

### `GET /api/fiscal/exits/list`

Lista todas as NF-e de saída.

**Resposta esperada:**
```json
[
  {
    "id": 1,
    "numero_nf": 1001,
    "serie": "001",
    "status": "autorizada",
    "cnpj_destinatario": "98765432000188",
    "razao_social_destinatario": "Cliente Exemplo SA",
    "valor_total": 10650.00,
    "data_emissao": "2024-05-15T00:00:00Z"
  }
]
```

---

### `GET /api/fiscal/exits/{code}`

Retorna os dados completos de uma NF-e de saída incluindo todos os itens.

---

### `GET /api/fiscal/exits/{id}/status`

Consulta o status atualizado da NF-e diretamente na API Focus NF-e e sincroniza o status local.

**Quando usar:** após enviar para autorização, para verificar se já foi processada (`"autorizado"`, `"rejeitado"`, `"cancelado"`, `"processando"`).

**Resposta esperada (`200 OK`):**
```json
{
  "exit_id": 1,
  "focus_ref": "10012345678",
  "status": "autorizado",
  "chave_nfe": "35240512345678000195550010000010011000000001",
  "protocolo": "135240000000001"
}
```

| Campo | Descrição |
|---|---|
| `status` | Status retornado pelo Focus: `autorizado`, `cancelado`, `rejeitado`, `erro_autorizacao`, `processando` |
| `chave_nfe` | Chave de acesso da NF-e (44 dígitos) — presente quando autorizada |
| `protocolo` | Protocolo de autorização da SEFAZ |
| `motivo` | Motivo de rejeição, quando disponível |

> O status local é atualizado automaticamente se o Focus retornar um estado terminal.

**Erros comuns:**
- `"NF-e X não possui referência Focus"` → NF-e ainda não foi enviada para autorização

---

### `GET /api/fiscal/exits/{id}/cartas-correcao`

Lista todas as CC-e emitidas para uma NF-e.

**Resposta esperada (`200 OK`):**
```json
[
  {
    "id": 1,
    "fiscal_exit_id": 1,
    "texto_correcao": "Corrigir razão social do destinatário para: Cliente Exemplo Comercial SA",
    "status": "autorizado",
    "numero_seq": 1,
    "created_at": "2024-05-16T10:30:00Z"
  }
]
```

---

### `GET /api/fiscal/exits/{id}/danfe`

Retorna as URLs do **DANFE** (PDF) e do **XML** da NF-e autorizada. O sistema
persiste os paths retornados pelo Focus na autorização; se não estiverem
armazenados (NF-es autorizadas antes dessa implementação), o endpoint consulta
a API Focus automaticamente e os persiste.

**Quando usar:** para exibir ou baixar o DANFE no front-end sem expor o token
Focus NF-e ao cliente — o back-end entrega a URL absoluta, e o download é feito
diretamente do CDN do Focus.

**Resposta esperada (`200 OK`):**
```json
{
  "exit_id": 1,
  "danfe_url": "https://homologacao.focusnfe.com.br/notas_fiscais/NFe35240512345678000195550010000010011000000001-nfe.pdf",
  "xml_url": "https://homologacao.focusnfe.com.br/notas_fiscais/NFe35240512345678000195550010000010011000000001-nfe.xml",
  "status": "autorizada"
}
```

| Campo | Descrição |
|---|---|
| `danfe_url` | URL absoluta do PDF do DANFE no servidor Focus NF-e |
| `xml_url` | URL absoluta do XML da NF-e no servidor Focus NF-e |
| `status` | Status atual da NF-e no banco local |

**Erros comuns:**
- `"NF-e deve estar autorizada para consultar DANFE"` → NF-e em rascunho ou cancelada
- `"token Focus NF-e não configurado"` → execute `PUT /api/fiscal/config` primeiro

---

## 5. Módulo Fiscal — NF-e de Entrada

NF-e de entrada representa compras/recebimento de mercadorias. Os impostos são informados pelo emitente (não calculados pelo sistema).

> **Vínculo com o fornecedor.** Na importação de NF-e de compra por chave de acesso, o
> sistema casa o **CNPJ/CPF do emitente** a um fornecedor cadastrado e grava o vínculo
> em `fiscal_entries.supplier_code` (campo `supplier_matched` no retorno indica se
> houve correspondência). Esse vínculo habilita a geração de Conta a Pagar por
> fornecedor e o uso de `icms_contributor` e do Tipo de NF default do fornecedor.
> Ver seção 33 e [`cadastros-fornecedor.md`](cadastros-fornecedor.md).

### `POST /api/fiscal/entries/create`

Lançamento manual de NF-e de entrada.

**Request:**
```json
{
  "numero_nf": 5500,
  "serie": "001",
  "modelo": "55",
  "data_emissao": "2024-05-10",
  "data_entrada": "2024-05-12",
  "cnpj_emitente": "11222333000181",
  "razao_social_emitente": "Fornecedor XYZ LTDA",
  "ie_emitente": "9876543210",
  "uf_emitente": "SP",
  "valor_produtos": 5000.00,
  "valor_frete": 200.00,
  "valor_seguro": 0.00,
  "valor_desconto": 0.00,
  "valor_ipi": 250.00,
  "valor_icms": 600.00,
  "valor_pis": 82.50,
  "valor_cofins": 380.00,
  "valor_total": 5450.00,
  "tipo_documento": "NF-e",
  "purchase_order_code": 15,
  "itens": [
    {
      "sequence": 1,
      "item_code": 20,
      "ncm": "84714900",
      "cfop": "1101",
      "quantity": 5,
      "unit_price": 1000.00,
      "total_price": 5000.00,
      "base_icms": 5000.00,
      "aliq_icms": 0.12,
      "valor_icms": 600.00,
      "base_ipi": 5000.00,
      "aliq_ipi": 0.05,
      "valor_ipi": 250.00,
      "valor_pis": 82.50,
      "valor_cofins": 380.00,
      "cst_icms": "00",
      "cst_ipi": "50",
      "cst_pis": "01",
      "cst_cofins": "01",
      "gera_credito_icms": true,
      "gera_credito_ipi": true,
      "gera_credito_pis": true,
      "gera_credito_cofins": true
    }
  ]
}
```

**Resposta esperada (`201 Created`):** objeto completo da entrada com `"status": "pendente"`.

---

### `POST /api/fiscal/entries/upload-nfe`

Importa NF-e a partir do XML da SEFAZ.

**Request:**
```json
{
  "xml_content": "<?xml version=\"1.0\" encoding=\"UTF-8\"?><nfeProc>...</nfeProc>"
}
```

O sistema extrai automaticamente todos os campos do XML e cria a entrada.

---

### `POST /api/fiscal/entries/import-nfe`

Importa uma NF-e de entrada diretamente pela **chave de acesso**, consultando a Focus NF-e. Não é necessário ter o XML em mãos — o sistema baixa, processa e já movimenta o estoque automaticamente.

**Pré-requisito:** token Focus NF-e configurado em `PUT /api/fiscal/config` e ambiente correto.

**Request:**
```json
{
  "access_key": "35260512345678000100550010000012341123456789"
}
```

**O que acontece internamente:**
1. Consulta a Focus NF-e com a chave de acesso
2. Baixa o XML e extrai todos os dados da NF-e (emitente, itens, impostos)
3. Casa o **CNPJ do emitente** com um fornecedor cadastrado (quando habilitado)
4. Cria a nota de entrada com status `"aprovada"` (entrada direta, sem etapa de aprovação manual)
5. Movimenta o estoque de cada item da nota com tipo **`IN`** — o movimento
   **atualiza o saldo** (`stock_balances`: quantidade + custo médio ponderado) na
   mesma transação
6. Quando informado `purchase_order_code`, **baixa o pedido de compra**: soma as
   quantidades recebidas em cada item (`received_qty`) e recalcula o status da linha
   e do cabeçalho (`PARTIAL`/`RECEIVED`) — via `PurchaseOrderRepository.RegisterReceipts`.
   O resultado traz `purchase_order_lines_written_down`.

**Resposta esperada (`201 Created`):**
```json
{
  "id": 42,
  "numero_nf": 12341,
  "status": "aprovada",
  "cnpj_emitente": "12345678000100",
  "razao_social_emitente": "Fornecedor XYZ LTDA",
  "valor_total": 5450.00
}
```

**Erros comuns:**
- `"token Focus NF-e não configurado"` → execute `PUT /api/fiscal/config` primeiro
- `"chave de acesso inválida"` → a chave deve ter exatamente 44 dígitos
- `"NF-e não encontrada"` → chave não existe na base Focus para o ambiente configurado

> **Diferença em relação ao `upload-nfe`:** o `upload-nfe` recebe o XML bruto; o `import-nfe` recebe só a chave de 44 dígitos e busca o XML automaticamente via API Focus. Ambos criam a entrada, mas o `import-nfe` já aprova e movimenta o estoque na mesma operação.

---

### `POST /api/fiscal/entries/{id}/approve`

Aprova uma NF-e de entrada (confirma o recebimento). Muda o status de `"pendente"` para `"aprovada"`. Cria automaticamente uma **Conta a Pagar** vinculada.

**Request:** `{}`

**Resposta esperada (`200 OK`):**
```json
{
  "id": 1,
  "status": "aprovada"
}
```

---

### `GET /api/fiscal/entries/list`

Lista todas as NF-e de entrada.

### `GET /api/fiscal/entries/{id}`

Retorna uma NF-e de entrada por ID com todos os itens.

---

## 6. CT-e (Conhecimento de Transporte)

O CT-e é registrado localmente para fins de custeio do frete e vinculação com NF-e de entrada. **A autorização na SEFAZ via Focus está disponível** (ver `POST /api/fiscal/cte/{code}/authorize` abaixo) — o registro local continua sendo o ponto de partida.

### `POST /api/fiscal/cte/create`

**Request:**
```json
{
  "numero_cte": 800,
  "serie": "001",
  "data_emissao": "2024-05-10",
  "data_entrada": "2024-05-12",
  "cnpj_emitente": "33000167000101",
  "razao_social_emitente": "Transportadora Nacional SA",
  "uf_emitente": "SP",
  "cfop": "1352",
  "valor_frete": 450.00,
  "valor_seguro": 50.00,
  "valor_outros": 0.00,
  "valor_total": 500.00,
  "valor_icms": 60.00,
  "base_icms": 500.00,
  "aliq_icms": 0.12,
  "cst_icms": "00",
  "tipo_rateio": "VALOR",
  "fiscal_entry_id": 1
}
```

| Campo | Descrição |
|---|---|
| `tipo_rateio` | `"VALOR"` (padrão) ou `"PESO"` — como o frete é rateado entre NF-es |
| `fiscal_entry_id` | Vincula o CT-e à NF-e de entrada correspondente |

**Resposta esperada (`201 Created`):** objeto do CT-e com ID gerado.

---

### `GET /api/fiscal/cte/list`

Lista todos os CT-es.

### `GET /api/fiscal/cte/{id}`

Retorna um CT-e por ID.

---

### `POST /api/fiscal/cte/{code}/authorize`

Transmite o CT-e para a SEFAZ via Focus (escopo `fiscal:authorize`). Exige que o
CT-e tenha sido criado com o campo **`emission_data`** — um JSON com o detalhe de
emissão exigido pelo documento. O **emitente** é preenchido automaticamente a partir
da configuração fiscal; valores e ICMS, quando ausentes no `emission_data`, são
herdados do CT-e registrado.

**`emission_data` (exemplo):**
```json
{
  "natureza_operacao": "Prestação de serviço de transporte",
  "tipo_cte": 0,
  "tipo_servico": 0,
  "modal": "01",
  "uf_inicio": "PR", "municipio_inicio": "Curitiba",
  "uf_fim": "SP", "municipio_fim": "São Paulo",
  "tomador_servico": 3,
  "remetente":   { "cnpj": "52454668000102", "nome": "TECNOFER FABRICACAO E MONTAGEM DE ESTRUTURAS METALICAS LTDA", "uf": "PR", "codigo_municipio": "4106902" },
  "destinatario":{ "cnpj": "98765432000188", "nome": "Destinatário SA", "uf": "SP", "codigo_municipio": "3550308" },
  "produto_predominante": "Peças metálicas",
  "valor_carga": 50000.00,
  "valor_total_prestacao": 500.00,
  "icms": { "situacao_tributaria": "00", "base_calculo": 500.00, "aliquota": 12.0, "valor": 60.00 }
}
```

**O que acontece:** monta o payload Focus CT-e, chama `POST /v2/cte?ref=...`, faz
*poll* até estado terminal e, autorizado, grava `chave_acesso`, `protocolo` e
`focus_ref`, mudando o status para `AUTORIZADO`.

**Resposta esperada (`200 OK`):** o CT-e atualizado com `status: "AUTORIZADO"`,
`chave_acesso`, `protocolo` e `focus_ref`.

**Erros comuns:**
- `"CT-e X não possui emission_data..."` → crie o CT-e com o campo `emission_data`
- `"token Focus NF-e não configurado"` → configure em `PUT /api/fiscal/config`
- `"Focus CT-e: ..."` → rejeição da SEFAZ (mensagem encaminhada)

---

## 7. Módulo Financeiro — Cadastros Base

### 7.1 Contas Bancárias

#### `POST /api/financial/contas-bancarias/create`

```json
{
  "banco": "341",
  "agencia": "1234",
  "conta": "56789",
  "digito": "0",
  "descricao": "Conta Principal Itaú",
  "titular": "Empresa Teste LTDA",
  "saldo_inicial": 10000.00,
  "chave_pix": "empresa@email.com",
  "tipo_chave_pix": "email"
}
```

**Resposta:** `201 Created` com o objeto da conta incluindo `id`.

#### `GET /api/financial/contas-bancarias/list`

Retorna lista de todas as contas bancárias com saldo atual.

---

### 7.2 Condições de Pagamento

#### `POST /api/financial/condicoes-pagamento/create`

```json
{
  "nome": "30/60/90",
  "parcelas": "30,60,90"
}
```

O campo `parcelas` é uma string de dias separados por vírgula. `"0"` = à vista.

---

### 7.3 Plano de Contas

#### `POST /api/financial/plano-contas/create`

```json
{
  "codigo": "3.1.01",
  "descricao": "Receita com Vendas de Produtos",
  "tipo": "RECEITA",
  "natureza": "CREDITO",
  "parent_code": "3.1",
  "nivel": 3
}
```

| `tipo` | Valores | Descrição |
|---|---|---|
| `tipo` | `RECEITA`, `DESPESA`, `ATIVO`, `PASSIVO`, `PATRIMONIO` | Classificação contábil |
| `natureza` | `DEBITO`, `CREDITO` | Natureza do lançamento |

---

### 7.4 Centros de Custo

#### `POST /api/financial/centros-custo/create`

```json
{
  "codigo": "CC-001",
  "descricao": "Produção",
  "tipo": "PRODUTIVO"
}
```

---

## 8. Contas a Pagar

### `POST /api/financial/contas-pagar/create`

```json
{
  "numero_documento": "NF-5500",
  "tipo_documento": "NF-e",
  "fornecedor_id": 3,
  "fiscal_entry_id": 1,
  "data_emissao": "2024-05-10",
  "data_vencimento": "2024-06-10",
  "valor_bruto": 5450.00,
  "desconto": 0.00,
  "parcela_numero": 1,
  "parcela_total": 1,
  "forma_pagamento": "transferencia",
  "plano_contas_id": 5,
  "centro_custo_id": 2,
  "observacao": "Compra de insumos"
}
```

**Resposta:** `201 Created` com `"status": "pendente"`.

---

### `POST /api/financial/contas-pagar/{id}/approve`

Aprova ou rejeita a conta a pagar (workflow de aprovação).

```json
{
  "motivo_rejeicao": null
}
```

Para rejeitar: `{ "motivo_rejeicao": "Nota fiscal com divergência de valores" }`.

---

### `POST /api/financial/contas-pagar/{id}/baixar`

Registra o pagamento (baixa). Operação **atômica**: atualiza a conta, lança no fluxo de caixa e atualiza o saldo da conta bancária.

```json
{
  "conta_bancaria_id": 1,
  "valor_pago": 5450.00,
  "data_pagamento": "2024-06-10",
  "observacao": "Pago via TED"
}
```

**Pagamento parcial:** informe `valor_pago` menor que o saldo. O sistema mantém o registro com o restante a pagar.

**Resposta esperada (`200 OK`):** objeto atualizado com `"status": "pago"` (ou `"parcial"` se pagamento parcial).

---

### `POST /api/financial/contas-pagar/{id}/cancel`

Cancela uma conta a pagar que ainda não foi paga.

**Request:** `{}`

---

### `GET /api/financial/contas-pagar/list`

Lista contas a pagar com filtros opcionais via query string:
- `?status=pendente` — `pendente`, `aprovado`, `pago`, `cancelado`
- `?start_date=2024-01-01&end_date=2024-12-31`
- `?fornecedor_id=3`

---

### `GET /api/financial/contas-pagar/{id}`

Retorna uma conta a pagar por ID.

---

### `GET /api/financial/contas-pagar/aging`

Relatório de aging (vencimento) para contas a pagar. Agrupa por faixa de atraso.

**Resposta esperada:**
```json
{
  "a_vencer": 15000.00,
  "vencido_ate_30": 3000.00,
  "vencido_31_60": 1500.00,
  "vencido_61_90": 500.00,
  "vencido_acima_90": 200.00,
  "total": 20200.00
}
```

---

## 9. Contas a Receber

Criada automaticamente ao autorizar uma NF-e de saída, ou manualmente.

### `POST /api/financial/contas-receber/create`

```json
{
  "numero_documento": "NF-1001",
  "cliente_id": 7,
  "fiscal_exit_id": 1,
  "data_emissao": "2024-05-15",
  "data_vencimento": "2024-06-15",
  "valor_bruto": 10650.00,
  "desconto": 0.00,
  "parcela_numero": 1,
  "parcela_total": 1,
  "forma_pagamento": "boleto",
  "observacao": "Venda de equipamentos"
}
```

---

### `POST /api/financial/contas-receber/{id}/baixar`

Baixa uma conta a receber. Operação **atômica**: atualiza o saldo da conta bancária, lança no fluxo de caixa e baixa o registro.

```json
{
  "conta_bancaria_id": 1,
  "valor_recebido": 10650.00,
  "data_recebimento": "2024-06-15",
  "observacao": "Recebido via PIX"
}
```

**Recebimento parcial:** informe `valor_recebido` menor que o saldo. Um novo registro com o restante é criado automaticamente.

---

### `POST /api/financial/contas-receber/{id}/cancel`

Cancela uma conta a receber não baixada.

---

### `GET /api/financial/contas-receber/aging`

Aging de contas a receber por faixa de vencimento.

---

## 10. Fluxo de Caixa & Saldos

### `GET /api/financial/fluxo-caixa?start_date=2024-01-01&end_date=2024-12-31`

Extrato real do fluxo de caixa (entradas e saídas efetivadas) por período.

**Resposta esperada:**
```json
[
  {
    "data": "2024-05-15",
    "tipo": "ENTRADA",
    "valor": 10650.00,
    "descricao": "Recebimento NF-1001",
    "conta_bancaria_id": 1,
    "conciliado": false
  }
]
```

---

### `GET /api/financial/fluxo-projetado?start_date=2024-05-01`

Projeção de caixa futura baseada em contas a pagar e a receber ainda em aberto.

**Resposta esperada:**
```json
[
  {
    "data_vencimento": "2024-06-10",
    "tipo": "SAIDA",
    "valor": 5450.00,
    "descricao": "CP: NF-5500"
  },
  {
    "data_vencimento": "2024-06-15",
    "tipo": "ENTRADA",
    "valor": 10650.00,
    "descricao": "CR: NF-1001"
  }
]
```

---

### `GET /api/financial/saldo-contas`

Saldo atual de todas as contas bancárias.

**Resposta esperada:**
```json
[
  {
    "id": 1,
    "banco": "341",
    "descricao": "Conta Principal Itaú",
    "saldo_atual": 15200.00
  }
]
```

---

## 11. Apuração de Impostos

### `POST /api/financial/apuracao-impostos`

Apura os impostos de um período (competência). Consolida ICMS, IPI, PIS e COFINS das NF-es autorizadas do período.

**Request:**
```json
{
  "competencia": "2024-05"
}
```

O formato da competência é `YYYY-MM`.

**Resposta esperada (`201 Created`):**
```json
{
  "competencia": "2024-05",
  "valor_icms_saidas": 12000.00,
  "valor_icms_entradas": 6000.00,
  "saldo_icms": 6000.00,
  "valor_ipi_saidas": 500.00,
  "valor_ipi_entradas": 250.00,
  "saldo_ipi": 250.00,
  "valor_pis_saidas": 1650.00,
  "valor_pis_entradas": 825.00,
  "saldo_pis": 825.00,
  "valor_cofins_saidas": 7600.00,
  "valor_cofins_entradas": 3800.00,
  "saldo_cofins": 3800.00,
  "status": "apurado"
}
```

---

### `GET /api/financial/apuracao-impostos/{competencia}`

Retorna uma apuração já realizada. Ex: `GET /api/financial/apuracao-impostos/2024-05`.

---

## 12. Relatórios

Todos os relatórios exigem query params `?start=YYYY-MM-DD&end=YYYY-MM-DD` (exceto aging e ficha técnica).

### R01 — Livro de Entradas
`GET /api/financial/relatorios/livro-entradas?start=2024-01-01&end=2024-12-31`

Lista todas as NF-e de entrada do período com seus itens e impostos.

**Resposta esperada:**
```json
[
  {
    "id": 1,
    "data_entrada": "2024-05-12",
    "numero_nf": 5500,
    "cnpj_emitente": "11222333000181",
    "razao_social_emitente": "Fornecedor XYZ LTDA",
    "valor_total": 5450.00,
    "valor_icms": 600.00,
    "valor_ipi": 250.00,
    "valor_pis": 82.50,
    "valor_cofins": 380.00
  }
]
```

---

### R02 — Livro de Saídas
`GET /api/financial/relatorios/livro-saidas?start=2024-01-01&end=2024-12-31`

Lista todas as NF-e de saída autorizadas do período.

---

### R03 — Impostos das Saídas
`GET /api/financial/relatorios/impostos-saidas?start=2024-01-01&end=2024-12-31`

Breakdown de impostos por CFOP nas saídas: ICMS, IPI, PIS, COFINS, DIFAL, diferido.

---

### R04 — Impostos das Entradas
`GET /api/financial/relatorios/impostos-entradas?start=2024-01-01&end=2024-12-31`

Créditos de impostos aproveitados nas entradas.

---

### R05 — DRE (Demonstração do Resultado)
`GET /api/financial/relatorios/dre?start=2024-01-01&end=2024-12-31`

DRE com **CMV real** calculado usando `stock_balances.avg_cost` (custo médio ponderado).

**Resposta esperada:**
```json
{
  "receita_bruta": 120000.00,
  "deducoes": 15000.00,
  "receita_liquida": 105000.00,
  "cmv": 62000.00,
  "lucro_bruto": 43000.00,
  "despesas_operacionais": 12000.00,
  "resultado_operacional": 31000.00
}
```

---

### R09 — Aging Receber Detalhado
`GET /api/financial/relatorios/aging-receber`

Detalhe de cada título em aberto com número de dias de atraso e cliente.

---

### R10 — Aging Pagar Detalhado
`GET /api/financial/relatorios/aging-pagar`

Detalhe de cada conta a pagar em aberto com dias de atraso e fornecedor.

---

### R11 — Extrato por Fornecedor
`GET /api/financial/relatorios/extrato-fornecedor/{id}?start=2024-01-01&end=2024-12-31`

Todas as contas a pagar (pagas e pendentes) de um fornecedor específico.

---

### R12 — Extrato por Cliente
`GET /api/financial/relatorios/extrato-cliente/{id}?start=2024-01-01&end=2024-12-31`

Todas as contas a receber de um cliente específico.

---

### R13 — Produtos Vendidos
`GET /api/financial/relatorios/produtos-vendidos?start=2024-01-01&end=2024-12-31`

Produtos vendidos no período com quantidade, receita, CMV (custo médio ponderado) e margem bruta.

**Resposta esperada:**
```json
[
  {
    "item_code": 10,
    "description": "Computador Portátil",
    "quantidade_vendida": 10,
    "receita_total": 10000.00,
    "cmv_total": 6000.00,
    "margem_bruta": 4000.00,
    "margem_percentual": 0.40
  }
]
```

---

### R14 — Produtos Produzidos
`GET /api/financial/relatorios/produtos-produzidos?start=2024-01-01&end=2024-12-31`

Produção do período com custo real da ordem de produção.

---

### R15 — Histórico de Custos
`GET /api/financial/relatorios/historico-custos?start=2024-01-01&end=2024-12-31`

Evolução do custo médio ponderado de cada item ao longo do tempo.

---

### R16 — Ficha Técnica com Custo
`GET /api/financial/relatorios/ficha-tecnica/{item_code}`

Estrutura de BOM do item com custo unitário de cada componente (do `stock_balances.avg_cost`) e custo total calculado.

**Resposta esperada:**
```json
[
  {
    "componente_code": 20,
    "componente_descricao": "Processador",
    "quantidade": 1,
    "unidade": "UN",
    "custo_unitario": 800.00,
    "custo_total_componente": 800.00
  }
]
```

---

### R17 — Curva ABC de Clientes
`GET /api/financial/relatorios/curva-abc-clientes?start=2024-01-01&end=2024-12-31`

Ranking de clientes por receita com classificação A/B/C usando acumulado percentual (janela SQL `SUM OVER`).

**Resposta esperada:**
```json
[
  {
    "cliente_id": 7,
    "razao_social": "Cliente Exemplo SA",
    "total_compras": 85000.00,
    "percentual_acumulado": 0.42,
    "classificacao": "A"
  }
]
```

> Classificação: A = até 80% acumulado, B = 80–95%, C = acima de 95%.

---

### R18 — Curva ABC de Produtos
`GET /api/financial/relatorios/curva-abc-produtos?start=2024-01-01&end=2024-12-31`

Ranking de produtos por receita com classificação A/B/C.

---

### R19 — Compras no Período
`GET /api/financial/relatorios/compras-periodo?start=2024-01-01&end=2024-12-31`

Resumo de compras por fornecedor e por produto, com valores de impostos recuperáveis.

---

## 13. Conciliação Bancária (OFX)

### `POST /api/financial/conciliacao/{conta_id}/importar-ofx`

Importa um arquivo de extrato bancário OFX e tenta conciliar automaticamente com o fluxo de caixa.

Suporta dois formatos:
- **OFX 1.x (SGML):** formato legado, usado pela maioria dos bancos brasileiros
- **OFX 2.x (XML):** detectado automaticamente pelo prefixo `<?xml`

**Request:**
```json
{
  "ofx_content": "OFXHEADER:100\nDATA:OFXSGML\n...<OFX>...</OFX>"
}
```

**O que acontece internamente:**
1. Parse do arquivo OFX (suporta datas `YYYYMMDD`, `YYYYMMDDHHMMSS`, `YYYYMMDDHHMMSS.000[-TZ]`)
2. Deduplicação por hash SHA-256 de `(conta_id | data | valor | FitID)` — transações já importadas são contadas como `duplicados` e ignoradas
3. Cada transação é salva em `extrato_bancario` com `ON CONFLICT DO NOTHING`
4. **Auto-match:** tenta casar cada transação do extrato com lançamentos do fluxo de caixa com valor igual (tolerância ±R$ 0,01) e data próxima (±3 dias)

**Resposta esperada (`200 OK`):**
```json
{
  "importados": 8,
  "duplicados": 2,
  "conciliados": 5
}
```

| Campo | Significado |
|---|---|
| `importados` | Transações novas gravadas no banco |
| `duplicados` | Transações que já existiam (hash idêntico) — ignoradas |
| `conciliados` | Transações que foram automaticamente casadas com o fluxo de caixa |

---

## 14. Validação de CNPJ/CPF

O pacote `internal/pkg/validation` expõe funções de validação usadas internamente.

```go
validation.ValidateCNPJ("12.345.678/0001-95")  // true
validation.ValidateCPF("123.456.789-09")         // true
validation.ValidateCNPJOrCPF("98765432000188")   // detecta automaticamente
```

**Regras implementadas:**
- Remove pontuação antes de validar
- Rejeita strings com todos os dígitos iguais (ex: `"11111111111"`)
- Calcula dois dígitos verificadores conforme algoritmo oficial (módulo 11)
- Para CNPJ: pesos `5,4,3,2,9,8,7,6,5,4,3,2` (1° dígito) e `6,5,4,3,2,9,8,7,6,5,4,3,2` (2° dígito)
- Para CPF: pesos `10..2` e `11..2`

---

## 15. Limitações conhecidas

| Item | Situação |
|---|---|
| **Endereço do emitente na NF-e** | Implementado. Configure via `PUT /api/fiscal/config` com os campos `logradouro`, `numero`, `bairro`, `municipio`, `codigo_municipio`, `cep` e `telefone`. Esses valores são enviados diretamente ao Focus NF-e. |
| **CT-e — autorização SEFAZ** | **Implementado.** Além do registro local, o CT-e pode ser autorizado na SEFAZ via Focus em `POST /api/fiscal/cte/{code}/authorize`. Os dados de remetente/destinatário/tomador/modal/municípios são enviados no campo `emission_data` (JSON) na criação; o emitente vem da config fiscal. Ver seção 6. |
| **Adiantamentos (advance payment)** | **Implementado.** Módulo de adiantamentos (`/api/financial/adiantamentos`): registra o adiantamento com movimento de caixa e aplica o saldo sobre contas a pagar/receber. Ver seção 40. |
| **Nota Fiscal de Serviços (NFS-e)** | **Implementado.** Módulo NFS-e (modelo ABRASF) via Focus em `/api/fiscal/nfse`: criação, autorização, cancelamento, consulta e listagem, com cálculo de ISS. Ver seção 41. |
| **Substituição Tributária (ST)** | **Implementado.** O motor calcula a ST quando o item informa `mva_pct` (MVA, podendo já ser a MVA ajustada). Fórmula: `BaseST = (BaseICMS + IPI) × (1 + MVA) × (1 − red_base_st_pct)` e `ICMS-ST = BaseST × alíq. interna do destino − ICMS próprio`. A alíquota interna do destino é resolvida da tabela de ICMS interno (interestadual) ou da config (interno), podendo ser sobreposta por `aliq_interna_destino_st`. O CST é promovido a `10`/`70` e os valores (`base_icms_st`, `aliq_icms_st`, `valor_icms_st`, `mva`) são persistidos por item e somados na nota; também são enviados no payload Focus NF-e. |
| **DANFE e XML** | **Implementado.** Ao autorizar a NF-e, os paths do DANFE e do XML são persistidos em `fiscal_exits.danfe_path` e `xml_path`. Consulte `GET /api/fiscal/exits/{id}/danfe` para obter as URLs absolutas (ver seção 4). |
| **Múltiplos ambientes simultâneos** | A config de ambiente (homologação/produção) é global. Não há separação por empresa. |

---

## 16. Parâmetros Fiscais — Cadastros de Apoio

### Dispositivos Legais

Cadastro de dispositivos legais referenciados nos parâmetros de ICMS/IPI.

| Endpoint | Descrição |
|----------|-----------|
| `POST /api/fiscal/support/dispositivos-legais/` | Criar dispositivo legal |
| `PUT /api/fiscal/support/dispositivos-legais/` | Atualizar dispositivo legal |
| `GET /api/fiscal/support/dispositivos-legais/` | Listar dispositivos (`?only_active=false` para inativos) |
| `GET /api/fiscal/support/dispositivos-legais/{code}` | Buscar por código |
| `GET /api/fiscal/support/dispositivos-legais/tipo/{type}` | Filtrar por tipo (`ICMS`, `IPI`, `LAUDO`, `PIS`, `COFINS`) |

**Payload `POST`:**
```json
{
  "type": "ICMS",
  "description": "Art. 12 do Dec. 45.490/2000 — Isenção ICMS"
}
```

---

### Naturezas de Operação (CFOP)

Cadastro dos códigos CFOP e suas classificações.

| Endpoint | Descrição |
|----------|-----------|
| `POST /api/fiscal/support/cfops/` | Criar CFOP |
| `PUT /api/fiscal/support/cfops/` | Atualizar CFOP |
| `GET /api/fiscal/support/cfops/` | Listar CFOPs (`?only_active=false`) |
| `GET /api/fiscal/support/cfops/{code}` | Buscar por código (ex: `5102`) |
| `GET /api/fiscal/support/cfops/direcao/{direction}` | Filtrar por direção (`ENTRADA`/`SAIDA`) |

**Payload `POST`:**
```json
{
  "code": 5102,
  "description": "Venda de mercadoria adquirida ou recebida de terceiros",
  "utilization": "INDUSTRIALIZACAO_COMERCIO",
  "ind_operacao": "NORMAL",
  "tipo_utilizacao": "NORMAL",
  "difal": false,
  "doacao": false
}
```

**Enums:**
- `utilization`: `INDUSTRIALIZACAO_COMERCIO` | `IMOBILIZADO` | `USO_CONSUMO`
- `ind_operacao`: `NORMAL` | `ENERGIA_ELETRICA` | `TELECOMUNICACAO`
- `tipo_utilizacao`: `NORMAL` | `VENDA_COMERCIAL_EXPORTADORA` | `COMPRA_FIM_ESPECIFICO_EXPORTACAO` | `EXPORTACAO`

---

### Parâmetros Básicos de ICMS/IPI por NCM/Item

Tabela simplificada de alíquotas e CSTs por NCM/Item + UF + Tipo de Operação. Para parametrização completa com FCI, DIFAL, substituição tributária, diferimento, benefícios fiscais e hierarquia de busca, utilize o módulo **Seção 28 — Cadastro de Redução/Substituição/Diferimento**.

| Endpoint | Descrição |
|----------|-----------|
| `POST /api/fiscal/support/parametros-icms-ipi/` | Criar parâmetro |
| `PUT /api/fiscal/support/parametros-icms-ipi/` | Atualizar parâmetro |
| `GET /api/fiscal/support/parametros-icms-ipi/` | Listar todos (`?only_active=false`) |
| `GET /api/fiscal/support/parametros-icms-ipi/{id}` | Buscar por ID |
| `GET /api/fiscal/support/parametros-icms-ipi/uf/{uf}` | Filtrar por UF (ex: `SP`) |
| `GET /api/fiscal/support/parametros-icms-ipi/item/{itemCode}` | Filtrar por código de item |
| `GET /api/fiscal/support/parametros-icms-ipi/ncm/{ncmCode}` | Filtrar por NCM |

**Payload `POST` (campos obrigatórios):**
```json
{
  "uf": "SP",
  "ncm_code": "84149000",
  "operation_type": "SAIDA",
  "icms_pct_contrib": 12.0,
  "icms_pct_non_contrib": 12.0,
  "cst_icms_contrib": "00",
  "cst_icms_non_contrib": "00"
}
```

> Forneça `ncm_code` **ou** `item_code` (nunca ambos). O campo `uf` é obrigatório.

**Enums `operation_type`:** `AMBAS` | `ENTRADA` | `SAIDA` | `CUSTOS`

---

## 17. Localização — Países e UFs

### Países

| Endpoint | Descrição |
|----------|-----------|
| `POST /api/location/countries/` | Criar país |
| `PUT /api/location/countries/` | Atualizar país |
| `GET /api/location/countries/` | Listar países (`?only_active=false`) |
| `GET /api/location/countries/{sigla}` | Buscar por sigla (ex: `BRA`) |
| `GET /api/location/countries/{sigla}/ufs` | Listar UFs de um país |

**Payload `POST`:**
```json
{
  "sigla": "BRA",
  "name": "Brasil",
  "ddi": "55",
  "bacen_code": "1058",
  "sis_comex": "BR"
}
```

### UFs

| Endpoint | Descrição |
|----------|-----------|
| `POST /api/location/ufs/` | Criar UF |
| `PUT /api/location/ufs/` | Atualizar UF |
| `GET /api/location/ufs/` | Listar UFs (`?only_active=false`) |
| `GET /api/location/ufs/{sigla}` | Buscar por sigla (ex: `SP`) |

> As 27 UFs brasileiras são **pré-populadas** pela migration `000125_countries_ufs.up.sql`.

---

## 18. Classificação de Itens

Cadastro hierárquico de classificações de itens. Usa **máscaras** para definir o formato dos códigos.

### Máscaras de Classificação

| Endpoint | Descrição |
|----------|-----------|
| `POST /api/items/classifications/masks/` | Criar máscara |
| `PUT /api/items/classifications/masks/` | Atualizar máscara |
| `GET /api/items/classifications/masks/` | Listar máscaras (`?only_active=false`) |
| `GET /api/items/classifications/masks/{code}` | Buscar máscara por código |
| `GET /api/items/classifications/masks/{maskID}/items` | Listar classificações de uma máscara |

### Classificações

| Endpoint | Descrição |
|----------|-----------|
| `POST /api/items/classifications/` | Criar classificação |
| `PUT /api/items/classifications/` | Atualizar classificação |
| `GET /api/items/classifications/{maskCode}/{code}` | Buscar classificação por máscara + código |
| `GET /api/items/classifications/{parentID}/children` | Listar filhos de uma classificação |

**Payload `POST` classificação:**
```json
{
  "code": "01.01",
  "mask_code": 1,
  "description": "Matéria-Prima Metálica",
  "parent_code": "01"
}
```

> O nível hierárquico (`level`) é calculado automaticamente a partir do `parent_code`.

---

## 19. Preços da Tabela de Vendas

Manutenção de preços por item dentro de uma tabela de vendas (migration `000127`).

| Endpoint | Descrição |
|----------|-----------|
| `POST /api/customers/support/sales-tables/` | Criar tabela de vendas |
| `GET /api/customers/support/sales-tables/` | Listar tabelas de vendas |
| `POST /api/customers/support/sales-tables/{tableID}/prices/` | Criar preço na tabela |
| `GET /api/customers/support/sales-tables/{tableID}/prices/` | Listar preços da tabela |
| `GET /api/customers/support/sales-tables/{tableID}/prices/{itemCode}` | Buscar preço por item |
| `PUT /api/customers/support/sales-tables/prices/` | Atualizar preço (envia `id` no body) |
| `DELETE /api/customers/support/sales-tables/prices/{id}` | Excluir preço por ID |

**Payload `POST`:**
```json
{
  "sales_table_id": 1,
  "item_code": "10001",
  "price": 149.90,
  "ume": "UN",
  "umc": "CX",
  "price_conv": 12.0,
  "situation": "ATIVO",
  "blocked": false,
  "observation": "Preço especial cliente A"
}
```

**Enums `situation`:** `ATIVO` | `INATIVO` | `PROMOCIONAL`

> O par `(sales_table_id, item_code)` é único. A conversão de preço entre unidades é gerida pelos campos `ume`, `umc` e `price_conv`.

---

## 20. Tipos de Movimento de Estoque

Cadastro de tipos de movimento de estoque — define o comportamento de cada movimentação (migration `000127`).

| Endpoint | Descrição |
|----------|-----------|
| `POST /api/estoque/tipos-movimento/` | Criar tipo de movimento |
| `PUT /api/estoque/tipos-movimento/` | Atualizar tipo de movimento |
| `GET /api/estoque/tipos-movimento/` | Listar (`?only_active=false`) |
| `GET /api/estoque/tipos-movimento/{id}` | Buscar por ID |
| `GET /api/estoque/tipos-movimento/sigla/{sigla}` | Buscar por sigla |

**Payload `POST`:**
```json
{
  "sigla": "ENT",
  "description": "Entrada por compra",
  "usage_type": "COMPRAS",
  "entry_order": true,
  "exit_order": false,
  "considers_consumption": false,
  "updates_avg_cost": true,
  "is_adjustment": false,
  "updates_cycle_count": false,
  "shows_in_summary": true,
  "entry_exit": "ENTRADA",
  "generates_fci_movement": false,
  "is_active": true
}
```

**Enums `usage_type`:** `PRODUCAO` | `COMPRAS` | `VENDAS` | `GERAL` | `AJUSTE` | `TRANSFERENCIA`

**Enums `entry_exit`:** `ENTRADA` | `SAIDA` | `TRANSFERENCIA` | `AMBOS`

---

## 21. Tipo de Nota Fiscal Saída — Campos Estendidos

Os campos abaixo foram adicionados à entidade `Tipo de NF Saída` pela migration `000126`.

### Novos campos fiscais

| Campo | Tipo | Descrição |
|-------|------|-----------|
| `description_nf` | string | Descrição específica para o corpo da NF |
| `impostos_nfe` | enum | Impostos que incidem no documento |
| `cfop_id` | int64 | CFOP padrão associado ao tipo de NF |
| `dispositivo_legal_ipi_id` | int64 | Dispositivo legal para IPI |
| `dispositivo_legal_icms_id` | int64 | Dispositivo legal para ICMS |
| `dispositivo_legal_icms_st_id` | int64 | Dispositivo legal para ICMS-ST |
| `dispositivo_legal_pis_id` | int64 | Dispositivo legal para PIS |
| `dispositivo_legal_cofins_id` | int64 | Dispositivo legal para COFINS |
| `hierarchy_ipi/icms/icms_st/pis/cofins` | string | Hierarquia de busca de alíquota por imposto |
| `ipi_transfer_sales_table_id` | int64 | Tabela de vendas para transferência IPI |

**Enum `impostos_nfe`:** `ICMS` | `IPI` | `PIS` | `COFINS` | `ICMS_IPI` | `TODOS`

### Flags SPED/SINTEGRA

| Flag | Descrição |
|------|-----------|
| `lista_valor_contabil` | Inclui valor contábil no SPED |
| `lista_registro_saida` | Lista registro de saída |
| `lista_icms_ipi` | Lista ICMS/IPI no livro fiscal |
| `sintegra_sped_fiscal` | Inclui no SINTEGRA/SPED |

### Flags de cálculo e comportamento

| Flag | Descrição |
|------|-----------|
| `calc_fomentar` | Calcula incentivo FOMENTAR/PRODUZIR |
| `excecao_fomentar` | Exceção ao benefício FOMENTAR |
| `comp_ress_ret_st` | Complemento/ressarcimento de ICMS-ST retido |
| `calc_reducao` | Aplica redução de base de cálculo |
| `complemento_itens` | NF complementar de itens |
| `busca_tipo_nf` | Busca automática do tipo de NF |
| `icms_st_ult_entrada` | Base ICMS-ST = valor da última entrada |
| `somente_consulta_lotes` | Apenas consulta lotes, não movimenta |
| `calc_imp_ibpt` | Calcula impostos IBPT (lei da transparência) |
| `cred_presumido_icms` | Usa crédito presumido de ICMS |
| `ciap` | Integra ao CIAP (crédito IPI/ICMS ativo permanente) |
| `vlr_agregado_base_subst` | Usa valor agregado como base de substituição |
| `contrato_facon` | Operação de facção/beneficiamento |
| `desc_icms_licitacoes` | Desconto ICMS em licitações |
| `sisdeclara` | Envia para SISDECLARA |

### Códigos de classificação

| Campo | Descrição |
|-------|-----------|
| `cod_clas_trib` | Código de classificação tributária |
| `cod_clas_trib_trib_reg` | Código de classificação para regime tributário |
| `cod_motivo_rest_comp_icms_st` | Código do motivo de ressarcimento/complemento ICMS-ST |
| `cod_beneficio_fiscal` | Código do benefício fiscal (cBenef) |

---

## 22. Motivos de Transferência DAPI

Cadastro de motivos de transferência utilizados na DAPI (migration `000128`).

| Endpoint | Descrição |
|----------|-----------|
| `POST /api/fiscal/support/motivos-transferencia-dapi/` | Criar motivo |
| `PUT /api/fiscal/support/motivos-transferencia-dapi/` | Atualizar motivo |
| `GET /api/fiscal/support/motivos-transferencia-dapi/` | Listar (`?only_active=false`) |
| `GET /api/fiscal/support/motivos-transferencia-dapi/{code}` | Buscar por código |

**Payload `POST`:**
```json
{
  "code": "01",
  "reason": "Transferência entre estabelecimentos",
  "destination": "SP",
  "valid_from": "2024-01-01",
  "is_active": true
}
```

---

## 23. Códigos de Ajuste ICMS — Tabela 5.1.1

Códigos de ajuste de apuração ICMS do SPED Fiscal (Tabela 5.1.1), por UF (migration `000128`).

| Endpoint | Descrição |
|----------|-----------|
| `POST /api/fiscal/support/codigos-ajuste-apuracao-icms/` | Criar código |
| `PUT /api/fiscal/support/codigos-ajuste-apuracao-icms/` | Atualizar código |
| `GET /api/fiscal/support/codigos-ajuste-apuracao-icms/` | Listar (`?only_active=false`) |
| `GET /api/fiscal/support/codigos-ajuste-apuracao-icms/{id}` | Buscar por ID |

**Payload `POST`:**
```json
{
  "code": "SP10000000",
  "uf": "SP",
  "description": "Ajuste a crédito — ICMS antecipado",
  "valid_from": "2024-01-01",
  "is_active": true
}
```

> O par `(code, uf)` é único.

---

## 24. Códigos de Ajuste ICMS — Tabelas 5.2 / 5.3 / 5.6 / 5.7

Códigos de ajuste para operações específicas: benefícios fiscais, incentivos, contribuições e estornos (migration `000128`).

| Endpoint | Descrição |
|----------|-----------|
| `POST /api/fiscal/support/codigos-ajuste-icms/` | Criar código |
| `PUT /api/fiscal/support/codigos-ajuste-icms/` | Atualizar código |
| `GET /api/fiscal/support/codigos-ajuste-icms/` | Listar (`?only_active=false`) |
| `GET /api/fiscal/support/codigos-ajuste-icms/{id}` | Buscar por ID |

**Payload `POST`:**
```json
{
  "uf": "SP",
  "code": "SP20000100",
  "description": "Benefício fiscal — isenção parcial",
  "table_ref": "5.2",
  "valid_from": "2024-01-01",
  "is_active": true
}
```

**Enums `table_ref`:** `5.2` | `5.3` | `5.6` | `5.7`

> A chave única é `(uf, code, table_ref)`.

---

## 25. Linhas de Apuração de ICMS

Linhas do bloco E do SPED Fiscal (apuração de ICMS) (migration `000128`).

| Endpoint | Descrição |
|----------|-----------|
| `POST /api/fiscal/support/linhas-apuracao-icms/` | Criar linha |
| `PUT /api/fiscal/support/linhas-apuracao-icms/` | Atualizar linha |
| `GET /api/fiscal/support/linhas-apuracao-icms/` | Listar (`?only_active=false`) |
| `GET /api/fiscal/support/linhas-apuracao-icms/{code}` | Buscar por código |

**Payload `POST`:**
```json
{
  "code": "E110",
  "description": "Saldo credor do período anterior",
  "line_type": "CREDITO",
  "accepts_entries": true,
  "nature": "Saldo credor transportado",
  "is_active": true
}
```

**Enums `line_type`:** `DEBITO` | `CREDITO` | `SALDO` | `DEDUCAO` | `OUTROS`

---

## 26. Lançamentos Resumo de ICMS

Resumo de ICMS por período, UF e CFOP — alimenta o bloco C do SPED Fiscal (migration `000128`).

| Endpoint | Descrição |
|----------|-----------|
| `POST /api/fiscal/support/lancamentos-resumo-icms/` | Criar lançamento |
| `PUT /api/fiscal/support/lancamentos-resumo-icms/` | Atualizar lançamento |
| `GET /api/fiscal/support/lancamentos-resumo-icms/` | Listar (`?only_active=false`) |
| `GET /api/fiscal/support/lancamentos-resumo-icms/{id}` | Buscar por ID |
| `POST /api/fiscal/support/lancamentos-resumo-icms/{id}/notas` | Adicionar nota ao lançamento |
| `GET /api/fiscal/support/lancamentos-resumo-icms/{id}/notas` | Listar notas do lançamento |

**Payload `POST` lançamento:**
```json
{
  "period": "2024-01",
  "uf": "SP",
  "cfop_id": 5,
  "icms_base": 10000.00,
  "icms_value": 1200.00,
  "is_active": true
}
```

**Payload `POST` nota:**
```json
{
  "note_number": "000123456",
  "note_series": "1",
  "emitter_cnpj": "12.345.678/0001-90",
  "issue_date": "2024-01-15",
  "item_value": 5000.00,
  "icms_base": 5000.00,
  "icms_value": 600.00
}
```

> O par `(period, uf, cfop_id)` é único por lançamento.

---

## 27. Apuração do Simples Nacional

Registro da apuração mensal do Simples Nacional por anexo (migration `000129`).

| Endpoint | Descrição |
|----------|-----------|
| `POST /api/fiscal/support/apuracao-simples-nacional/` | Criar apuração |
| `PUT /api/fiscal/support/apuracao-simples-nacional/` | Atualizar apuração |
| `GET /api/fiscal/support/apuracao-simples-nacional/` | Listar (`?only_active=false`) |
| `GET /api/fiscal/support/apuracao-simples-nacional/{period}/{annex}` | Buscar por período e anexo |

**Payload `POST`:**
```json
{
  "period": "2024-01",
  "annex": "I",
  "receita_interna": 80000.00,
  "receita_externa": 20000.00,
  "folha_pagamento": 15000.00,
  "receita_bruta_12m": 1200000.00,
  "simples_recolhido": 8500.00,
  "aliquota_nominal": 7.30,
  "aliquota_efetiva": 6.84,
  "aliquota_efetiva_icms": 1.25,
  "parcela_deduzir": 5940.00,
  "observation": "Apuração referente ao mês de janeiro/2024",
  "is_active": true
}
```

**Enums `annex`:** `I` | `II` | `III` | `IV` | `V` | `VI`

> O par `(period, annex)` é único. `period` deve estar no formato `YYYY-MM`.

---

## 28. Cadastro de Redução / Substituição / Diferimento de ICMS/IPI

Tabela de parametrização avançada de ICMS por item, NCM, UF, tipo de operação e segmento de cliente. Implementa a hierarquia de busca em 11 níveis (preferencial > item+máscara+cliente+estabelecimento > classificação).

| Endpoint | Descrição |
|----------|-----------|
| `POST /api/fiscal/icms-reducao/` | Criar registro |
| `PUT /api/fiscal/icms-reducao/` | Atualizar registro |
| `GET /api/fiscal/icms-reducao/` | Listar (`?uf=SP&item_id=123&active=true`) |
| `GET /api/fiscal/icms-reducao/find` | Buscar regra prioritária (`?uf=SP&item_id=1&customer_id=2&op_type=SAIDA`) |
| `GET /api/fiscal/icms-reducao/{id}` | Buscar por ID |

**Payload `POST` (campos obrigatórios mínimos):**
```json
{
  "uf": "SP",
  "operation_type": "SAIDA",
  "icms_pct_contrib": 12.0,
  "icms_pct_non_contrib": 12.0,
  "cst_icms_contrib": "00",
  "cst_icms_non_contrib": "00"
}
```

**Campos opcionais principais:**
- `item_id`, `item_mask`, `ncm_code` — escopo do item
- `customer_id`, `establishment_id`, `supplier_id`, `market_segment_id` — escopo do parceiro
- `invoice_type_out_id`, `invoice_type_in_id` — restringir ao tipo de nota
- `is_preferential: true` — sobrepõe qualquer outra regra (nível 1 da hierarquia)
- Campos de redução: `icms_red_pct_contrib`, `icms_red_target_contrib` (`BASE`|`PERCENTUAL`)
- Campos de diferimento: `icms_deferral_pct`, `icms_deferral_target`
- Campos de substituição tributária: `icms_subst_pct_contrib`, `mod_bc_icms_st`
- Campos FCI: `fci_icms_pct`, `fci_reduce_base`
- Campos DIFAL EC 87/2015: `difal_icms_red_pct`, `difal_icms_type`

**Enums:**
- `operation_type`: `ENTRADA` | `SAIDA` | `AMBAS` | `CUSTOS`
- `icms_red_target_contrib` / `icms_deferral_target` / `ipi_red_target_*`: `BASE` | `PERCENTUAL`

---

## 29. Aba Adicionais do Resumo de ICMS (C197 / Processos Judiciais)

Registros adicionais vinculados a um lançamento resumo de ICMS (tabela 5.x SPED). Mapeiam indicadores de arrecadação, processos judiciais e códigos DIEF.

| Endpoint | Descrição |
|----------|-----------|
| `POST /api/fiscal/icms-resumo-adicionais/` | Adicionar registro adicional |
| `GET /api/fiscal/icms-resumo-adicionais/{id}` | Listar adicionais do lançamento resumo `id` |

**Payload `POST`:**
```json
{
  "summary_entry_id": 42,
  "arrecadacao_indicator": "SEFAZ",
  "processo": "0012345-12.2023.8.26.0001",
  "description": "Decisão judicial — suspensão ICMS ST"
}
```

**Enums `arrecadacao_indicator`:** `SEFAZ` | `JUSTICA_FEDERAL` | `JUSTICA_ESTADUAL` | `OUTROS`

---

## 30. Restituição / Ressarcimento / Complementação de ICMS ST

Módulo para registro e geração de pedidos de restituição de ICMS ST conforme STF RE 593.849/MG. Mapeia os registros SPED Fiscal C180, C181, C185, C186, 1250 e 1251.

| Endpoint | Descrição |
|----------|-----------|
| `POST /api/fiscal/icms-st-restituicao/` | Criar pedido |
| `PUT /api/fiscal/icms-st-restituicao/` | Atualizar pedido |
| `GET /api/fiscal/icms-st-restituicao/` | Listar (`?empresa_id=1&period=2024-01&uf=SP`) |
| `GET /api/fiscal/icms-st-restituicao/{id}` | Buscar por ID |

**Payload `POST`:**
```json
{
  "empresa_id": 1,
  "period": "2024-01",
  "restitution_type": "RESTITUICAO",
  "uf": "SP",
  "orig_doc_model": "55",
  "orig_doc_number": "000001234",
  "orig_doc_date": "2024-01-15T00:00:00Z",
  "orig_emitter_cnpj": "12.345.678/0001-90",
  "item_code": "PROD-001",
  "cfop": "6102",
  "cst_icms": "10",
  "icms_st_base": 1000.00,
  "icms_st_aliq": 12.0,
  "icms_st_value": 120.00,
  "icms_st_base_restitution": 850.00,
  "icms_st_value_restitution": 102.00
}
```

**Enums `restitution_type`:** `RESTITUICAO` | `RESSARCIMENTO` | `COMPLEMENTACAO`

> `period` deve estar no formato `YYYY-MM`. O par `(empresa_id, period, uf)` pode ter múltiplos registros (um por nota/item).

---

## 31. Notas Especiais de Ajuste

Notas fiscais complementares e de ajuste emitidas para correção de apuração de ICMS. Suporta geração automática de lançamento resumo (`auto_generate_summary`). Tipo `COMPLEMENTAR` complementa base/alíquota; tipo `AJUSTE` gera linha de ajuste na apuração.

| Endpoint | Descrição |
|----------|-----------|
| `POST /api/fiscal/notas-especiais/` | Criar nota (status inicial: RASCUNHO) |
| `PUT /api/fiscal/notas-especiais/` | Atualizar nota |
| `GET /api/fiscal/notas-especiais/` | Listar (`?empresa_id=1&period=2024-01`) |
| `GET /api/fiscal/notas-especiais/{id}` | Buscar nota por ID |
| `POST /api/fiscal/notas-especiais/{id}/itens` | Adicionar item à nota |
| `GET /api/fiscal/notas-especiais/{id}/itens` | Listar itens da nota |

**Payload `POST` nota:**
```json
{
  "empresa_id": 1,
  "purpose": "AJUSTE",
  "issue_date": "2024-01-31T00:00:00Z",
  "period": "2024-01",
  "cfop_id": 12,
  "icms_apuracao_line_id": 3,
  "adjustment_code_id": 7,
  "auto_generate_summary": true,
  "total_value": 5000.00,
  "total_icms": 600.00,
  "observation": "Ajuste referente ao diferimento parcial"
}
```

**Payload `POST` item:**
```json
{
  "item_code": "MP-001",
  "description": "Matéria Prima X",
  "quantity": 100.0,
  "unit": "KG",
  "unit_value": 50.0,
  "total_value": 5000.00,
  "icms_base": 5000.00,
  "icms_pct": 12.0,
  "icms_value": 600.00,
  "cst_icms": "00",
  "cfop_id": 12
}
```

**Enums `purpose`:** `COMPLEMENTAR` | `AJUSTE`
**Enums `status`:** `RASCUNHO` | `EMITIDA` | `CANCELADA`

> O campo `period` deve estar no formato `YYYY-MM`. A sequência dos itens é atribuída automaticamente.

---

## 32. SPED Contábil — ECD (Escrituração Contábil Digital)

Módulo de escrituração contábil com geração do arquivo SPED ECD (Blocos 0, I, J, 9). Cobre plano de contas, contas contábeis, lançamentos contábeis e demonstrativos financeiros.

### Planos de Contas

| Endpoint | Descrição |
|----------|-----------|
| `POST /api/accounting/plans/` | Criar plano de contas |
| `GET /api/accounting/plans/` | Listar planos (`?empresa_id=1`) |

**Payload:**
```json
{
  "empresa_id": 1,
  "name": "Plano de Contas 2024",
  "year": 2024,
  "is_active": true
}
```

### Contas Contábeis

| Endpoint | Descrição |
|----------|-----------|
| `POST /api/accounting/accounts/` | Criar conta |
| `GET /api/accounting/accounts/` | Listar contas (`?plan_id=1`) |

**Payload:**
```json
{
  "plan_id": 1,
  "code": "1.1.1.01",
  "name": "Caixa",
  "account_type": "ANALITICA",
  "nature": "DEVEDORA",
  "parent_id": null
}
```

### Lançamentos Contábeis

| Endpoint | Descrição |
|----------|-----------|
| `POST /api/accounting/journal-entries/` | Criar lançamento |
| `GET /api/accounting/journal-entries/` | Listar (`?empresa_id=1&period=2024-01`) |

**Payload:**
```json
{
  "empresa_id": 1,
  "entry_date": "2024-01-31T00:00:00Z",
  "period": "2024-01",
  "history": "Venda de mercadorias ref. NF 1234",
  "debit_account_id": 10,
  "credit_account_id": 25,
  "value": 5000.00
}
```

### Demonstrativos Contábeis

| Endpoint | Descrição |
|----------|-----------|
| `POST /api/accounting/demonstratives/` | Criar demonstrativo (DRE, BP, etc.) |

**Payload:**
```json
{
  "empresa_id": 1,
  "type": "DRE",
  "period": "2024-01",
  "name": "DRE Janeiro 2024"
}
```

### Geração do Arquivo SPED ECD

| Endpoint | Descrição |
|----------|-----------|
| `POST /api/accounting/sped/ecd` | Gerar arquivo SPED Contábil (ECD) |

**Payload:**
```json
{
  "empresa_id": 1,
  "period": "2024-01",
  "cnpj": "12.345.678/0001-90",
  "razao_social": "Empresa Exemplo S.A.",
  "municipio": "São Paulo",
  "uf": "SP"
}
```

O endpoint retorna o arquivo `.txt` pipe-delimitado como attachment (`Content-Disposition: attachment`), pronto para transmissão à SEFAZ via PVA.

**Blocos gerados:** `0000`, `0001`, `0990`, `I001`, `I010`, `I050`, `I100`, `I200`, `I300`, `I350`, `I990`, `J001`, `J005`, `J100`, `J150`, `J990`, `K001`, `K990`, `9001`, `9900`, `9990`, `9999`

---

## 33. Cadastro de Fornecedores (integração fiscal)

O cadastro de fornecedores/transportadoras é documentado em detalhe em
[`cadastros-fornecedor.md`](cadastros-fornecedor.md). Esta seção resume os
pontos que tocam o módulo fiscal.

### Campos fiscais do fornecedor

- **Contrib. ICMS** (`icms_contributor`): `CONTRIBUINTE`, `NAO_CONTRIBUINTE` ou
  `ISENTO` — usado no enquadramento da NF de entrada.
- **Inscrição Estadual** (`state_registration`): obrigatória, exceto para fornecedores
  do tipo `TRANSPORTADORA`, `TRANSP_REDESP` ou `REDESPACHO`.
- **Tipo Frete** (`freight_type`): default do frete no pedido de compra / NF de entrada.
- **Registro M.A.**, **Obr. Vitícola**, **MEI**, **GLN** e os campos de consulta SEFAZ
  (situação de recebimento, última consulta) compõem o cadastro.

### Vínculo por empresa (pasta Empresas → `supplier_enterprises`)

Por empresa, o fornecedor define a **conta financeira**, o checkbox **IPI**, a
**tabela de preço de compra** e o **Tipo de NF default** (`default_invoice_type_id`),
usado na entrada conforme a hierarquia do sistema (Cadastro de Redução/Substituição
de ICMS → Cadastro de Fornecedores).

### Redução / Substituição / Diferimento de ICMS

A tabela `icms_reduction_substitutions` (seção 28) possui o filtro `supplier_id`,
permitindo regras tributárias específicas por fornecedor. Seguindo a convenção do
módulo fiscal, a coluna é um `BIGINT` sem FK rígida (validação em nível de aplicação).

### Provider de defaults

`GET /api/suppliers/{code}/purchasing-defaults?enterprise=<código>` retorna, num único
payload, os defaults consumidos pelo pedido de compra e pela entrada fiscal: condição
de pagamento, tipo de frete, contribuinte de ICMS, inscrição estadual, conta
financeira, Tipo de NF default e tabela de preço de compra.

---

## 34. Cadastro de Classificações Fiscais

Cadastro da **classificação fiscal de mercadorias** (migration `000137`), base para o
cálculo de I.I., IPI, PIS, COFINS e ICMS por item. Tabela `fiscal_classifications` +
filhas `fiscal_classification_languages` (idiomas) e
`fiscal_classification_export_attributes` (atributos de exportação SISCOMEX).

### Campos

- Identificação: **Classificação** (`code`), **Descrição**, **NCM**, **CEST**, **Ex Tarifário**.
- **IPI**: `ipi_rate` + `ipi_indicator` (`PERCENTUAL`/`VALOR`), **Apuração** (periodicidade),
  CST IPI entrada/saída, **UN p/ IPI**, **UN de Tributação**.
- **PIS** e **COFINS**: alíquota + indicador, CST entrada/saída, **COFINS Majorado**,
  além de variantes **ST**, **Consumo** (com CSTs), **Retenção** (com CST), **Redução**
  (com CST) e **Desconto ZF** (Zona Franca de Manaus) para PIS e COFINS.
- **ICMS**: modalidade da BC (`mod_bc_icms`) e da BC ST (`mod_bc_icms_st`).
- **CBS/IBS**: `cod_clas_trib` e `cod_clas_trib_trib_reg` (Tributação Regular).
- **Obs Fiscal** (`obs_fiscal`): texto incorporado à tag `infAdFisco` da NF-e.

### Pastas

- **Idiomas** — descrição da classificação por idioma.
- **Atributos de Exportação** — por NCM: código, descrição, domínio e vigência
  (`start_date`/`end_date`); a mesma NCM pode ter vários atributos/domínios.

### Endpoints (`/api/fiscal-classifications`)

- `POST /` — criar (gera `code`); `PUT /` — atualizar; `GET /` — listar
  (`?only_active=false` inclui inativas); `GET /{code}` — obter (com idiomas e atributos).
- `POST /languages` — adicionar/atualizar descrição por idioma.
- `POST /export-attributes` — adicionar atributo de exportação.

> Esta classificação será consumida pelo cálculo de impostos do item do Pedido de
> Compra (%IPI da classificação, hierarquia com o Cadastro de Redução/Substituição de
> ICMS — seção 28) nas fases seguintes do módulo de compras.

> O par `(empresa_id, period)` identifica um ECD. Utilize um plano de contas ativo com pelo menos uma conta analítica antes de gerar o arquivo.

---

## 35. Tipos de Operação de Entrada

Cadastro de **Tipos de Operação de Entrada** (migration `000144`) usado na inclusão da
NF de entrada. Tabelas: `entry_operation_types` + **Grupo de Estado** (`state_groups`
+ `state_group_ufs`).

### Campos

Código, descrição, **Tipo de Nota** (`invoice_type_code`), **Natureza de Operação**
(`nature_operation`), classificação (tipo + código), **Grupo Estado**
(`state_group_code`) e **Tipo Fornecedor** (`supplier_type_code`).

### Validação UF × Natureza

O 1º dígito da natureza determina a regra contra a UF da empresa:

| 1º dígito | Significado | Regra |
| --- | --- | --- |
| `1` | dentro do estado | UF da empresa **deve** pertencer ao Grupo de Estado |
| `2` | fora do estado | UF da empresa **não deve** pertencer ao Grupo de Estado |
| `3` | fora do país | operação estrangeira (sem validação de grupo) |

`GET /api/entry-operations/{code}/validate?uf=XX` retorna `{valid, reason}`.

### Endpoints

- `/api/entry-operations` — `POST` · `PUT` · `GET` · `GET /{code}` · `GET /{code}/validate`.
- `/api/entry-operations/state-groups` — `POST` · `GET` · `GET /{code}` ·
  `POST /{code}/ufs` (adicionar UF ao grupo).

## 36. Manifestação do Destinatário e Inutilização de Numeração

Operações de eventos da NF-e via FocusNFE (exigem token configurado em §2 e o
escopo `fiscal:authorize`).

### Manifestação do Destinatário
`POST /api/fiscal/manifestacao` — body `{ "chave_nfe": "<44 dígitos>", "tipo": "ciencia|confirmacao|desconhecimento|nao_realizada", "justificativa": "..." }`.
Usa o CNPJ da empresa (config fiscal). `justificativa` é obrigatória para
`desconhecimento`/`nao_realizada`. Cliente: `focusnfe.ManifestarDestinatario`.

### Inutilização de Numeração
`POST /api/fiscal/inutilizacao` — body `{ "serie": 1, "numero_inicial": 100, "numero_final": 110, "justificativa": "..." }`.
Invalida no SEFAZ uma faixa de números **não utilizados**. Cliente:
`focusnfe.InutilizarNumeracao`.

> Junto com **Carta de Correção** (CC-e) e **Cancelamento** (§4), completam os
> eventos fiscais da NF-e.

## 37. IBPT / SCI — Carga Tributária Aproximada (Lei da Transparência) — migration 000145

Importa a tabela oficial **IBPT (TabelaIBPTax)** por UF e permite consultar a carga
tributária aproximada por NCM (Lei 12.741/2012). Tabela `ibpt_rates`
(NCM/EX/UF/versão único; %federal nacional/importado, estadual, municipal,
vigências, chave, fonte).

### Endpoints (`/api/fiscal/ibpt`)
| Ação | Endpoint | Escopo |
|---|---|---|
| Importar CSV | `POST /api/fiscal/ibpt/import` (`{ "uf": "PR", "csv": "<conteúdo do arquivo>" }`) | `admin` |
| Consultar NCM | `GET /api/fiscal/ibpt/lookup?ncm=72085400&uf=PR` | ADMIN/USER |

O parser aceita o CSV oficial (delimitado por `;`, decimais com vírgula) com as
colunas `codigo;ex;tipo;descricao;nacionalfederal;importadosfederal;estadual;municipal;vigenciainicio;vigenciafim;chave;versao;fonte`; faz **upsert** por (NCM, EX, UF, versão).

## 38. CNAB 240 — Remessa de Boletos

`POST /api/financial/cnab/remessa-240` (escopo `financial:manage`) gera o arquivo
de **remessa CNAB 240** (layout-padrão FEBRABAN) a partir de uma configuração de
cedente/banco e uma lista de títulos. Retorna `text/plain` (anexo `remessa.rem`),
com registros de 240 colunas: header de arquivo/lote, segmentos **P** e **Q** por
título, trailers de lote/arquivo. Gerador: `internal/infrastructure/cnab`.

**Perfis por banco.** O gerador agora resolve um **perfil por código de banco**
(`internal/infrastructure/cnab` → `bankProfiles`) que ajusta os campos que mais
divergem dentro do padrão FEBRABAN: **carteira** (segmento P, pos. 058), **espécie
do título** (pos. 107-108) e as **versões de layout** do arquivo (pos. 164-166) e do
lote (pos. 014-016). Os trailers de lote/arquivo passam a gravar o **código do
banco** (antes fixo em `000`). Perfis embutidos: Itaú `341`, Bradesco `237`,
Santander `033`, Banco do Brasil `001` e Caixa `104`; bancos não mapeados usam um
perfil padrão. Qualquer campo pode ser sobreposto explicitamente em `RemessaConfig`
(`Carteira`, `EspecieTitulo`, `LayoutArquivo`, `LayoutLote`).

> ⚠️ Os perfis cobrem as divergências mais comuns, mas a composição do
> **nosso-número** e blocos de convênio ainda variam por contrato/carteira —
> **homologar com o banco** antes de uso em produção.

## 39. Balancete (Contábil)

`GET /api/accounting/balancete?plan_id=&empresa_id=&from=YYYY-MM-DD&to=YYYY-MM-DD`
agrega os **lançamentos contábeis** do período por conta (débitos e créditos),
devolvendo saldo por conta, totais e o indicador `balanced` (partidas dobradas:
total de débitos = total de créditos). Implementado em
`accounting_uc.BalanceteUseCase`.

> Os demais relatórios gerenciais — **DRE** (`/api/financial/relatorios/dre`),
> **Curva ABC** de clientes/produtos e **conciliação de extrato** — já existem no
> módulo financeiro (§12–§13).

---

## 40. Adiantamentos (Advance Payments) — migration 000148

Controle de **adiantamentos** a fornecedores (PAGAR) e de clientes (RECEBER), com
aplicação do saldo sobre contas a pagar/receber. Tabelas: `adiantamentos` (saldo) e
`adiantamento_aplicacoes` (auditoria de cada aplicação). Use case:
`financial_uc` (`CreateAdiantamentoUseCase`, `AplicarAdiantamentoUseCase`, etc.).

### `POST /api/financial/adiantamentos/create`

Registra um adiantamento e **movimenta o caixa** na mesma transação: tipo `PAGAR`
gera saída (paga-se o fornecedor antecipadamente); tipo `RECEBER` gera entrada
(cliente paga antecipadamente). Atualiza o saldo da conta bancária.

**Request:**
```json
{
  "tipo": "PAGAR",
  "parceiro_id": 3,
  "conta_bancaria_id": 1,
  "numero_documento": "ADV-001",
  "data_adiantamento": "2024-05-10",
  "valor_original": 2000.00,
  "descricao": "Sinal de pedido de compra 15"
}
```

**Resposta esperada (`201 Created`):** o adiantamento com `status: "ABERTO"` e
`valor_utilizado: 0`.

### `POST /api/financial/adiantamentos/{id}/aplicar`

Aplica parte (ou todo) o saldo do adiantamento sobre um título. **Não move caixa**
(o dinheiro já se moveu na criação) — apenas quita o título contra o adiantamento.

**Request:**
```json
{ "conta_tipo": "PAGAR", "conta_id": 42, "valor": 500.00, "data_aplicacao": "2024-06-10" }
```

**Regras:** valida que o tipo do adiantamento casa com `conta_tipo`, que há saldo no
adiantamento e que o valor não excede o saldo do título. Para `PAGAR`, abate em
`valor_adiantamento_abatido` e marca `PAGO` quando quitado; para `RECEBER`, soma em
`valor_recebido` e marca `RECEBIDO` quando quitado. Atualiza o `valor_utilizado` e o
status do adiantamento (`ABERTO`→`PARCIAL`→`QUITADO`).

### `GET /api/financial/adiantamentos/list`

Lista adiantamentos. Filtros opcionais: `?tipo=PAGAR` e `?parceiro_id=3`.

### `GET /api/financial/adiantamentos/{id}`

Retorna o adiantamento com o **saldo** e o histórico de **aplicações**.

---

## 41. NFS-e (Nota Fiscal de Serviços eletrônica) — migration 000150

Módulo de **NFS-e** (modelo ABRASF) emitida via Focus. O prestador é a empresa
(config fiscal); a tabela `nfse` guarda o RPS, o tomador, o serviço e o resultado da
autorização. Use cases em `nfse_uc`.

### `POST /api/fiscal/nfse/create`

Cria a NFS-e em rascunho e **calcula o ISS** (`base = valor_servicos − deducoes`,
`ISS = base × aliquota_iss`) e o valor líquido (se `iss_retido`, desconta o ISS).

**Request:**
```json
{
  "numero_rps": 100,
  "serie_rps": "1",
  "tipo_rps": 1,
  "data_emissao": "2024-05-15",
  "natureza_operacao": 1,
  "optante_simples": false,
  "tomador_cnpj_cpf": "98765432000188",
  "tomador_razao_social": "Cliente Serviços SA",
  "tomador_email": "financeiro@cliente.com",
  "tomador_codigo_municipio": "3550308",
  "tomador_uf": "SP",
  "item_lista_servico": "14.01",
  "codigo_tributario_municipio": "140100",
  "discriminacao": "Manutenção de equipamento industrial",
  "codigo_municipio": "4106902",
  "valor_servicos": 1000.00,
  "valor_deducoes": 0.00,
  "aliquota_iss": 0.05,
  "iss_retido": false
}
```

**Resposta esperada (`201 Created`):** a NFS-e com `status: "RASCUNHO"`, `valor_iss`
e `valor_liquido` calculados.

### `POST /api/fiscal/nfse/{code}/authorize`

Transmite a NFS-e para a prefeitura via Focus (escopo `fiscal:authorize`). Monta o
payload ABRASF (prestador da config fiscal + tomador + serviço), faz *poll* até
estado terminal e, autorizada, grava `numero_nfse`, `codigo_verificacao`, `url` e
`focus_ref` com `status: "AUTORIZADA"`.

### `POST /api/fiscal/nfse/{code}/cancel`

Cancela uma NFS-e autorizada na prefeitura. **Request:** `{ "justificativa": "..." }`
(mínimo 15 caracteres).

### `GET /api/fiscal/nfse/list` · `GET /api/fiscal/nfse/{code}`

Lista as NFS-e e retorna uma por ID.

> **Observação:** o layout da NFS-e **varia por município**. Os campos
> `item_lista_servico`, `codigo_tributario_municipio` e `codigo_municipio` devem
> seguir a tabela da prefeitura; homologue com o município antes de produção.
