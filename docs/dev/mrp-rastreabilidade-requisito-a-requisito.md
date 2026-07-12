# Rastreabilidade requisito a requisito — planejamento e manufatura

Fonte: `.ai/tasks/planejamento-de-materiais-manufatura.md`. “Aceito” exige contrato,
persistência quando aplicável e evidência automatizada nos portões indicados.

| ID | Requisito verificável | Evidência/teste principal | Estado |
|---|---|---|---|
| PLN-01 | Múltiplos planos por empresa | production plan/integrado | Aceito |
| PLN-02 | Cálculo exclusivo | migration 205/concorrência | Aceito |
| PLN-03 | Independentes: não, por data ou todas | demand sources | Aceito |
| PLN-04 | Agrupar item/máscara/mesma data | serviço MRP | Aceito |
| PLN-05 | MRP/MPS/mín-máx/reposição/Kanban | serviço MRP | Aceito |
| PLN-06 | Classificação com descendentes | cálculo tenant | Aceito |
| PLN-07 | Item exclusivo de pedido | parâmetros de execução | Aceito |
| PLN-08 | Inter-fábrica/liberação automática | migrations 207/212 | Aceito |
| PLN-09 | Número inicial e LLC opcional | migrations 209/210 | Aceito |
| PLN-10 | Lotes, segurança, lead time e calendário | suíte MRP | Aceito |
| DEM-01 | Pedidos, previsão, independente e segurança | integração de fontes | Aceito |
| DEM-02 | Explosão dependente multinível | domínio MRP | Aceito |
| DEM-03 | Assistência por divisão/almoxarifado | parâmetros 17/45 | Aceito |
| DEM-04 | Estoque e suprimento firme tenant-aware | adapters/integrado | Aceito |
| ORD-01 | Liberar/firmar/replanejar individual/lote | transitions | Aceito |
| ORD-02 | Datas firmes imutáveis | testes de transição | Aceito |
| ORD-03 | Bloqueio Kanban parâmetro 25 | release validator | Aceito |
| ORD-04 | Terceiros parâmetro 45 | integração parâmetro 45 | Aceito |
| ORD-05 | OF→requisição→OCS | integração de requisição | Aceito |
| ENT-01 | Entrega atômica/idempotente | rollback integrado | Aceito |
| ENT-02 | EPP/EPE, REP e co-produtos | delivery integrado | Aceito |
| ENT-03 | Bloqueio com OCS pendente | complete integrado | Aceito |
| RPT-01 | Perfil analítico/sintético e origens | mrp_report | Aceito |
| RPT-02 | Disponibilidade por pedido/item | mrp_report | Aceito |
| RPT-03 | Necessidades em seis períodos | mrp_report | Aceito |
| RPT-04 | Explosão simples/custo/saldo | mrp_report | Aceito |
| RPT-05 | Reposição por saldo/compras/OF/Kanban | mrp_report | Aceito |
| DRW-01 | Revisões, aprovação, motivo e distribuição | drawing integrado | Aceito |
| DRW-02 | Desenho por item/configuração | drawing integrado | Aceito |
| DRW-03 | Replicação conforme parâmetro 8 | drawing integrado | Aceito |
| PUR-01 | Todos os filtros da consulta de compras | purchase integrado | Aceito |
| PUR-02 | Atendido/Pendente/Cancelado e todos itens | purchase integrado | Aceito |
| PUR-03 | Conversão por data-base | USD→BRL integrado | Aceito |
| PUR-04 | Kanban/comprador/tipo/importação/cliente | purchase integrado | Aceito |
| PUR-05 | Totais/frete e impostos somente OCL | purchase integrado | Aceito |
| PUR-06 | Anexos/download tenant-aware | purchase integrado | Aceito |
| MNT-01 | OF manual e demandas do primeiro nível | criação integrada | Aceito |
| MNT-02 | Número/planejador/transferência de linha | migration 230 | Aceito |
| MNT-03 | Bloqueios atividade/Kanban/OFC | manutenção integrada | Aceito |
| MNT-04 | Parâmetros 10/14, produzido e fração | manutenção focada | Aceito |
| MNT-05 | OCS/WMS, demandas e substitutos | material integrado | Aceito |
| MNT-06 | Retrabalho parâmetro 66 | integração parâmetro 66 | Aceito |
| MNT-07 | Lote temporário e datas | fase 4 integrada | Aceito |
| SCR-01 | Destinar item ou demandas | refugo integrado | Aceito |
| SCR-02 | Limites por refugo/requisição | refugo integrado | Aceito |
| SCR-03 | Período/intervalo/valorização | período fechado | Aceito |
| SCR-04 | Grupo secundário e conversão UM | UN→KG integrado | Aceito |
| SCR-05 | Lote/endereço de origem | fase 4 integrada | Aceito |
| SCR-06 | Movimentos de devolução/sucata | saldo integrado | Aceito |
| SCR-07 | Exclusão compensada sem saldo negativo | reversão integrada | Aceito |
| SCR-08 | Alteração preservando ID e rollback | PUT/update integrado | Aceito |
| LOT-01 | Seleção unitária e várias OFs | lot allocation | Aceito |
| LOT-02 | FIFO, saldo e necessidade | integração de saldo | Aceito |
| LOT-03 | Intermediário WMS | integração WMS | Aceito |
| LOT-04 | Parâmetros 53 e 44 A/I/E | fase 4 integrada | Aceito |
| LOT-05 | Endereço ativo | fase 4 integrada | Aceito |
| LOT-06 | Distribuição crescente por OF | batch integrado | Aceito |
| LOT-07 | Confirmação parcial explícita | fase 4 integrada | Aceito |
| NFR-01 | Isolamento por empresa | auditoria SQL/tenant | Aceito |
| NFR-02 | Precisão numeric/decimal | testes de precisão | Aceito |
| NFR-03 | Concorrência e race | `go test -race` | Aceito |
| NFR-04 | Volume industrial sintético | 2.000+ OFs/k6 | Aceito |
| NFR-05 | Hardening HTTP/JWT | portão segurança | Aceito |

## Portões

- `scripts/audit-mrp-manufacturing.sh` — funcional, schema, regressão e CI.
- `scripts/test-mrp-manufacturing-security.sh` — hardening, JWT e tenant.
- `MANUFACTURING_VOLUME_ROWS=10000 make test-integration` — volume PostgreSQL.
- `k6 run scripts/loadtest/k6/mrp-manufacturing.js` — erro `<1%`, p95 `<1,5s`, p99 `<3s`.
