# Sequenciamento da produção

Implementação da task `.ai/tasks/sequenciamento-de-produto.md`, ampliando o APS existente.

## Fluxo e APIs

1. Cadastre grupos (`POST/GET /api/aps/resource-groups`) e calendários
   (`POST/GET /api/aps/machine-calendars`).
2. Configure máquinas (`PUT /api/aps/resources/{id}/sequencing`), centros
   (`PUT /api/aps/work-centers/{id}/sequencing`) e o parâmetro 48
   (`PUT /api/aps/sequence/settings`).
3. Calcule em `POST /api/aps/sequence`, selecionando `order_ids`, `machine_ids`,
   `work_center_ids`, `operation_ids` e `start_from`.
4. Consulte `POST /api/aps/sequence/view`: exige `from`, `to` e exatamente um
   `resource_group_id`; aceita faixas de ordem, máquina, centro e planejador.
5. Exporte refugo/paradas em `POST /api/aps/sequence/events/export?format=csv`.

`GET /api/aps/sequence/resources` é a LOV que respeita o parâmetro 48. A unidade
da consulta é `HOUR` ou `MINUTE`; `refresh_value` não pode ser negativo. Todas as
operações usam a empresa autenticada.

## Rastreabilidade requisito a requisito

| ID | Requisito | Evidência | Estado |
|---|---|---|---|
| SEQ-01 | Calcular OFs liberadas | APS/EDD existente | Concluído |
| SEQ-02 | Selecionar OF/máquina/centro/operação | DTO e repositório tenant | Concluído |
| SEQ-03 | Exportar refugo/paradas | JSON/CSV | Concluído |
| SEQ-04 | Parâmetro 48: somente ativos | settings + LOV | Concluído |
| VIEW-01 | Data/hora inicial e final | `/sequence/view` | Concluído |
| VIEW-02 | Unidade e valor | validação no use case | Concluído |
| VIEW-03 | Um grupo por consulta | grupo obrigatório | Concluído |
| VIEW-04 | Faixas OF/máquina/centro/planejador | filtros tenant | Concluído |
| WC-01 | Código/descrição | `machine_types` existente | Já implementado |
| WC-02 | CC máquina/homem distintos | migration 231 + use case | Concluído |
| WC-03 | Capacidade hierárquica | centro + máquina | Concluído |
| EMP-01 | Funcionário/situação/funções/contatos | módulo existente | Já implementado |
| MAC-01 | Recurso/situação/capacidade | módulo de máquinas | Já implementado |
| MAC-02 | Grupo/calendário/local/crítico | migration 231 + API | Concluído |
| GRP-01 | Cadastro de grupos | API e tabela tenant | Concluído |
| CAL-01 | Calendário semanal | intervalos validados | Concluído |
| TEN-01 | Isolamento por empresa | contexto + integração | Concluído |
| MAC-03 | Máquina efetivamente escolhida | `production_sequences.machine_id` | Concluído |
| STOP-01 | Parada planejada/não planejada por intervalo | CRUD `machine-downtimes` | Concluído |
| EMP-02 | Contatos, funções, CC e crédito | perfil transacional do funcionário | Concluído |
| SVC-01 | Serviços, itens e responsáveis | perfil industrial da máquina | Concluído |
| SPC-01 | Dados especiais texto/numérico | perfil industrial da máquina | Concluído |
| NFR-01 | Concorrência e race detector | script de aceite | Concluído |
| NFR-02 | Volume industrial | 10.000 registros; leitura 8,45 ms | Concluído |
| NFR-03 | Carga HTTP | 6.309 requests, erro 0%, p95 1,18 ms | Concluído |

O CC homem é opcional e não pode repetir o CC máquina. Intervalos de calendário
não cruzam a meia-noite. Serviços preventivos configuram periodicidade, materiais
e responsáveis; suas execuções continuam integradas às ordens de manutenção.

## Perfis e paradas

- `PUT /api/aps/employees/{id}/sequencing-profile`: telefones, e-mails, funções,
  centro de custo, supervisor/gerente e crédito, em uma transação.
- `GET /api/aps/employees/{id}/sequencing-profile`: retorna o perfil com os IDs
  necessários para edição granular.
- `PATCH/DELETE /api/aps/employees/{employeeID}/contacts/{contactID}` e
  `/functions/{functionID}`: altera ou exclui apenas o subregistro selecionado.
- `PUT /api/aps/resources/{id}/industrial-profile`: aquisição, preparação, marca,
  preferência, responsável, serviços preventivos, itens, responsáveis e dados especiais.
- `GET /api/aps/resources/{id}/industrial-profile`: retorna IDs dos vínculos de
  serviço, itens e campos especiais.
- `PATCH/DELETE /api/aps/resources/{machineID}/services/{serviceID}`: manutenção
  individual do vínculo preventivo e de seus responsáveis.
- `PATCH/DELETE /api/aps/resources/{machineID}/services/{serviceID}/items/{itemID}`:
  manutenção individual dos materiais do serviço.
- `PATCH/DELETE /api/aps/resources/{machineID}/special-values/{fieldID}`:
  manutenção individual do campo e valor especial da máquina.
- `POST/GET/DELETE /api/aps/machine-downtimes`: paradas exatas planejadas,
  imprevistas ou vinculadas à manutenção.

Todas as alterações granulares exigem `ADMIN` e retornam `204`; consultas aceitam
`ADMIN` ou `USER`. Os IDs do pai e do subregistro são verificados juntamente com
a empresa autenticada. Um registro de outra empresa ou pertencente a outro pai
não é alterado. O `PUT` continua sendo substituição integral transacional: omitir
um subregistro no payload o exclui; use `PATCH` para edição pontual.

Não foi necessária migration adicional para as rotas granulares. Com o banco na
versão limpa 232, migrations futuras numeradas a partir de 233 são aplicadas pelo
fluxo normal `make migrate_up`; não é necessário executar `force` em um banco limpo.

O custo padrão já separa `MachineRate()` e `LaborRate()` e multiplica o esforço
de máquina e mão de obra do roteiro independentemente. Os centros de custo da
máquina e do homem identificam as origens contábeis; o CC homem é opcional e não
pode coincidir com o CC máquina.

## Recuperação de instalação legada na migration 206

Instalações que registraram a migration 093, mas perderam manualmente a tabela
`stock_balances`, falhavam na 206. A 206 agora recria idempotentemente a estrutura
canônica antes de adicionar `enterprise_id`. Se o migrador ficou em `206 (dirty)`,
após publicar estes arquivos execute contra o mesmo banco:

```bash
migrate -path migrations -database "$DATABASE_URL" force 205
migrate -path migrations -database "$DATABASE_URL" up
```

Durante a recuperação do ambiente legado também foram detectados e reparados:

- migration 216: `stock_movements` sem `created_by` e colunas de referência;
- migration 220: ausência de reservas, médias de consumo, lotes e inventários.

Cada cenário foi reproduzido isoladamente até a versão 232. Em 12/07/2026 o
Supabase informado foi recuperado de `206 dirty`, depois de `216 dirty` e
`220 dirty`, chegando à versão limpa `232`.
