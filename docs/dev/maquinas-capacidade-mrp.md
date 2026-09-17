# Máquinas e capacidade por item

## Pesquisa de referência
- SAP: centro de trabalho fornece recurso e parâmetros; operações do roteiro definem
  preparação, execução e quantidade-base para cálculo de duração/carga.
  https://help.sap.com/docs/SAP_S4HANA_ON-PREMISE/5e23dc8fe9be4fd496f8ab556667ea05/b1a5b953495bb44ce10000000a174cb4.html
- Oracle: consumo do recurso por item ou lote; CRP utiliza quantidades e datas do planejamento.
  https://docs.oracle.com/cd/E26401_01/doc.122/e48792/T473818T474891.htm
  https://docs.oracle.com/en/cloud/saas/supply-chain-and-manufacturing/25c/faumf/how-you-manage-resources.html
- Focco: recurso ligado ao roteiro participa do CRP e sequenciamento; recurso preferencial
  do roteiro orienta planejamento. A capacidade do centro pode ser expressa em horas/minutos.
  https://help.foccoerp.com.br/Programas/FoccoERP/Cadastros%20Auxiliares/Manufatura/Engenharia/Roteiro%20de%20Fabrica%C3%A7%C3%A3o/FENG0111/
  https://help-preview.foccoerp.com.br/Processos/Manufatura/roteiro-de-fabricacao/

## Decisões do Venture
Disponibilidade é tempo de calendário. Produtividade pertence ao par item/máscara/máquina.
Produção contínua: quantidade / quantidade-base × tempo-base / eficiência + setup.
Ciclos fechados: teto(quantidade / quantidade-base) × tempo-base / eficiência + setup.
Setup é informado em minutos, uma vez por ordem. Eficiência informada no item substitui
(sem multiplicar) a eficiência padrão da máquina. Taxa real medida usa eficiência 100%.
Exemplo: 120 peças/h, eficiência 80%, 60 peças, setup 15 min = 52,5 min no modo contínuo.
No modo de ciclos inteiros o mesmo lote ocupa 90 min.

Não converter peças/hora em horas disponíveis. A capacidade nominal antiga permanece
para compatibilidade; horas disponíveis e calendários sustentam a análise de ocupação.
MRP propõe materiais e ordens; CRP compara carga; APS sequencia ordens liberadas.

## Planejamento e liberação
- MRP agenda sugestões pela necessidade, respeitando componentes antes da montagem.
- Prioriza a máscara exata e depois a prioridade da máquina; não usa perfil de outra máscara.
- Roteiros com várias operações reservam todos os recursos, em ordem de precedência.
  Tempo explícito da operação prevalece sobre a taxa do item; sem override, usa a taxa medida.
- Turnos sobrepostos são unidos. Paradas e reservas de outros planos são descontadas.
- Produção proporcional pode continuar na próxima janela. Ciclos fechados só entram inteiros.
- Sem janela suficiente no horizonte de um ano, retorna erro e desfaz o replanejamento.
- Filas, espera, movimentação e prazo externo adiam o término sem criar carga de máquina.
- Mantém a necessidade original e expõe início, término previsto e atraso por capacidade.
- Ao liberar, transfere máquina, duração, término e intervalos às operações da OF. O APS
  preserva a programação dessas ordens liberadas; a reserva permanece ocupando capacidade.
- CRP soma os intervalos produtivos por dia, sem duplicar a sugestão após sua conversão.
- Falta de dados de carga é exposta; calendário fechado não vira disponibilidade fictícia.

## Revisão: correções e acréscimos

**Eficiência coerente entre ciclo e capacidade.** A capacidade por minuto era
calculada com a eficiência da MÁQUINA enquanto o ciclo já usava a do ITEM. O
resultado reportava "eficiência 50% (item)" com capacidade a 100% — o dobro — e
o indicador de gargalo ficava otimista justamente nos itens que rendem menos
naquela máquina. As duas passam a usar a mesma eficiência aplicada.

