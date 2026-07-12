# Planejamento de materiais — matriz de requisitos

Fonte: `.ai/tasks/planejamento-de-materiais-manufatura.md`.

| Processo/requisito | Estado inicial | Evidência/impacto | Estado desta etapa |
|---|---|---|---|
| Planos: identificação, demandas, agrupamento, tipos, filtros, pedido e parâmetros | parcial | CRUD existia sem validação uniforme | concluído: domínio validado; filtros por classificação hierárquica; item de pedido exclusivo; agrupamento apenas por item, máscara e mesma data |
| Empresas inter-fábrica e liberação automática | ausente | sem agregado específico no plano | concluído: configuração tenant-aware, demanda DIF somente de empresa autorizada, rastreio OIF/OCI e conversão/liberação idempotente quando `auto_release=true` |
| Cálculo exclusivo, plano, ordem inicial e LLC | parcial | cálculo e LLC existiam; parâmetros de execução eram ignorados | concluído: exclusão global, numeração inicial preservada até a OF e gravação opcional do LLC |
| MRP/MPS, mínimos/máximos, ponto de reposição e Kanban | concluído | modos no `MRPServiceImpl` | concluído |
| Demandas de venda, previsão, independente, segurança e dependente | parcial | fontes e explosão existem | concluído: saldo aberto de pedidos, previsões, independentes, segurança, explosão dependente, assistência e demanda inter-fábrica autorizada |
| Estoque, suprimento firme, lotes de planejamento, lead time e calendário | concluído | cálculo líquido, supply port e calendário | concluído |
| Assistência técnica/OAT e almoxarifado | parcial | classificação anterior usava associação do item e não propagava almoxarifado | concluído no planejamento: exige parâmetros 17 e 45, usa divisão do pedido, segrega almoxarifados e preserva o destino até a OF; entrega física permanece na etapa de entrega |
| Sugestões, perfil, exceções, prioridade e máquina | concluído | persistência, consultas e pós-processamento | concluído |
| Entrega, movimentos EPP/EPE/REP, encerramento e OCS | parcial | entrega anterior não era atômica e não alimentava OCS | concluído: transação única, idempotência, divisão EPP/EPE, REP/co-produtos, rollback testado e vínculo OF–requisição–OCS efetivo |
| Liberação/firmação/replanejamento de OF e Kanban | parcial | firma individual existia sem estados e proteções completos | concluído: ações individuais/em lote, liberada versus firme, replanejamento sem movimentos, datas firmes imutáveis e parâmetro 25 para Kanban |
| Consultas operacionais e relatórios | parcial | os cinco conjuntos de dados existiam sem todas as variações | concluído: filtros, layouts, ordenações, quebras, seis períodos, origens analíticas persistidas, pedidos, desenhos tenant-aware, explosão por item/OF/carga e reposição por estruturas liberadas/bloqueadas |
| Desenhos, prioridades e manutenção de OF | parcial | manutenção/cancelamento, quantidade/datas, prioridades, demandas e substituições existem | concluído: desenhos na fase 2 e manutenção integral de OF na fase 4, com validação transversal na fase 5 |
| Parâmetros 66 (retrabalho) e 45 (terceiros) | parcial | regras operacionais não estavam ligadas à liberação/manutenção | concluído: quantidade de retrabalho; apontamento/baixa; roteiro misto; remessa de terceiros; testes integrados |
| Lotes/endereço/WMS em requisição e devolução | parcial | seleção individual/em lote, FIFO, saldo e intermediário WMS existiam | concluído na fase 4 e revalidado na fase 5: parâmetros 44/53, modos A/I/E, endereço, WMS efetivo, confirmação parcial e distribuição entre OFs |
| Isolamento por empresa | ausente | tabelas MRP legadas não tinham `enterprise_id` | concluído: associação autenticada, JWT, migration, filtros e escritas por tenant |

## Dependências e ordem

1. Exclusão mútua no banco e conflito HTTP.
2. Validação de entrada antes do cálculo.
3. Estratégia de `enterprise_id` e backfill concluída com quarentena de registros ambíguos.
4. Filtros/ordem inicial e assistência técnica com regras confirmadas.
5. Processos operacionais em tasks próprias dos respectivos domínios.

## Premissas conservadoras

