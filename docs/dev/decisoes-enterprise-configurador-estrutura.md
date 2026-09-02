# Decisões enterprise — configurador de produto, restrições e estrutura de produto

## Escopo e referências

Rodada de fechamento de engenharia/cadastros (VCLA0100, VITE0114, VENT0200/0210/0800,
VSUP0120/0130/0500, VBOM0100, VMAQ0200, VCUS0100, configurador e restrições).

A referência funcional primária é o FoccoERP, cuja documentação pública descreve os
programas equivalentes: [FENG0116 — Manutenção de Restrições e
Dependências](https://help.foccoerp.com.br/Programas/FoccoERP/Manufatura/Configurador%20de%20Produto/FENG0116/),
[Configurador de Produto](https://help.foccoerp.com.br/Processos/Manufatura/configurador-de-produto/),
[FENG0210 — Estrutura de Produto](https://help.foccoerp.com.br/Programas/FoccoERP/Manufatura/Engenharia/Estrutura%20de%20Produto/FENG0210/)
e [CENG0401 — Consulta de Estrutura](https://help.foccoerp.com.br/Programas/FoccoERP/Manufatura/Engenharia/Consultas/CENG0401/).
Para os contratos de plataforma (código público do item, versionamento de estrutura,
fórmulas de quantidade e configuração variante) valem as mesmas fontes já usadas em
[decisoes-enterprise-engenharia-manufatura.md](decisoes-enterprise-engenharia-manufatura.md):
SAP S/4HANA (Variant Configuration / Object Dependencies), Oracle Fusion (Configurator
e Work Definitions), Dynamics 365 (Product Configuration Models) e Odoo (Product
Variants com atributos e fórmulas).

## Decisões aplicadas

### 1. O configurador não tem tela própria

Como no FoccoERP, o configurador é **um botão dentro do Cadastro de Estrutura de
Produto**, não um programa separado. O servidor expõe isso como dois contratos, ambos
sob a própria estrutura:

- `GET /api/items/structure/{itemCode}/configurator` devolve o painel inteiro numa
  única leitura: as perguntas do item na ordem de apresentação, as respostas possíveis
  de cada uma (com a resposta padrão marcada), as configurações já geradas, os
  componentes cuja quantidade sai de fórmula e as variáveis que essas fórmulas usam.
- `POST /api/items/structure/{itemCode}/configurator/apply` aplica as respostas:
  valida as restrições, gera a máscara e — quando a configuração é gravada — devolve
  a estrutura já resolvida com as fórmulas avaliadas.

As rotas granulares de `/api/configurator/*` continuam existindo para o cadastro de
conjuntos, variáveis, características e regras. O que **não** existe é uma tela de
operação do configurador: quem configura um produto faz isso de dentro da estrutura.

O painel informa `configurable: false` com uma mensagem explicativa quando o item não
tem características cadastradas, para a tela manter o botão desabilitado em vez de
abrir um painel vazio.

### 2. Restrições e dependências explicam o motivo da recusa

O motor de restrições (`/api/restriction`, equivalente ao FENG0116) já decidia se uma
combinação era válida, mas devolvia apenas sim/não. Isso é insuficiente para a tela:
o usuário precisa saber **qual** resposta violou **qual** dependência.

`ExplainCombination` passa a devolver, além do veredito, a lista de determinantes
violados com característica, operador, valor esperado, valor respondido e uma frase
pronta em PT-BR. O `apply` do configurador responde 422 com código estável
`RESTRICAO_DE_CONFIGURACAO` e o array `violations`, para a tela destacar exatamente as
perguntas problemáticas.

A validação passou a valer para **todo** caminho que gera máscara, não só para a
geração em lote (produto cartesiano): `GenerateMask` agora roda o motor antes de
compor a máscara. Antes era possível gerar por `/api/configurator/generate-mask` uma
combinação que o gerador em lote recusaria.

### 3. Fórmula de quantidade na estrutura

Cada componente da estrutura passa a aceitar uma **fórmula de quantidade** avaliada com
as variáveis da configuração do produto pai — exatamente o caso do enunciado:

```
2*(COMPRIMENTO/1000)+2*(PROFUNDIDADE/1000)
```

Decisões de contrato (migração `000331`):

- `quantity_formula` é opcional. Quando preenchida, `quantity` deixa de ser a
  quantidade efetiva e passa a ser o **valor nominal de reserva**, usado quando a
  fórmula não puder ser avaliada (item sem configuração, variável sem resposta).
  Uma explosão nunca falha por causa de fórmula: ela degrada para o valor nominal.
- `quantity_rounding` (`NONE`/`UP`/`DOWN`/`NEAREST`) e `quantity_scale` (0–6) definem
  o arredondamento do resultado. O padrão é não arredondar.
- As variáveis são os códigos normalizados das características do configurador, em
  maiúsculas. A fórmula é validada na gravação: expressão malformada é recusada com
  422, não descoberta na hora de explodir a OF.
- A **perda** incide sobre o resultado da fórmula, não sobre a quantidade nominal.
- O painel do configurador lista as fórmulas do item e aponta em
  `missing_formula_variables` as variáveis que nenhuma pergunta responde — uma fórmula
  assim nunca seria avaliável, e é melhor dizer isso no cadastro.

A avaliação acontece nos três lugares que precisam dela, com a mesma semântica: a
consulta de estrutura (CENG0401), a resolução da árvore por máscara e a **explosão do
MRP** — sem esta última, a OF continuaria consumindo a quantidade fixa e a fórmula
seria decorativa. O MRP carrega as respostas por par item/máscara uma única vez por
nível e reaproveita o resultado durante toda a explosão.

### 4. Cadastros por código de negócio, ator pelo JWT

Mantida e estendida a decisão já vigente: o código público do item é textual. Nesta
rodada migraram para `TextCode` os contratos que ainda exigiam a chave numérica e por
isso quebravam a tela — cabeçalho de estrutura (VBOM0100), conversões por item
(VSUP0130), preço de compra por item (VSUP0120), parâmetros de estoque por item
(VPRO1100), ordem de fabricação (VPRO0900) e as referências do cadastro de item
(item-base e item de embalagem, VENT0200).

Em todos eles o autor (`created_by`/`updated_by`) sai do JWT e o campo é `json:"-"`:
o corpo da requisição não decide quem assinou o registro. As respostas devolvem o
código de negócio e mantêm a chave interna em `legacy_*` para integrações antigas.

### 5. Multiempresa no cabeçalho de estrutura

`bom_headers` nasceu sem coluna de empresa: qualquer tenant abria ou alterava o
cabeçalho de outro pelo id. A migração `000330` acrescenta `enterprise_id`, herda a
empresa do item quando possível, torna a unicidade de versão por empresa e todas as
consultas do repositório passam a filtrar por tenant.

### 6. Operações que "não faziam nada"

Vários `UPDATE` de status eram `:exec` sem verificação de linhas afetadas — bloquear
um fornecedor inexistente, inativar uma ferramenta inexistente ou zerar a vida útil de
uma que não existe respondiam sucesso sem efeito algum. A regra adotada: **toda
mutação de estado confere a existência antes e o estado depois**, devolvendo 404
quando o registro não existe, 409 quando já está no estado pedido e erro explícito
quando a atualização não surtiu efeito.

## Fronteira operação → roteiro → OF (VPRO0100 × VENT0115 × VENT0202)

A tela canônica de roteiro é `/api/routing`, com três papéis distintos:

| Papel | Tela | Contrato |
| --- | --- | --- |
| Biblioteca de operações | VENT0115 | `/api/routing/operations` |
| Roteiro por item/máscara | VENT0202 | `/api/routing/routes` |
| Consulta operacional do roteiro na produção | VPRO0100 | leitura de `/api/routing` |

VPRO0100 não tem contrato próprio: é uma visão de produção sobre o mesmo cadastro.
O cadastro vive em VENT0115 (modelos de operação) e VENT0202 (roteiro do item); a
produção lê. O ciclo completo é operação → roteiro → ordem planejada (MRP) → OF
firmada (com fotografia de operações e materiais) → CRP/APS → apontamento.

## Catálogos para os modais

Telas que exigiam o usuário digitar um código de cabeça passam a ter catálogo:

- `GET /api/crp/plans` lista os planos da empresa com resumo da carga calculada e a
  marca `calculated`. Com `only_calculated=true` devolve apenas os que já têm carga —
  são os únicos que produzem exportação com conteúdo. A exportação vazia de VPRO0200
  vinha justamente de calcular sobre um plano sem CRP.
- `GET /api/standard-cost/work-centers` (VCUS0100) e `/api/machine/list` (VMAQ0200)
  já cumpriam esse papel e foram mantidos.
- As consultas de carga do CRP passaram a validar que o plano pertence à empresa
  autenticada; antes bastava adivinhar o código.

## Mensagens de domínio

Mensagens de usuário em PT-BR com acento, nomeando o campo e, quando o valor vem de um
conjunto fechado, listando as opções aceitas. Códigos técnicos de enum (`MBOM`,
`APPROVED`, `TRANSFER`, `A`/`I`/`E`) continuam estáveis no contrato, mas a resposta
acompanha um rótulo pronto para exibição (`status_label`, `bom_type_label`) ou a
explicação junto da mensagem — por exemplo "use A (automático), I (informado pelo
operador) ou E (estorno para o lote de origem)".

O mapeamento HTTP é uniforme via `security.RespondUseCaseError`: 422 para validação de
domínio, 404 para registro inexistente, 409 para conflito de estado e 403 para
autorização. Os handlers que devolviam 500 para qualquer falha do caso de uso (criação
de OF, máquina, APS, ferramenta, CRP) foram corrigidos.