**Turno que vira o dia** (migração 349). O `CHECK (end_time > start_time)`
tornava o terceiro turno impossível: 22:00–06:00 era recusado pelo banco. O
contorno seria partir em duas janelas, e aí um ciclo fechado de três horas não
cabe em nenhuma das metades. Fim menor ou igual ao início passa a significar
"termina no dia seguinte", nos TRÊS construtores de janela — MRP, APS e CRP.
Corrigir um só faria o turno existir para o planejamento e não para a carga.
O turno pertence ao dia em que começa.

**Fuso fixado na transação de planejamento.** O agendamento traz paradas e
sequências de `timestamptz` para hora de parede com `::timestamp`, e essa
conversão usa o fuso da SESSÃO. Com o Postgres em UTC — o padrão da imagem
oficial — uma parada das 08:00 virava 11:00 e bloqueava três horas erradas, sem
erro nenhum. A transação passa a fixar o fuso da empresa (`SET LOCAL`, o mesmo
cadastro usado para agendar notificações); a trava é local e não vaza para a
conexão do pool.

**Índice de ocupação por máquina.** `mrp_machine_allocation_slots` só tinha a
chave primária `(suggestion_code, starts_at)`, e a consulta de ocupação filtra
por máquina e faixa de tempo — varria a tabela inteira a cada sugestão × etapa.
Com 50 mil slots a janela de um ano caiu de 141 ms para 60 ms mesmo no pior caso
(todos os slots na mesma máquina).

**Consumíveis** (migração 350). Um cilindro não dura "N horas": dura conforme o
que está sendo cortado. Por isso a AUTONOMIA (quanto rende uma carga, quanto
demora a troca) fica em `machine_consumables`, por máquina, e a TAXA DE CONSUMO
fica no par item × máscara × máquina, no mesmo grão da produtividade — que é
onde o sistema sabe o que está sendo feito. O planejamento conta as trocas
(`teto(consumo / capacidade) − 1`; a primeira carga já está montada) e soma o
tempo delas à ocupação. Preparação não corta, então não consome. A chave
estrangeira é COMPOSTA `(consumable_id, machine_code)`: apontar o consumo de um
item para o consumível de outra máquina é impossível no banco, não uma
conferência que o código precise lembrar de fazer.

**Tela reorganizada em abas.** A `VMAQ0200` empilhava sete cadastros numa
rolagem só. São nove abas, cada uma respondendo a uma pergunta: centros de
trabalho, máquinas, turnos, paradas, consumíveis, produtividade, preparação,
simulador e agenda. Turnos e paradas não tinham tela nenhuma — a API existia e
só era alcançável pelo console genérico de rotas, então o usuário podia escolher
um calendário mas não criá-lo, e não tinha como registrar uma quebra.

## Segunda revisão: o que apareceu usando

**403 em inglês, fora do envelope.** `RequireRole` e `RequirePermission` devolviam
`http.Error(w, "forbidden", …)` — texto cru, enquanto todo o resto do sistema
responde JSON em português. Era isso que chegava à tela. Agora dizem qual perfil
o usuário tem, qual a operação exige, e que o perfil vale **por empresa**
(`user_enterprises.role`, não `users.role` — os dois podem divergir).

**Permissão invertida.** Um `USER` criava a máquina e a produtividade dela, mas
não podia cadastrar o turno nem registrar uma parada. Cadastrar turno, parada e
transição de setup passa a aceitar `USER`; excluir continua restrito a `ADMIN`,
porque apagar reescreve capacidade que o planejamento já usou.

**Lookup memoizado escondia o cadastro novo.** `loadMachines` guarda a lista pela
vida da página e `resetLookups()` não era chamado em lugar nenhum: a máquina
criada numa aba não aparecia no modal da outra. A tela invalida ao gravar
máquina, centro de trabalho e consumível.

**SEGUNDO como unidade de tempo** (migração 351). A ficha de chão de fábrica traz
o ciclo em segundos; converter à mão coloca erro de arredondamento na entrada do
dado que governa a capacidade.