- A vedação a cálculos simultâneos é global e vale entre réplicas da API.
- Logs órfãos não são encerrados automaticamente, evitando liberar concorrência por falso timeout.
- OF, estoque, lote, WMS e desenhos não são reimplementados dentro do motor MRP.
- A associação entre item e classificação é tenant-aware; selecionar um nó inclui seus descendentes e filtra somente demandas raiz, preservando os filhos da estrutura.
- Quando `order_item_code` é informado, previsões, demais pedidos, demandas independentes e estoque de segurança são ignorados.

## Matriz detalhada de aceite

| Grupo da task | Implementado e localizado | Resultado transversal |
|---|---|---|
| Relatório do perfil | plano, item, planejador, período, tipo, posição, classificação, ordem 1/2, quebra, analítico/sintético, desenhos, pedidos e mensagens | concluído na fase 1; `mrp_profile_details` preserva cada origem antes do agrupamento |
| Disponibilidades | pedido(s) ou item/quantidade, estrutura, classificação e layouts Ambos/Necessidades/Itens Pedido | concluído na fase 1, com teste de explosão e seleção de layout |
| Necessidades agrupadas | plano, planejador, item, classificação, ordem/quebra e seis períodos independentes | concluído na fase 1; `period_values` mantém a ordem dos períodos |
| Explosão | multinível, vigência, classificação, almoxarifado, Simples/Custo/Saldo/Saldo-Dem, filhos imediatos e raízes por item/OF/carga | concluído na fase 1 |
| Ponto de reposição | estoque total/disponível, segurança, máximo, consumo, OF, compras, Kanban/reposição e estruturas de pedidos liberados/bloqueados | concluído na fase 1, com teste das duas posições de pedido |
| Código/revisão de desenho | cadastro, revisões, aprovação, motivo, distribuição, características, código composto, item/configuração, parâmetro 8, replicação condicional e isolamento por empresa | concluído na fase 2 |
| Consulta de pedido de compra | concluído na fase 3: intervalos e filtros combinados, posição calculada Atendido/Pendente/Cancelado, todos os itens, conversão por data, Kanban, comprador, tipos OCL/OSL/ORM/ORD e cliente, processos de importação, totais/frete, impostos OCL, anexos/download e isolamento tenant | nenhum requisito residual confirmado para esta consulta |
| Manutenção de OF | concluído na fase 4: criação manual/demandas imediatas, retrabalho, consulta com tipos, bloqueios de atividade/Kanban/OFC, parâmetros 10/14, fração, OCS/WMS, demandas/devoluções, substituição, lote temporário e isolamento tenant | nenhum requisito residual confirmado nesta fase |
| Destinação de refugos | concluído na fase 4: item/demanda, limites, devolução e sucata, períodos/intervalo/valorização, UM/conversão, grupo secundário, lote/endereço, movimentos atômicos e reversão sem estoque negativo | nenhum requisito residual confirmado nesta fase |
| Seleção de lotes | concluído na fase 4: unitária/em lote, FIFO, saldo, WMS intermediário, parâmetros 44 A/I/E e 53, endereço obrigatório, distribuição por OF e confirmação parcial explícita | nenhum requisito residual confirmado nesta fase |

Todos os itens desta matriz possuem evidência em contrato, regra de aplicação,
persistência, teste automatizado e documentação. Novas regressões devem reabrir o
item correspondente; a conclusão não elimina a necessidade de monitoramento em produção.

## Ordem restante de implementação

1. ~~Completar contratos e filtros dos cinco relatórios/consultas de MRP.~~ Concluído.
2. ~~Completar manutenção e replicação de desenhos, usando a fundação tenant criada na fase 1.~~ Concluído.
3. ~~Completar a consulta de pedidos de compra.~~ Concluído.
4. ~~Completar as regras residuais de OF, refugos, parâmetros e lotes.~~ Concluído.
5. ~~Executar a validação transversal e somente então promover cada linha a concluída.~~ Concluído.

## Auditoria de aceite enterprise

Aceite transversal concluído em 12/07/2026. Os bloqueadores técnicos corrigidos incluem atomicidade e rollback da entrega,
EPP/EPE, vínculo OCS, isolamento tenant de OF e estoque, parâmetros 45/66 e
precisão decimal no fluxo novo. Não restam linhas abertas na matriz detalhada.

Última evidência: todos os portões passaram em 12/07/2026. Resultados e limites
de cobertura estão em [`mrp-aceite-enterprise.md`](mrp-aceite-enterprise.md).
O detalhamento de cada comportamento aceito está em
[`mrp-rastreabilidade-requisito-a-requisito.md`](mrp-rastreabilidade-requisito-a-requisito.md).
