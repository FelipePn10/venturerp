# Roadmap de maturidade — Comercial

Plano de evolução do setor **Comercial** do VentureERP, usando como referência
pública a árvore de telas do FoccoERP (`help.foccoerp.com.br`). O setor
**Compras / Suprimentos** já foi fechado no backend (ver
[`maturidade-erp-roadmap.md`](maturidade-erp-roadmap.md)); Comercial é o próximo.

> **Importante:** os códigos `FxxxNNNN`/`CxxxNNNN` são **apenas referência do Focco**
> para localizar a rotina equivalente. **Não** entram no backend: nada de pastas,
> tabelas, rotas, variáveis ou funções com esses nomes. Modelamos os conceitos na
> linguagem do VentureERP (ex.: "orçamento" → `sales_quote`, "representante" →
> `sales_rep`).

## Situação atual (o que o Comercial já tem)

Maduro: **Pedido de Venda** (crédito, reserva, status/demanda, itens), **Cliente**
(`FCLI0200` — segmento de mercado, limite de crédito, formação de preço,
transportadora), **Expedição/Romaneio** (Outbound Delivery), **Faturamento** (NF-e /
NFS-e de saída, autorização fiscal), **Previsão de Vendas** (forecast/blocos/rateio) e
**Promessa de Entrega** (ATP/params/calendário). Recebíveis são gerados da saída
fiscal (Financeiro).

## Matriz de construção (14 módulos)

Cada módulo lista as telas Focco de referência e a situação no VentureERP.

### 1. Precificação — ❌ tabela de preço de venda ausente (só há campo de referência)
- `FPRV0200` — Cadastro da Tabela de Vendas
- `FPRV0201` — Cadastro de Preços da Tabela de Vendas
- `FCST0205` — Formação do Preço de Venda (módulo Custos)
- `FCST0262 PREC` — Precificação de Produtos
- `FPDV0200` (política) — Cadastro de Política de Formação de Preço de Venda

### 2. Política Comercial — ❌ ausente (motor de desconto/acréscimo/comissão)
- `FPDV0108` — Política Comercial de Descontos
- `FPDV0109` — Política Comercial de Acréscimo
- `FPDV0110` — Política Comercial de Fretes
- `FPDV0115` — Cadastro de Regras (configurador de produto)
- `FPDV0117` — Itens/classificações com políticas específicas
- `FPDV0250` — Relatório de política de descontos/acréscimos e comissões

### 3. Orçamento — ❌ ausente
- `FPDV0200 ORC` — Cadastro de Orçamentos
- `FPDV0205 ORC` — Cancelamento/Atendimento de Orçamentos
- `CPDV0410 ORC` — Consulta de Orçamentos
- `FPDV0206 ORC` — Relatório de Orçamentos
- Conversão → `FPDV0200 PDV`

### 4. Pedido de Venda — ✅ base existe; validar completude
- `FPDV0200 PDV` — Cadastro de Pedido de Venda *(validar se completo)*
- `FCST0262 PREC` — Precificação de Produtos
- `FPDV0202` — Análise Financeira/Comercial dos Pedidos
- `FPDV0203 COM` — Liberação Comercial dos Pedidos
- `FPDV0203 FIN` — Liberação Financeira dos Pedidos
- `FPDV0204 ENG` — Liberação de Itens do Pedido
- `FPDV0205 PDV` — Cancelamento/Atendimento de Pedidos
- `FPDV0210` — Conferência de Pedidos
- `FPDV0211` — Gera Pedidos de Transferência
- `FPDV0239` — Cadastro de Ferramentas para Amortização
- `FPDV0272` — Manutenção de Datas de Entrega
- `FPDV0251` — Desatendimento de Pedidos de Venda
- `FPDC0200 ORD` — Ordem de Recebimento de Devoluções
- `FPDC0200 ORM` — Ordem de Recebimento de Materiais
- Consultas: `CPDV0410 PDV`, `CPDV0411`, `CPRV0400`
- Relatórios: `FPDV0206 PDV`, `FPDV0301`–`FPDV0311`

### 5. Representantes — ❌ ausente (prioridade alta)
- `FREP0200` — Cadastro de Representantes
- `FREP0101` — Cadastro de Tipos de Representantes
- `FREP0251` — Relatório de Representantes
- `FREP0253` — Ficha de Acompanhamento de Representantes
- Comissão → Política Comercial (`FPDV0250`)

### 6. Meta de Vendas — ❌ ausente
- `FMET0100` — Cadastro base de Metas
- `FMET0200` — Cadastro de Metas
- `FMET0201` — Cadastro de Metas por Grupo Comercial
- `FMET0202` — Cadastro de Saldos de Metas
- `FMET0300` — Relatório de Metas

