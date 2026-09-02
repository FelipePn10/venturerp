# Decisões enterprise — engenharia e manufatura

## Escopo e referências

Pesquisa realizada em 31/08/2026 nas documentações oficiais de [SAP S/4HANA Cloud](https://help.sap.com/doc/7c9e0bbbd1664c2581b2038a1c7ae4b3/latest), [Oracle Fusion Manufacturing](https://docs.oracle.com/en/cloud/saas/supply-chain-and-manufacturing/26a/faumf/overview-of-work-definitions.html), [Dynamics 365 Supply Chain](https://learn.microsoft.com/en-us/dynamics365/supply-chain/master-planning/planning-optimization/production-planning) e [Odoo Manufacturing](https://www.odoo.com/documentation/17.0/applications/inventory_and_mrp/manufacturing/advanced_configuration/using_work_centers.html). O conteúdo oficial público da TOTVS localizado não descreveu esses contratos com precisão suficiente; por isso não foi usado como fundamento normativo.

## Decisões aplicadas

- O código público do item é `business_code` textual. A chave numérica continua interna e números JSON são aceitos somente durante a janela de compatibilidade.
- A estrutura é multi-nível e navegável; cada submontagem fabricada pode ter estrutura e roteiro próprios. Operações vêm de um catálogo único e o roteiro associa sequência, centro, recurso e tempos ao item.
- Uma OF gerada/fixada pelo MRP deve materializar uma fotografia da estrutura e do roteiro efetivos. Alterações posteriores no cadastro não reescrevem uma OF liberada.
- Tempo efetivo por ciclo = tempo padrão × fator do recurso ÷ eficiência. Tempo total = setup + teto(quantidade/quantidade base) × tempo efetivo. Capacidade requerida e disponível são devolvidas separadamente para explicar gargalos.
- O MRP cria ordens planejadas. A firmagem, manual ou automática dentro do horizonte, cria a OF e agenda recursos; APS/CRP usa roteiro, precedências, calendários e capacidade finita. Criação manual é exceção operacional e não pode iniciar sem operações/materialização do roteiro.
- Não será copiada complexidade sem benefício imediato: alternativas versionadas, yield e utilização ficam evolutivas; o contrato atual expõe fator de recurso `1` até existir cadastro governado por centro/recurso.

## Fluxo canônico operação → roteiro → OF

1. A biblioteca em `/api/routing/operations` mantém operações ativas do tenant.
2. `/api/routing/routes` vincula essas operações a um item/máscara e calcula `lead_time_hours` e `critical_path`.
3. O MRP explode a BOM multi-nível, cria ordem planejada e preserva a origem da sugestão.
4. Ao firmar, o backend gera a OF com materiais, operações, centros e tempos. CRP/APS reserva capacidade e define datas.
5. Apontamento e conclusão consomem a fotografia da OF, não o cadastro mutável.

## Semântica do cadastro de item

- **UM de estoque base:** unidade física canônica dos saldos e movimentos.
- **UM de suprimentos/compras:** unidade usada em pedidos e recebimentos; converte para a UM base pela tabela de conversões.
- **UM contábil de venda/compra:** unidade documental/fiscal da saída ou entrada; não altera silenciosamente saldo físico.
- **Classificação de planejamento:** políticas MRP, criticidade e lote.
- **Classificação comercial:** segmentação de venda, preço e canais (`mobile_enabled` significa “Habilita no aplicativo”).
- **Classificação contábil:** contabilização e agrupamento fiscal/gerencial.
- **Classificação de suprimentos:** compra, inspeção, fornecedor e abastecimento.

`volume_conversion_factor` representa composição de volume/embalagem comercial. Ele não substitui a conversão dimensional de UM de VSUP0110; os dois campos não devem se sobrescrever. Conversão de UM transforma quantidades entre unidades, enquanto o fator comercial dimensiona volumes de venda/expedição.

## Contrato de criação de OF

A criação manual informa no mínimo item, quantidade, datas e máscara quando aplicável. Antes do início, o backend deve selecionar um roteiro efetivo e materializar operações e componentes; ausência de roteiro deve bloquear o início com erro de domínio. A geração pelo MRP também preenche origem da sugestão, necessidades multi-nível, roteiro, centros, tempos e dados de programação. Mutações de firmagem/reprogramação/movimento exigem chave de idempotência.
