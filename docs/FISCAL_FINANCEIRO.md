# Módulo Fiscal & Financeiro — Venture ERP

> **Autenticação:** todos os endpoints exigem `Authorization: Bearer <JWT>`.
> Obtenha o token em `POST /users/login`.
> Todas as requests usam `Content-Type: application/json`.

---

## Índice

1. [Visão geral da arquitetura](#1-visão-geral-da-arquitetura)
2. [Pré-requisito: Configuração Fiscal](#2-pré-requisito-configuração-fiscal)
3. [Motor Tributário](#3-motor-tributário)
   - 3.1 [Gestão de Tabelas Tributárias](#31-gestão-de-tabelas-tributárias)
4. [Módulo Fiscal — NF-e de Saída](#4-módulo-fiscal--nf-e-de-saída)
5. [Módulo Fiscal — NF-e de Entrada](#5-módulo-fiscal--nf-e-de-entrada)
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

**Request:**
```json
{
  "cnpj_empresa": "12345678000195",
  "razao_social": "Empresa Teste LTDA",
  "ie_empresa": "1234567890",
  "regime_tributario": "3",
  "uf_empresa": "PR",
  "logradouro": "Rua das Industrias",
  "numero": "100",
  "complemento": "Galpão A",
  "bairro": "Distrito Industrial",
  "municipio": "Curitiba",
  "codigo_municipio": "4106902",
  "cep": "80000000",
  "telefone": "41999990000",
  "icms_interno_aliquota": 0.12,
  "icms_diferimento_percentual": 38.46,
  "focus_nfe_token": "Djya39nqXUn3w93TB2dvPGBda3ho1mY1",
  "focus_nfe_ambiente": "homologacao",
  "juros_mes": 0.01,
  "multa_atraso": 0.02,
  "vencimento_icms_dia": 10,
  "vencimento_ipi_dia": 15,
  "vencimento_pis_cofins_dia": 25
}
```

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
2. POST /api/fiscal/exits/{id}/authorize → envia para Focus NF-e → status: autorizada
3. (opcional) POST /exits/{id}/carta-correcao → CC-e para correção de dados
4. (opcional) POST /exits/{id}/cancel → cancela NF-e autorizada
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

### `POST /api/fiscal/exits/{id}/authorize`

Envia a NF-e para a SEFAZ via API Focus NF-e. Só funciona se o status for `"rascunho"`.

**Request:** body vazio `{}`

**O que acontece internamente:**
1. Busca os dados da NF-e e seus itens
2. Lê o token Focus NF-e da configuração fiscal
3. Monta o payload conforme layout Focus NF-e v2
4. Chama `POST https://homologacao.focusnfe.com.br/v2/nfe/{ref}?substituicao=true`
5. Salva a chave de acesso, protocolo e ref retornados pelo Focus
6. Cria automaticamente uma **Conta a Receber** vinculada à NF-e (vencimento em 30 dias)
7. Registra o log da requisição/resposta no banco

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

### `POST /api/fiscal/exits/{id}/cancel`

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

### `POST /api/fiscal/exits/{id}/carta-correcao`

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

### `GET /api/fiscal/exits/{id}`

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

## 5. Módulo Fiscal — NF-e de Entrada

NF-e de entrada representa compras/recebimento de mercadorias. Os impostos são informados pelo emitente (não calculados pelo sistema).

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

O CT-e é registrado localmente para fins de custeio do frete e vinculação com NF-e de entrada. **Não há integração com SEFAZ para autorização do CT-e** (apenas NF-e de saída usa Focus API).

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
| **CT-e — autorização SEFAZ** | O CT-e é registrado localmente mas **não é enviado para a SEFAZ**. A integração Focus para CT-e não foi implementada — apenas NF-e de saída usa o Focus API. |
| **Adiantamentos (advance payment)** | O schema possui o campo `adiantamento` em contas a pagar/receber, mas não há use case específico para aplicar adiantamentos. |
| **Nota Fiscal de Serviços (NFS-e)** | Não implementado. O sistema cobre apenas NF-e (modelo 55) e CT-e. |
| **Substituição Tributária (ST)** | O motor tributário não calcula MVA/ST. O CST de ST pode ser informado manualmente nos itens. |
| **Múltiplos ambientes simultâneos** | A config de ambiente (homologação/produção) é global. Não há separação por empresa. |
| **DANFE e XML** | O sistema não gera DANFE. O PDF e o XML ficam disponíveis pela URL do Focus NF-e após a autorização. |
