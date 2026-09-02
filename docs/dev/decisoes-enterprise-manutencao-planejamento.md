# Decisões enterprise — manutenção, APS, planejamento, MRP e previsão

Pesquisa realizada em 01/09/2026 nas documentações oficiais de [SAP S/4HANA — integração de ordens de manutenção com PP/DS](https://help.sap.com/docs/SAP_S4HANA_ON-PREMISE/f899ce30af9044299d573ea30b533f1c/480dd1bbb35c4aa5e10000000a421937.html), [SAP S/4HANA — planejamento de capacidade em manutenção](https://help.sap.com/docs/SAP_S4HANA_ON-PREMISE/e72f747389b340229f7fa343975bfa57/fe4463b408504fe1a7120bcbc8093d98.html), [Oracle Fusion Cloud Maintenance](https://docs.oracle.com/en/cloud/saas/supply-chain-and-manufacturing/25b/faumm/using-maintenance.pdf), [Dynamics 365 Asset Management](https://learn.microsoft.com/en-us/dynamics365/supply-chain/asset-management/overview/asset-management-overview) e [Odoo Maintenance](https://www.odoo.com/documentation/18.0/applications/inventory_and_mrp/maintenance/maintenance_requests.html). A documentação pública da TOTVS localizada não detalha contratos HTTP ou isolamento multiempresa suficientes para fundamentar decisões técnicas desta rodada.

## Decisões aplicadas

1. Plano e ordem de manutenção são cadastros distintos. O plano define recorrência; a ordem representa execução programada/corretiva, com recurso, janela e estado próprios.
2. Ordens e paradas de manutenção ocupam capacidade. APS/CRP devem tratá-las como indisponibilidade do recurso, sem permitir que o sequenciador altere o documento de manutenção.
3. Referências de plano, ordem, item e máquina são validadas antes da persistência. Ausência retorna erro de domínio em PT-BR, nunca texto de FK ou `no rows in result set`.
4. Catálogos canônicos são a origem dos modais: planos e ordens de manutenção, paradas filtradas por máquina/período, parâmetros por número, calendário por item/máscara e cargas/romaneios.
5. Datas operacionais usam RFC3339 quando incluem hora e `AAAA-MM-DD` quando o contrato representa apenas um dia. Ano e mês inválidos retornam HTTP 422.
6. O MRP valida o plano da empresa autenticada antes de criar o log de execução. O pipeline MRP→CRP→APS aplica a mesma validação antes de produzir efeitos parciais.
7. O realizado de previsão é uma série mensal consultável por item/ano. A fonte é explícita: pedidos liberados (`ORDERS`), faturamento autorizado (`INVOICING`) ou ambas (`BOTH`). Toda agregação filtra a empresa do JWT.
8. O ator de mutações vem exclusivamente do JWT; campos legados como `created_by` no corpo são ignorados quando o fluxo autenticado está configurado.
9. Apontamento, execução do pipeline e criação/desativação de paradas exigem `Idempotency-Key`. A chave, a impressão SHA-256 do corpo e a resposta concluída ficam persistidas por empresa, usuário, método e rota; reuso com outro corpo retorna `409/IDEMPOTENCY_KEY_REUSED`.
10. As mutações operacionais desta rodada geram auditoria append-only com estado anterior e posterior. Tentativas de `UPDATE` ou `DELETE` no histórico são rejeitadas pelo banco; o log HTTP autenticado mantém o ator obtido do JWT.
11. Previsões, bloqueios e tabelas de apropriação possuem chave de empresa própria. Chaves naturais e seleção da tabela padrão são únicas dentro do tenant, não globalmente.

## Limites desta rodada

- A previsão estatística continua separada do realizado: nenhuma série real é sobrescrita por cálculo de forecast.
- A API não tenta reproduzir heurísticas proprietárias de otimização dos ERPs pesquisados; preserva o sequenciamento e a capacidade finita já existentes.
- O pipeline coordena três motores existentes. A validação de referências ocorre antes do primeiro estágio e a idempotência persistente torna a repetição segura; uma unidade de trabalho única entre MRP, CRP e APS exigiria contratos transacionais comuns e permanece fora desta alteração corretiva.
