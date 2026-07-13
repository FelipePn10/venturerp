# Serviços de terceiros — referência técnica

Este módulo controla preço contratado, seleção do fornecedor, custeio padrão e
acompanhamento de operações `EXTERNA`/`TERCEIROS`, desde a firmação da OF até o
retorno do serviço. Todos os registros operacionais são isolados por
`enterprise_id`; quantidades e valores são `NUMERIC`/`decimal`, sem cálculo em
ponto flutuante no novo domínio.

## Fluxo integrado

1. O planejador mantém preços por item, máscara, fornecedor, operação e vigência.
2. A resolução considera data, máscara, fornecedor preferencial e regras do item
   configurado. Preço zero registra um fornecedor possível sem compor custo.
3. O custo padrão usa preço vigente + frete; o custo real desconta os tributos
   recuperáveis. A conversão explícita do contrato prevalece sobre a conversão
   configurada do item, que prevalece sobre a conversão global.
4. O MRP cria uma sugestão `SERVICO` enriquecida por operação externa. Ao criar
   uma OF manual ou firmar a sugestão de produção, o ERP cria atomicamente uma
   OS firme por operação e vincula requisição e pedido de compra.
5. A geração da OC atualiza a ordem de serviço para `RELEASED_WITH_PO` na mesma
   transação. Remessas, retornos, recebimentos e ajustes formam a rastreabilidade;
   o recebimento total conclui a ordem.

## API

Todas as rotas começam por `/api/third-party-services` e exigem JWT. `USER` pode
consultar; manutenção, liberação e movimentos exigem `ADMIN`.

| Método | Rota | Uso |
|---|---|---|
| GET/POST | `/prices` | listar/criar preços |
| GET | `/prices/resolve` | resolver o preço vigente |
| GET | `/cost?mode=STANDARD\|REAL` | memória de cálculo unitária |
| POST | `/prices/readjust` | reajustar uma seleção de forma atômica |
| POST | `/prices/copy-move` | copiar ou mover uma seleção atomicamente |
| GET/PUT/DELETE | `/prices/{id}` | consultar, alterar ou inativar |
| GET | `/prices/{id}/history` | trilha de alterações e justificativas |
| POST | `/production-orders/{id}/orders` | regenerar idempotentemente as OS da OF |
| GET | `/orders` | consultar OS por plano, item, OF, operação, fornecedor, OC, período, situação e posição |
| GET | `/orders/report?format=csv\|xlsx\|pdf\|docx` | relatório operacional imprimível |
| GET | `/orders/{id}` | detalhe da OS |
| PATCH | `/orders/{id}/status` | transição controlada de situação |
| POST/GET | `/orders/{id}/movements` | registrar/listar remessa, retorno, recebimento ou ajuste |
| GET | `/orders/{id}/history` | histórico de criação, situação e movimentos |
| GET/POST | `/global-conversions` | listar/manter conversões globais |
| DELETE | `/global-conversions/{id}` | inativar conversão global |

Filtros de preços: `item_from`, `item_to`, `item_search`, `supplier_from`,
`supplier_to`, `supplier_search`, `operation_id`, `mask`,
`classification_mask_code`, `classification_codes`, `reference_date`, `preferred`, `price_type`
(`WITH_PRICE`, `WITHOUT_PRICE`, `BOTH`), `order_by`, `limit` e `offset`. Na data
de referência, a listagem inclui vigências futuras e, se necessário, a vigência
imediatamente anterior de cada chave; a resolução operacional escolhe a última
vigência não futura.

Exemplo de manutenção:

```json
{
  "item_code": 10001,
  "mask": "AZUL-220V",
  "supplier_code": 200,
  "operation_id": 37,
  "uom": "PC",
  "reference_date": "2026-07-13T00:00:00Z",
  "preferred": true,
  "unit_price": "125.500000",
  "conversion_factor": "2.00000000",
  "freight_type": "PERCENT",
  "freight_value": "4.500000",
  "tax_percent": "5.000000",
  "formula": "BASE + ESPESSURA * 2.5",
  "reason": "Contrato anual 2026",
  "rules": [{"characteristic": "COR", "answer": "AZUL"}]
}
```

A fórmula aceita números decimais, variáveis, parênteses e `+ - * /`. As
variáveis são carregadas automaticamente dos atributos PDM, peso e dimensões do
item; o parâmetro `attributes` de `/prices/resolve` pode complementar ou
sobrescrever valores. Expressões inválidas, divisão por zero, resultado negativo
e duas regras válidas com a mesma prioridade são rejeitados explicitamente.

## Conversão e arredondamento

O fator informado no preço é a primeira opção. Sem ele, a resolução procura a
conversão direta ou inversa do item e da máscara; por último, procura a conversão
global da empresa. O cadastro por item aceita `rounding_percent`,
`tolerance_value` e `tolerance_type` (`VALUE` ou `PERCENT`). Itens que aceitam
fração não são arredondados. Para os demais, uma quantidade só é convertida para
inteiro dentro da faixa configurada; fora dela a movimentação é rejeitada.

## Consulta e movimentos das OS

Com `plan_code`, `/orders` combina OS firmes ligadas à OF com sugestões
planejadas `SERVICO`. Uma linha planejada possui `planned_suggestion_code` e
`production_order_id=0`; movimentos só são permitidos após existir a OF/OS
firme. A consulta aceita listas separadas por vírgula de OF, OS, operação,
fornecedor e pedido de compra, períodos de emissão/entrega, item/fornecedor por
código ou descrição, classificação, Kanban, situação, posição e ordenação.