### 7. Previsão de Vendas — ✅ existe; alinhar às telas do Focco
- `FPRE0201` — Cadastro de Previsão de Vendas
- `FPRE0251` — Geração de Previsão de Vendas

### 8. Promessa de Entrega — ✅ parcial (ATP); evoluir p/ CTP + tanque
- `FPME0200` — Manutenção da Promessa de Entrega
- `FPME0201` — Reprogramação das Datas de Entrega
- `FPME0203` — Comprometimento de Tanque

### 9. Assistência Técnica — ⚠️ mínimo (só flag em sales_division)
- `FASS0101` — Grupos e Motivos de Defeitos
- `FASS0102` — Responsáveis pela Garantia
- `FASS0201` — Cadastro de Chamados de Assistência Técnica
- `FASS0200` — Notas de Devolução (Remessa Garantia / RMA)
- `FASS0202` — Geração de Ordens de Assistência Técnica
- `CASS0402` — Consulta de Assistência Técnica
- `FASS0302` — Relatório de Chamados de Assistência Técnica

### 10. Atendimento ao Consumidor (SAC) — ❌ ausente
- `FATC0200` — Cadastro de Consumidores
- `FATC0201` — Cadastro de Contato com Cliente
- `FATC0280` — Cadastro de Chamados
- `FATC0302` — Etiqueta de Consumidores
- `CATC0480` — Consulta de Chamados
- `FATC0380` — Relatório de Chamados
- `FATC0301` — Relatório de Histórico de Chamados

### 11. Central de Vendas — ❌ ausente (cockpit)
- `FCVN0200` — Central de Vendas
- `FCVN0201` — Nova Venda
- `FCVN0202` — Consulta de Pedidos e Orçamentos

### 12. Vendas Recorrentes — ❌ ausente
- `FVRE0200` — Console de Vendas Recorrentes
- `FVRE0202` — Consulta de Receita Recorrente Mensal
- `FVRE0203` — Consulta de Comissões Futuras

### 13. Expedição — ✅ romaneio forte, mas SEM carga/rota/monitores do Focco
- `FITE0251` — Geração de Almoxarifados de Assistência Técnica
- `FPLC0200` — Manutenção de Carga
- `FPLC0201` — Manutenção de Cargas
- `FPLC0202` — Inclusão de Notas para Manifesto de Carga
- `FPLC0208` — Controle de Carregamento
- `FPLC0209` — Liberação de Cargas
- `FPLC0211` — Cadastro de Orientações de Entrega
- `FPLC0248` — Vinculação de Box de Expedição p/ Carga
- `FPLC0402` — Reserva de Estoque p/ Carga
- `FPLC0250` — Monitor de Expedição
- `FPLC0251` — Monitor de Separação
- `FPLC0253` — Painel de Gerenciamento Logístico por Rota
- Consultas: `CPLC0400`, `CPLC0402`, `FPLC0401`
- Relatórios: `FPLC0303`, `FPLC0306`–`FPLC0321`

### 14. Faturamento — ✅ NF-e/NFS-e saída existem; falta faturar por carga
> Cancelamento/exclusão de nota **já existem** — não refazer.
- `FFAT0220` — Emissão de Notas Fiscais por Carga *(principal novo)*
- `FFAT0221` — Emissão de Notas Fiscais de Saída *(base já existe)*
- `FFAT0258` — Geração de NF a partir de Cupom Fiscal *(contexto)*
- `FNFC0200 CFE` — Cupom Fiscal Eletrônico *(contexto)*

## Fora da lista de construção (existentes / transversais)

- **Cliente** (`FCLI0200`) — já rico; apenas melhorias (crédito/serasa, curva ABC,
  múltiplos endereços, vínculo a representante).
- **Administrador de Pagamentos** — módulo Financeiro já existe (recebíveis, CNAB,
  adiantamento, fluxo de caixa); melhorias: régua de cobrança/dunning e comissão a
  pagar do representante.

## Ordem de construção sugerida

Base primeiro (tudo depende de preço e política), depois o que gera receita e o
que o cliente marcou como crítico (representante):

1. Precificação
2. Política Comercial
3. Pedido de Venda (validar/completar)
4. Orçamento
5. Representantes
6. Meta de Vendas
7. Assistência Técnica
8. Atendimento ao Consumidor (SAC)
9. Central de Vendas
10. Vendas Recorrentes
11. Expedição (carga/rota/monitores)
12. Faturamento (por carga)
13. Promessa de Entrega → CTP (transversal)
14. Melhorias em Cliente e Administrador de Pagamentos

Cada módulo é implementado isoladamente com backend + testes (unitários da lógica
pura + script HTTP) + documentação + atualização do `SESSION_SUMMARY.md`, no mesmo
padrão usado no fechamento de Suprimentos.