**Família de preparação** (migração 351). A matriz já aceitava regra por família
com coringa — o que evita a explosão combinatória: quarenta chapas pedem três
regras entre famílias, não mil e seiscentos pares. Mas a família vinha de
`commercial_classification_code`, um agrupamento comercial. Agora vem de
`items.setup_family`, que agrupa por processo. Quando várias regras casam, vence
a mais específica: par de itens > família > coringa.

**Parada em aberto** (migração 352). `ends_at` passa a aceitar nulo: a máquina
parou e ainda não voltou. O planejamento trata o intervalo aberto como ocupado
até agora — `COALESCE(ends_at, NOW())` nas três consultas de ocupação. Índice
único parcial garante uma parada aberta por máquina, senão tocar duas vezes no
botão bloquearia o recurso para sempre. Abrir é idempotente: a segunda chamada
devolve a parada que já existe.

A `VPRO1200` é o terminal do operador — escolhe a máquina, toca no motivo, e
toca de novo quando volta. O horário é sempre o do servidor; o relógio da tela
só desenha. O cadastro completo (retroativo, correção, consulta por período)
continua na `VMAQ0200`, aba Paradas.

**Papel vale por empresa.** O que `RequireRole` compara é `user_enterprises.role`,
não `users.role` — os dois podem divergir, e o usuário aparece ADMIN no cadastro
sendo USER no vínculo. É a causa típica de "403" em tela.

## Convenções e limites operacionais
- Jornada sem calendário começa à meia-noite de segunda a sexta. Para horários reais e
  trabalho aos sábados/domingos, cadastre um calendário de turnos.
- Manutenção sem horário bloqueia conservadoramente o dia da máquina no MRP/CRP.
  Paradas com horários exatos devem ser cadastradas em indisponibilidades da máquina.
- Ramos paralelos do roteiro e componentes compartilhados são sequenciados de modo
  conservador. Não há otimização global de nesting, ferramentas, mão de obra ou fornecedores.
- A máquina preferida é escolhida pela prioridade; não há busca global de menor prazo
  entre todas as máquinas alternativas. Ordens firmes mantêm a programação aceita.
- Produção por chapa deve ser convertida para a unidade do item conforme o rendimento;
  a produtividade não calcula automaticamente o aproveitamento de um plano de corte.
- Cadastros antigos preservam ciclos e herança de eficiência. Novos perfis na tela usam
  produção proporcional e eficiência 100%, pois a taxa informada já é produção real.
- `inherit_work_center_hours=true` limpa a jornada específica ao editar; omitir o campo
  e a jornada mantém o valor anterior, para compatibilidade com clientes antigos.
- A matriz de setup por transição alimenta o APS, não o MRP. É proposital: o MRP
  ainda não conhece a sequência, então só o sequenciamento pode aplicar setup
  dependente de transição. Mesmo caminho do SAP.
- O consumo é linear na hora de usinagem. Não há curva por espessura nem por
  programa de corte; para isso, cadastre a taxa por máscara.
- A janela de capacidade é relida por sugestão e por etapa do roteiro. Com o
  índice novo isso deixou de ser o gargalo, mas continua sendo O(n) de consultas.

## Validação
- Suíte Go completa e análise estática (`go test ./...`, `go vet ./...`).
- Integração real em PostgreSQL 16 descartável, migrations até 348 e reversão/reaplicação.
- Simulador e MRP com o mesmo resultado; taxa real sem dupla eficiência; ciclos completos.
- Máscaras, isolamento por empresa, máquina inativa, calendários fechados/sobrepostos,
  paradas, duas ordens concorrentes, transbordo ao dia seguinte e rollback sem capacidade.
- Roteiro com duas operações, transferência para OF, idempotência das reservas,
  preservação da programação no APS e carga do CRP após liberação.
- Frontend: lint, TypeScript/build de produção e Chromium com API simulada para conferir
  jornada, calendário, eficiência em percentual e payload de edição. O teste visual não
  substitui homologação com os dados e turnos reais da fábrica.

## Aplicação
Alterações locais sem commit, push ou release. Aplicar a migration 348 no ambiente de
homologação antes de usar a nova tela. Nenhum banco real foi migrado nesta implementação.