Todo movimento exige `idempotency_key`. Repetir exatamente a mesma requisição
retorna o movimento original; reutilizar a chave com outro conteúdo é erro.
`REMITTANCE` não pode exceder a OS, `RETURN` não pode exceder o remetido e apenas
`RECEIPT` aumenta a quantidade atendida. Ordens não liberadas ou terminais não
aceitam novos movimentos.

## Persistência e atomicidade

A migration `000233_third_party_services` cria preços, regras, histórico, ordens
e movimentos. As migrations `000234`–`000236` completam conversões, idempotência,
histórico de OS, dados da sugestão do MRP e remessa genérica do roteiro. Os
respectivos `down` removem ou restauram apenas esses contratos. Criação/alteração de preço e
suas regras/histórico, reajuste, cópia/movimentação e recebimento são transações.
Uma falha em qualquer item de um lote desfaz todo o lote. Exclusão de preço é
lógica para preservar histórico. A chave OF + operação de roteiro torna a criação
das OS idempotente.

## Situações

Transições válidas: `PLANNED → FIRM`; `FIRM → RELEASED_WITH_PO` ou
`RELEASED_WITHOUT_PO`; liberada → `COMPLETED`. Situações não terminais aceitam
`CANCELLED`. Estados concluído/cancelado são terminais. `RELEASED_WITH_PO` exige
o código da OC e um movimento não pode ultrapassar a quantidade pendente.

## Validação

```bash
TEST_DATABASE_URL='postgres://...' scripts/test-third-party-services.sh
k6 run -e TOKEN='...' scripts/loadtest/k6/third-party-services.js
```

Para um ensaio reproduzível em volume industrial, crie pela API o usuário local
`loadtest.thirdparty@panossoerp.test`, associe-o à empresa de testes e execute
`scripts/loadtest/seed-third-party-services.sql`. O seed é idempotente e reserva
as faixas `930000000`–`989999999`; ele cria 1.000 itens, 100 fornecedores,
50.000 preços, 20.000 OF/OS e 1.000 conversões exclusivamente no banco apontado
para testes. Nunca execute esse seed em produção.

Baseline local de 13/07/2026, API compilada da branch
`feature/third-party-services`, PostgreSQL 16 e k6 em Docker: 25 iterações/s
sustentadas, com cinco consultas autenticadas por iteração. Foram concluídas
3.637 iterações e 18.185 requisições, sem interrupções ou erros HTTP; 100% dos
checks passaram, com p95 de 14,96 ms, p99 de 15,70 ms e máximo de 126,20 ms.
Os thresholds exigidos permanecem erro abaixo de 1%, p95 abaixo de 750 ms,
p99 abaixo de 1.500 ms e checks acima de 99%.

O teste integrado cobre vigência, fórmula e atributos automáticos, regras,
histórico, reajuste e rollback em lote, tenant, OS planejada, geração concorrente
e idempotente, criação pela OF manual, vínculo de requisição, logística,
conversões e detalhes do roteiro. Os testes HTTP cobrem CRUD, movimentos,
histórico, conversões, CSV/PDF e autorização `USER` versus `ADMIN`.

## Rastreabilidade da tarefa

| Requisito funcional | Implementação/verificação |
|---|---|
| Item/máscara, fornecedor, operação externa, vigência, preço zero e preferencial | `Price`, validação transacional e `/prices` |
| Filtro por código/descrição, intervalos, classificação, data-base, tipo e ordenação | `PriceFilter`/`ListPrices`; filtros parametrizados e paginados |
| Regra da data-base: futuras + anterior mais próxima; resolução sem preço futuro | consultas de listagem/resolução e teste integrado |
| UM e prioridade fator do preço → item/máscara → global, direta/inversa | `ResolveConversionFactor`, migrations 234 e endpoints globais |
| Arredondamento, tolerância e item fracionável | `ConvertQuantityConfigured` e testes de conversão |
| Frete somente no padrão e impostos recuperáveis somente no real | `CostPerUnit`, `/cost` e testes unitários |
| Fórmula com PDM/peso/dimensões e variáveis informadas | `formulaAttributes`, `EvaluateFormula` e integração |
| Alteração motivada, histórico, reajuste e cópia/movimentação atômicos | `third_party_service_price_history` e transações com rollback testado |
| Regras configuradas e conflito de mesma prioridade | `third_party_service_price_rules` e detecção de ambiguidade |
| Operação externa preserva origem quando usada; fornecedor/custo/prazo/remessa persistem | routing use cases/repository e teste de round-trip |
| Sugestão OS no MRP com plano/operação/fornecedor/remessa | migration 235, `calcNetReqFast` e consulta por `plan_code` |
| OS na inclusão manual da OF e na firmação | `createThirdPartyOrdersTx`, hook de firmação e testes integrados |
| Consulta por item/classificação/OF/OS/operação/período/situação/Kanban/posição | `OrderFilter`, `/orders` e `/orders/{id}` |
| Vínculo OF → requisição → OC | firmação e `LinkServicePurchaseOrder`, tenant-aware |
| Remessa, retorno, recebimento, saldo, idempotência e histórico | `/movements`, `/history` e invariantes transacionais |
| Relatório com histórico e exportação imprimível | `/orders/report` em JSON/CSV/XLSX/PDF/DOCX |
| Segurança, tenant, concorrência e carga | middleware de papel, filtros tenant, testes concorrentes e k6 |
