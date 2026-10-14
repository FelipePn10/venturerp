# Auditoria das tabelas sem coluna de empresa

Feita em 13/10/2026 sobre o esquema na migração 000399. Serve de base para a
próxima auditoria: o método abaixo é repetível.

## Resultado

- 450 tabelas; **158 sem** `enterprise_id`/`enterprise_code`.
- **116 filhas**: chegam a uma tabela com empresa por FK (etapa → roteiro,
  parcela → condição de pagamento…). O isolamento depende de toda consulta
  filtrar pelo pai.
- **42 sem caminho** de FK até uma tabela com empresa — classificadas abaixo.
- 599 consultas tocam essas tabelas; 244 não citam empresa; 181 em tabelas não
  globais; **96** sem proteção visível no dono nem no chamador. Todas foram lidas.

### Corrigido

| Onde | Falha |
|---|---|
| Roteiro (`route_operations`, `route_operation_resources`, `route_operation_network`) | Outra empresa alterava/inativava etapa, mexia em recurso alternativo e precedência pelo id; a própria empresa apontava operação/centro alheio |
| `route_operation_tools` | Já conferia a posse no repositório; o filtro foi para o SQL também |
| `supplier_due_dates` | Condição de pagamento de outra empresa era aceita |
| `capacity_requirements` (carga por centro) | Lia todas as empresas e cenários (consulta sem rota hoje) |
| Qualidade, quadro mensal e sequenciamento do APS | Corrigidos na rodada anterior |

### Protegido por outro caminho (falso positivo da varredura)

O pai é conferido antes, no caso de uso ou no repositório: parcelas da condição
de pagamento, idiomas/atributos da classificação fiscal, classificações de item
(máscara), filhos do fornecedor (contatos), CRP (`assertPlanTenant`), desenhos
(`*BelongsToEnterprise`), máscara de lote, manutenção (`GetPlanByID`,
`requireMachine`), séries e ficha de ferramenta, configurador (características
filtradas), custos (`ConferirItemDaEmpresa`), etapas da OF (roteiro tem de ser do
item da ordem), SAC (`consumer_service_consumer_enterprises`), extrato
(`contaDaEmpresa`), nota especial de ajuste, inspeção de recebimento, consulta de
pedido de compra, reajustes de preço de compra, assistência técnica.

`items.code` é único no banco: filho ligado por `item_code` pertence a uma só
empresa, e basta conferir o item na entrada.

### Sem caminho até a empresa (42)

- **Catálogos legais/globais** (corretos sem empresa): `cfops`, `legal_devices`,
  `ncm_tax_table`, `ibpt_rates`, `icms_internal`, `icms_interstate`,
  `icms_adjustment_codes`, `icms_apuracao_adjustment_codes`,
  `dapi_transfer_reasons`, `accounting_reference_accounts`, `countries`, `ufs`,
  `complement_a/b`, `formas_pagamento`.
- **Infraestrutura**: `users`, `enterprise`, `schema_migrations`,
  `http_idempotency_records` (chave inclui empresa e usuário),
  `notification_*`, sequências (`shipment_sequences`,
  `shipment_load_sequences`, `delivery_tank_reservation_sequences`),
  `procurement_immutable_audit` (só gravação).
- **Pelo item** (`item_code`, sem FK): `item_masks`, `item_unit_conversions`,
  `item_calendar_promises`, `item_structure_history`, `cost_rollup_log` — filtram
  pela empresa do item.
- **Coluna com outro nome**: `icms_st_restitutions.empresa_id` (filtrada).
- **Legado sem rota**: `products`, `components`, `component_masks`,
  `mask_composition`, `material_consumption`, `stock`, `mrp_parameters`,
  `perfis_usuario`, `permissoes`, `usuarios_perfis`.

### Decisões e pendências

- `cfops` e `legal_devices` continuam catálogo único do banco; incluir/alterar exige
  ADMIN (caso de uso e rota).
