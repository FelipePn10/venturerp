# Máquinas e Roteiro — Documentação técnica

Documento técnico do **módulo de Máquinas** (tipos, máquinas, tempos por item, agenda)
e da sua relação com o **Roteiro de Fabricação**. A versão de apresentação
(linguagem de negócio, exemplos passo a passo de cálculo de tempo, conversão de
unidades e gargalo) está em [`../apresentacao/maquinas.md`](../apresentacao/maquinas.md).

> Convenções: `Authorization: Bearer <JWT>`, `Content-Type: application/json`.
> Todas as rotas exigem papel `ADMIN` ou `USER`.

---

## 1. Modelo do módulo

O módulo é composto por quatro entidades, todas servidas pelo `MachineHandler`
(`internal/interfaces/http/handler/machine_handler.go`) sobre o repositório SQLC
`machine.NewMachineRepositorySQLC`:

| Entidade | Papel |
|---|---|
| **Tipo de máquina** (`machine type`) | Categoria do equipamento (corte, dobra, solda, pintura, torno…). Apenas organiza. |
| **Máquina** | Equipamento físico: capacidade, unidade, período e eficiência. |
| **Tempo por item × máquina** (`item machine time`) | Tempo de ciclo, quantidade base e setup para fabricar um item (ou variante/máscara) numa máquina. **Cadastro central do cálculo.** |
| **Agenda** (`schedule`) | Disponibilidade/paradas da máquina, consumida pelo CRP/APS. |

Casos de uso (`internal/application/usecase/machine_uc/`): `CreateMachineUseCase`,
`ListMachinesUseCase`, `GetMachineUseCase`, `CreateMachineTypeUseCase`,
`ListMachineTypesUseCase`, `GetMachineTypeUseCase`, `CreateItemMachineTimeUseCase`,
`ListItemMachineTimesUseCase`, `CalculateProductionTimeUseCase`,
`ScheduleMachineUseCase`.

---

## 2. Endpoints (`/api/machine`)

| Método | Rota | Ação |
|---|---|---|
| POST | `/api/machine/create` | Cria máquina (capacidade, unidade, período, eficiência) |
| GET | `/api/machine/list` | Lista máquinas |
| GET | `/api/machine/{code}` | Busca máquina por código |
| POST | `/api/machine/types/create` | Cria tipo de máquina |
| GET | `/api/machine/types/list` | Lista tipos |
| GET | `/api/machine/types/{code}` | Busca tipo por código |
| POST | `/api/machine/time/create` | Cria tempo por item × máquina (variante opcional) |
| GET | `/api/machine/time/list` | Lista tempos |
| POST | `/api/machine/time/{code}` | Busca tempo por código |
| POST | `/api/machine/time/production/calculate` | **Calcula o tempo de produção** de uma quantidade |
| POST | `/api/machine/schedule/create` | Cria agenda/disponibilidade |
| GET | `/api/machine/schedule/list` | Lista agendas |
| POST | `/api/machine/schedule/{code}` | Busca agenda por código |

> Exemplos de corpo de request em [`API_REQUEST_BODIES.txt`](API_REQUEST_BODIES.txt).

---

## 3. Cálculo de tempo de produção

`CalculateProductionTimeUseCase` (`POST /api/machine/time/production/calculate`):

1. Resolve a configuração de tempo pela **variante (máscara)** do item; na ausência,
   usa a configuração **padrão** (sem variante).
2. **Normaliza o período** do tempo de ciclo para minutos (min/hora/dia → minutos;
   1 dia = 480 min / 1 turno de 8h).
3. **Verifica compatibilidade de unidade** item × máquina e converte quando físico
   (kg↔t, mm↔m, m³↔L, peça/unidade). Incompatibilidades (ex.: kg × m) são bloqueadas.
4. `ciclos = teto(quantidade / quantidade_base)` (**arredonda para cima** — último
   ciclo parcial ocupa a máquina inteira).
5. `tempo_total = ciclos × tempo_ciclo + setup` (setup uma única vez).
6. Avalia **gargalo**: compara a vazão exigida com a capacidade efetiva
   (`capacidade × eficiência`); sinaliza sobrecarga.
7. Retorna o prazo em minutos/horas/dias.

A regra de **prioridade** (campo no tempo por item×máquina) define qual máquina é
escolhida quando o item pode ser feito em mais de uma (1 = preferida).

> A lógica detalhada, com exemplos numéricos, está na versão de apresentação.

---

## 4. Relação com o Roteiro de Fabricação

O **Roteiro** (operações encadeadas por centro de trabalho) é documentado em
[`manufatura-e-compras.md`](manufatura-e-compras.md) **§1**, incluindo o cálculo de
**lead time via CPM** (caminho crítico) que alimenta as datas do MRP. A ponte entre
os dois módulos:

- cada **operação** do roteiro referencia um **centro de trabalho / máquina**;
- o **tempo por item × máquina** fornece setup e tempo de ciclo daquela operação;
- a soma das operações (CPM) é o lead time de fabricação usado pelo MRP, e a carga
  por máquina/dia é o que o **CRP** soma e o **APS** sequencia (ver
  `manufatura-e-compras.md` §2 e §3).

---

## 5. Migrations e referências

- Cadastro de máquinas/tempos: ver `project_mrp_machine` (migration base 000058) e o
  histórico em `migrations/`.
- Item × máquina e conversão de UM: [`cadastros-item.md`](cadastros-item.md) §7 e
  `manufatura-e-compras.md` §11 (conversão de UM por item).
