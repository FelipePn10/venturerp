# Decisões enterprise — almoxarifado, estoque e expedição

## Referências consultadas

- SAP EWM — Physical Inventory: https://help.sap.com/docs/SAP_S4HANA_ON-PREMISE/9832125c23154a179bfa1784cdc9577a/4adf9f15c9534e7de10000000a42189c.html
- Oracle Warehouse Management — Cycle Count: https://docs.oracle.com/en/cloud/saas/warehouse-management/
- Dynamics 365 Supply Chain — Cycle counting: https://learn.microsoft.com/dynamics365/supply-chain/warehousing/cycle-counting
- TOTVS WMS — documentação de estoque e inventário: https://tdn.totvs.com/display/public/PROT/WMS
- Odoo Inventory — Cycle counts: https://www.odoo.com/documentation/19.0/applications/inventory_and_mrp/inventory/warehouses_storage/inventory_management/count_products.html
- Odoo Inventory — Lots and serial numbers: https://www.odoo.com/documentation/19.0/applications/inventory_and_mrp/inventory/product_management/product_tracking.html

## Decisões

1. O código público do almoxarifado é textual e estável. O `id` numérico continua sendo a chave interna usada em movimentos, saldos, endereços e caixas.
2. `location` representa a finalidade operacional (`INTERNO`, `INSPECAO`, `EXPEDICAO` etc.); `type` representa a estrutura (`NORMAL` ou `LINHA_DE_PRODUCAO`). A inversão histórica foi corrigida por migration.
3. A política/peridiocidade de contagem cíclica pertence ao cadastro do item. VEST0500 cria e executa ocorrências; não mantém uma segunda política concorrente.
4. `warehouse_address_id` identifica endereço físico WMS. O catálogo canônico é `GET /api/warehouse-addresses`, isolado por empresa e opcionalmente filtrado por `warehouse_id`.
5. Inventário físico segue estados e operações explícitas: abrir, contar, ajustar e fechar. Consultar linhas valida primeiro a existência do inventário no tenant.
6. Ajustes e movimentos conservam documento de origem, ator e empresa. Auditoria antes/depois é imutável.
7. Máscaras de lote são resolvidas por contexto e tenant, do vínculo mais específico ao geral. O código comercial do item é textual na API e resolvido para a chave interna.
8. Estados técnicos de carga/romaneio permanecem estáveis para integrações; a API fornece `status_label` em PT-BR para exibição.
9. Cargas, romaneios, caixas e transportadoras usam listagens canônicas. Uma caixa só pode apontar para almoxarifado da mesma empresa.

## Contratos canônicos

- Almoxarifados: `/api/warehouse`.
- Endereços WMS: `/api/warehouse-addresses`.
- Inventários: `/api/inventory`; `/api/stock/inventories` permanece temporariamente compatível.
- Saldos por almoxarifado: `/api/stock/balances/warehouse/{warehouseId}` com `item_code`, `lot`/`mask`, `page` e `per_page`.
- Contagens cíclicas: `/api/stock/cycle-counts`.
- Máscaras e geração: `/api/lot-masks`.
- Romaneios, cargas e caixas: `/api/shipments`, `/api/shipments/loads` e `/api/shipments/dispatch-boxes`.
