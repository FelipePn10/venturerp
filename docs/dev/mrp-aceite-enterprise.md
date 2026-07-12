# Aceite enterprise — planejamento de materiais e manufatura

Data da última execução local: 12/07/2026.

## Portões executados

- `make test`: aprovado.
- `make test-integration` com PostgreSQL migrado até `230`: aprovado.
- `make ci` (`fmt-check`, `go vet`, build e cobertura global): aprovado.
- `scripts/test-mrp-manufacturing.sh`: aprovado, incluindo `-race`.
- migrações da task até `230`: aprovadas; `229` e `230` tiveram reversão e
  reaplicação repetidas pelo portão transversal.
- `git diff --check`: aprovado.
- `scripts/audit-mrp-manufacturing.sh`: aprovado; schema/tenant, migrations,
  aceite focado, regressão global e CI executados num único comando.
- `scripts/test-mrp-manufacturing-security.sh`: aprovado; hardening HTTP/JWT e
  isolamento entre empresas em estoque, OF, desenhos e compras.
- ensaio integrado de manutenção com 10.000 OFs sintéticas: aprovado; consulta
  completa em 36,4 ms no ambiente local de teste (limite de aceite: 5 s).
- k6 sobre API e PostgreSQL locais, 25 usuários virtuais e 90 s: 7.794
  requisições, 0% de erros, p95 de 3,23 ms e p99 de 4,26 ms.

## Cenários críticos automatizados

- cálculo exclusivo e isolamento por empresa;
- fontes de demanda, parâmetros de execução, LLC e inter-fábrica;
- entrega idempotente com divisão EPP/EPE;
- rollback integral quando o estoque falha;
- REP, co-produtos, lotes e precisão de seis casas;
- isolamento de movimento/saldo entre duas empresas;
- vínculo OF → requisição de serviço → OCS;
- parâmetro 45 na liberação e parâmetro 66 no retrabalho;
- substituição, bloqueio WMS, intermediário e distribuição de lote entre OFs;
- limite e movimento de estoque na destinação de refugo;
- perfil, disponibilidade, necessidades agrupadas, explosão e reposição.
- desenhos tenant-aware, revisão inicial e replicação;
- consulta de pedidos com posições, conversão, tributos e anexos;
- manutenção de OF, parâmetros 10/14/44/53, lote temporário e transferência;
- períodos, UM, endereço, WMS e reversão de refugo sem saldo negativo;
- alteração de destinação de refugo com preservação do ID, compensação de
  estoque, isolamento por empresa e rollback atômico do estado anterior;
- invariantes SQL de tenant e unicidade do número da OF por empresa.

## Cobertura medida

Cobertura unitária focada após a ampliação de testes:

| Pacote | Cobertura |
|---|---:|
| `mrp_calculation/service` | 21,3% |
| `mrp_calculation_uc` | 42,5% |
| `mrp_uc` | 51,5% |
| `planned_order_uc` | 42,9% |
| `production_plan_uc` | 40,2% |
| `production_order_uc` | 37,9% |
| `mrp_report_uc` | 71,9% |
| `purchase_order_uc` | 24,0% |

A cobertura global do monólito é 13,3%; esse número inclui centenas de pacotes
legados fora desta task. Portanto, o aceite se apoia também nos testes integrados,
race detector, invariantes SQL e matriz de cenários críticos, e não apenas na
porcentagem global.

## Limite da declaração

Os ensaios de volume usam massa industrial sintética e infraestrutura local;
não substituem um teste de capacidade no hardware, topologia e dados anonimizados
do cliente. Testes aprovados reduzem o risco conhecido, mas não demonstram
matematicamente ausência de defeitos em produção. Novas integrações, volumes e
combinações de cadastro devem continuar entrando na suíte de regressão.