- Plano de corte ficou fora desta auditoria (inclusive `cutting_plan_parts`,
  `cutting_stock_pieces` e `cutting_plans`, que têm escrita sem filtro de empresa no
  SQL).

## Tabelas COM coluna de empresa

136 consultas não citam a empresa (88 leituras, 33 alterações, 6 exclusões, 9
inclusões). Todas protegidas pelo chamador (pai conferido, `WHERE` montado por quem
chama, ou operação interna do worker por id gerado pelo sistema), exceto o plano de
corte (fora do escopo) e consultas sem uso (`bom_headers`, `item_preferred_suppliers`,
`purchase_price_tables`, `drawings` sem `ForEnterprise`, etapas da OF antigas).

## Referência cruzada (registro apontando para cadastro de outra empresa)

`scripts/audit-referencias-entre-empresas.sh` confere, nos dados, as 293 FKs entre
tabelas com empresa e as colunas `item_code` sem FK (somente leitura, `\gexec`).
Produção (`venturerp`, `venturerp_usimac`, uma empresa cada) em 13/10/2026: nenhuma.

Não há trava genérica no banco porque há cruzamento **por desenho**: o MRP entre
fábricas planeja na empresa A o item do pedido INTER_FACTORY da empresa B, e as
tabelas `*_enterprises` ligam cadastros a várias empresas. No código, 177 gravações
preenchem FK para cadastro com empresa sem conferir no SQL; a maioria resolve o
código de negócio na empresa da sessão antes (item, cliente, fornecedor), e os
catálogos comerciais têm gatilho próprio (migração 000395). Uma trava completa
exige decidir, vínculo a vínculo, o que pode cruzar.

## Método

1. Banco migrado do zero; tabelas sem coluna de empresa e o grafo de FKs
   (`information_schema`) dão filhas × sem caminho.
2. Toda consulta (`queries/*.sql` e SQL escrito à mão em Go) que toca essas
   tabelas sem citar empresa é ligada à função dona e aos chamadores.
3. O que não tem conferência de empresa em nenhum dos dois níveis é lido à mão.


## Conferência de empresa nas referências (migração 000402)

A FK garante que o registro apontado existe; o gatilho `zz_conferir_empresa_ref`
garante que ele é da **mesma empresa**. `instalar_conferencia_de_empresa()` lê o
catálogo e gera uma função por tabela com as consultas estáticas de cada vínculo
(FK de uma coluna para tabela com empresa, mais `item_code`). As duas convenções
(`enterprise_id`/`enterprise_code`) são comparadas pela tabela `enterprise`. A
violação sai como `23503`, que a API traduz em "um dos vínculos informados não existe
na empresa autenticada".

Não confere: empresa nula (cadastro global legado), pai inexistente (papel da FK) e
UPDATE que não mexe na referência nem na empresa. Exceções de propósito
(`conferencia_empresa_excecao`): `mrp_*`, `planning_*`, `planned_orders`,
`production_plan_inter_factories` (fluxo interfábricas) e `*_enterprises`.

**Migration que cria tabela ou FK nova termina com
`SELECT instalar_conferencia_de_empresa();`** — o teste
`TestConferenciaDeEmpresaCobreTodosOsVinculos` falha se esquecer.
Teste que precisa simular dado legado cruzado usa `testutil.DadoLegadoCruzado`.

### Colunas de código sem FK (migração 000403)

Coluna `X_code` sem FK que tem uma tabela `X(e)s` com `code` e empresa também é
conferida — `fiscal_exits.sales_order_code`, `sales_orders.customer_code`,
`purchase_orders.supplier_code` etc. (87 vínculos). Como o código pode repetir
entre empresas (cada uma tem a sua cópia do cliente), a regra é outra: é violação
quando o código **não existe na empresa do registro e existe em outra**. Código
inexistente segue livre; cadastro global (empresa nula) vale para todas.
Exceções por coluna em `conferencia_empresa_excecao_coluna`: plano de corte,
`sales_order_demands` (planejamento interfábricas) e `warehouse_code` (o nome
coincide com a tabela `warehouse` sem ser ela).
