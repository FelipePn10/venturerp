# Decisões enterprise para o ciclo comercial

Status: adotado incrementalmente pela tarefa `confiabilidade-e-evolucao-enterprise-ciclo-comercial`.

## Princípios adotados

- A regra comercial é resolvida por linha. A capa pode informar contexto ou preferência, mas não duplica uma tabela de preço que pode variar entre itens.
- Toda resolução retorna o resultado e a explicação: regra de origem, candidatos, prioridade, vigência e motivo de eventual override.
- Preço confirmado é um snapshot auditável. Mudanças posteriores na tabela não reescrevem silenciosamente uma negociação existente; um recálculo explícito gera evento e novo snapshot.
- Comissão é calculada separadamente da formação do preço. A política define a competência (`FATURAMENTO`, `RECEBIMENTO` ou `RATEIO`), e cancelamentos/devoluções geram estorno, não exclusão do histórico.
- Promessa e reprogramação trabalham com saldo por linha, mostram fatores disponíveis (estoque, reservas, compras, produção e capacidade) e só efetivam um lote idempotente e atômico.
- Recorrência, reajuste, workflow e RMA usam máquinas de estado com `allowed_actions`, motivo obrigatório nas exceções e eventos imutáveis com ator do JWT.
- O isolamento por empresa, a autorização, a validação e os cálculos ficam no servidor. Dados de empresa e identidade de usuário enviados pelo cliente não são fontes confiáveis.
- `item_code` público é sempre o código comercial textual. A chave numérica
  legada permanece interna; JSON numérico é aceito apenas durante a janela de
  compatibilidade e produz telemetria de depreciação.
- Atendimento e faturamento são fatos distintos. Atendimento registra ator,
  data e motivo; somente o evento fiscal autorizado coloca o pedido em `F`.
- Relatório sem razão social e documento da empresa é rejeitado com erro de
  domínio, em vez de gerar papel timbrado vazio.

## Referências e recorte aplicado

| Mercado | Prática observada | Decisão no VentureERP |
|---|---|---|
| SAP S/4HANA | aATP/BOP reavalia confirmações quando oferta ou demanda muda, com seleção e prioridade configuráveis. | Prévia explicável por linha e reprogramação em lote; não reproduzir toda a suíte de otimização enquanto os sinais de ATP/MRP/CRP existentes forem suficientes. |
| Oracle Fusion Cloud | Order Management preserva informação de preço e ajustes no documento e integra promessa, alteração de pedido e compensação. | Snapshot de preço, trilha antes/depois e eventos para integrações; evitar acoplamento direto entre módulos por meio de outbox. |
| Dynamics 365 | Acordos específicos prevalecem sobre preço-base; concorrência pode escolher menor preço ou maior prioridade; preço líquido pode bloquear descontos adicionais. | Precedência explícita, candidatos e razão da escolha; política de empilhamento e preço final protegido contra desconto duplicado. |
| TOTVS | Tabelas consideram vigência/faixa, admitem tabela por item e preservam o preço já negociado; comissão tem hierarquia e competência configurável. | Tabela por linha, quantidade/data/unidade como contexto, snapshot confirmado e política de comissão vigente sem somá-la novamente ao preço. |
| Odoo | Listas variam por cliente, grupo, quantidade, moeda e período; recorrências têm planos e renovação automática ou manual. | Resolver apenas dimensões úteis ao VentureERP, com calendário e idempotência por competência; não expor override irrestrito ao usuário comum. |

O recorte da segunda rodada reforçou práticas comuns: SAP determina preço e ATP
quando a linha recebe o produto; Oracle mantém crédito/comissão por linha e
encaminha o resultado para recebíveis/compensação; Dynamics distingue preço
sugerido do preço autoritativo do Supply Chain e preserva revisões; Odoo aplica
prioridade de listas por cliente, quantidade, moeda e vigência. No VentureERP,
isso se traduz em código comercial estável, preço autoritativo por linha,
passivo de comissão separado do preço e estados consultáveis de carteira.

Fontes oficiais consultadas:

- SAP Help — [Advanced ATP: Backorder Processing](https://help.sap.com/docs/SAP_S4HANA_ON-PREMISE/f132c385e0234fe68ae9ff35b2da178c/73a1a457ef816b10e10000000a441470.html)
- SAP Help — [Pricing and Conditions](https://help.sap.com/docs/SAP_S4HANA_ON-PREMISE/f340785101c548c9beeda9284efd18a0/5e49b753128eb44ce10000000a174cb4.html)
- SAP Help — [Manage Sales Orders](https://help.sap.com/docs/SAP_S4HANA_CLOUD/a376cd9ea00d476b96f18dea1247e6a5/e7f14402cf5846b4b3d0d677c15414b1.html)
- Oracle — [Using Order Management](https://docs.oracle.com/en/cloud/saas/supply-chain-and-manufacturing/26a/fauom/using-order-management.pdf)
- Oracle — [Sales credits for sales orders](https://docs.oracle.com/en/cloud/saas/supply-chain-and-manufacturing/26a/fauom/sales-credits-for-sales-orders.html)
- Microsoft — [Sales trade agreement prices](https://learn.microsoft.com/en-us/dynamics365/supply-chain/unified-pricing-management/upm-sales-trade-agreement-prices)
- Microsoft — [Sales agreements and version history](https://learn.microsoft.com/en-us/dynamics365/supply-chain/sales-marketing/sales-agreements)
- TOTVS TDN — [Tabela de preço por item do pedido](https://tdn.totvs.com/pages/viewpage.action?pageId=783577764)
- TOTVS TDN — [Parâmetros de pedidos, preços e comissão](https://tdn.totvs.com/pages/viewpage.action?pageId=240295063)
- Odoo 19 — [Pricelists](https://www.odoo.com/documentation/19.0/applications/sales/sales/products_prices/prices/pricing.html)
- Odoo 19 — [Subscription renewals](https://www.odoo.com/documentation/19.0/applications/sales/subscriptions/renewals.html)
- Odoo 19 — [Commissions](https://www.odoo.com/documentation/19.0/applications/hr/payroll/commissions.html)

## Complexidade deliberadamente não copiada

- Otimizadores avançados e heurísticas proprietárias de alocação não serão introduzidos antes de existirem capacidade, calendários e qualidade de dados suficientes.
- Não haverá um motor de regras genérico sem tipos. Precedência e empilhamento serão contratos explícitos, versionados e testáveis.
- Não haverá tabela de preço obrigatória na capa quando as linhas divergirem, nem recálculo implícito de negociações confirmadas.
- Histórico, outbox e lançamentos não serão apagados para “corrigir” estado; correções ocorrerão por eventos compensatórios.

## Critério de evolução

Cada incremento precisa provar resposta correta, estado persistido, evento/auditoria, ator do JWT e isolamento cruzado entre duas empresas. Um endpoint `2xx` isoladamente não caracteriza aceite.
