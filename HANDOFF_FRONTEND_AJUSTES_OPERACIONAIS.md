# Handoff frontend — ajustes operacionais

Este documento consolida as decisões e contratos da task de ajustes operacionais.
O agente de frontend deve observar o status de cada seção antes de implementar.

## Estado da branch

- Branch backend: `fix/ajustes-operacionais`.
- Base: `main` v1.1.4, commit `5bbb759`.
- Worktree: `/home/felipepanosso/GolandProjects/panossoerp-ajustes`.
- A `develop` possui outra task local e não deve receber estas mudanças.
- As alterações descritas como implementadas abaixo estão neste worktree.

## Convenções confirmadas

- O código do próprio item é alfanumérico desde a criação, por exemplo `TEA452-0`.
- Empresas diferentes podem cadastrar o mesmo código de item.
- A unicidade será `(enterprise_id, code)`.
- Códigos são strings; nunca converter para número ou remover zeros à esquerda.
- Relacionamentos persistentes devem usar o `id` interno imutável do item.
- Rotas podem permanecer em inglês.
- Enums e valores controlados novos devem ser apresentados em português.
- JSON permanece em `snake_case`.
- Regras de aplicabilidade são validadas no backend, não apenas pela interface.

## 1. Código alfanumérico do item — IMPLEMENTADO (ROLLOUT COMPATÍVEL)

Exemplo do contrato pretendido:

```json
{
  "code": "TEA452-0",
  "name": "Tubo de aço",
  "purchasable": true,
  "manufactured": false,
  "sellable": false,
  "stocked": true,
  "service": false
}
```

Normalização:

- remover apenas espaços externos;
- converter letras para maiúsculas;
- aceitar letras, números, hífen, ponto, barra e sublinhado;
- máximo inicial de 60 caracteres;
- preservar códigos numéricos legados como texto.

`POST /api/items/create` recebe `code` como string. `GET /api/items/search/{code}`
aceita o código alfanumérico. A resposta traz `code` textual e `legacy_code`
numérico durante a migração dos módulos antigos. O frontend não deve exibir nem
permitir edição de `legacy_code`.

### Classificações fiscais de compra e venda

- Item somente comprado: classificação de compra aplicável; venda não obrigatória.
- Fabricado para venda: classificação de venda aplicável; compra somente se comprável.
- Revenda: classificações de compra e venda aplicáveis.
- Serviço: regras fiscais próprias.
- Quando compra e venda usam o mesmo mestre fiscal, valores específicos devem ser
  overrides, e não duplicações obrigatórias.
- Compra e venda são contextos fiscais independentes do tipo de reposição. Um
  item comprado pode ter as duas classificações quando também é vendido.
- Nenhuma das duas classificações é obrigatória apenas por o item ser comprado.
- O backend valida referências informadas e responde `422` se o mestre não existe.

## 2. Classificação hierárquica de itens — VALIDAÇÕES IMPLEMENTADAS

Este cadastro não é classificação fiscal ou NCM. Ele agrupa itens para MRP,
Compras e Custos por máscaras como `99.99.99`.

### Criar máscara

`POST /api/items/classifications/masks/`

```json
{
  "mask": "99.99.99",
  "description": "CLASSIFICACAO INDUSTRIAL DE ITENS"
}
```

O backend gera o `code` da máscara. Esse valor é usado como `mask_code`.

### Criar nível raiz

`POST /api/items/classifications/`

```json
{
  "code": "10",
  "mask_code": 1,
  "description": "METAIS"
}
```

### Criar níveis filhos

```json
{
  "code": "10.20",
  "mask_code": 1,
  "parent_code": "10",
  "description": "CHAPAS"
}
```

```json
{
  "code": "10.20.30",
  "mask_code": 1,
  "parent_code": "10.20",
  "description": "CHAPAS DE ACO CARBONO"
}
```

O fluxo acima foi testado ponta a ponta no ambiente de treinamento em 12/08/2026.
Os níveis `1`, `2`, `3` e respectivos `parent_id` foram persistidos corretamente.
Os dados de teste foram removidos ao final.

Implementado: validação da máscara, dígitos por segmento, prefixo completo do pai,
nível raiz, bloqueio de saltos de nível, isolamento por empresa e geração do
código da máscara por sequence (segura em concorrência).

Pendências de rollout:

- impedir saltos de nível e ciclos;
- usar HTTP `409` para duplicidade e `422` para hierarquia inválida;
- preservar busca por descendentes nos filtros de MRP, Compras e Custos.

## 3. Calendário industrial — IMPLEMENTADO

`POST /api/industrial-calendar/generate/{year}/{month}`

- segunda a sexta são úteis por padrão;
- não sobrescrever dias existentes.
- resposta: `year`, `month`, `created`, `preserved`.
- origem em português: `AUTOMATICO`, `FIM_DE_SEMANA`, `FERIADO`, `MANUAL`.
- calendário isolado por empresa.

## 4. Exportação PDF/VITM0100 — IMPLEMENTADO

`POST /api/reports/export?format=pdf` aceitará:

```json
{
  "orientation": "paisagem"
}
```

Valores: `retrato` e `paisagem`.

A logo deve usar caixa fixa com proporção preservada, sem sobrepor a razão social,
em primeira página e continuações. Foram validados casos horizontal, quadrado,
vertical, nome empresarial longo e regressão do romaneio.

## 5. Item por fornecedor — BUSCA E RASTREABILIDADE IMPLEMENTADAS

Um item interno aceitará vários fornecedores, cada qual com código, descrição,
unidade, conversão, preferência e vigência próprios. Haverá resolução por código
ou descrição externa e rastreabilidade na entrada fiscal.

Busca: `GET /api/item-suppliers/search?supplier_code=123&term=ABC`. O retorno
prioriza o código externo exato e depois a descrição. Estratégias fiscais:
`CODIGO_FORNECEDOR`, `DESCRICAO_FORNECEDOR` e `MANUAL`.

## 6. Parametrização fiscal automática — PLANEJADO

O item deve informar valores fiscais efetivos, origem herdada/override e referência
ao cadastro mestre. Valores controlados novos serão publicados em português.

## 7. Código de barras da ordem de produção — PLANEJADO

Será usado token opaco validado pelo servidor, com resolução, início/conclusão,
idempotência persistente, transições válidas e auditoria por empresa, usuário e
dispositivo.

## 8. Auditoria de campos duplicados — DECISÃO DE GOVERNANÇA

Não remover campos no frontend até classificá-los como fonte da verdade, derivado,
override ou conceito distinto. Prioridades: descrição, unidade, peso, classificação,
comprado/fabricado, mínimos de estoque e códigos externos de fornecedores.

- Nome, descrição técnica PDM e descrição comercial são conceitos distintos.
- UM de estoque é base; UMs de compra/venda são contextuais e usam conversão.
- Peso de engenharia é mestre; pesos fiscais/logísticos são transacionais.
- Estoque mínimo, segurança e ponto de reposição não são duplicados.
- Classificações comercial, contábil e fiscal têm finalidades diferentes.

## Operação realizada em produção

Em 12/08/2026 foi feita uma limpeza autorizada no banco de produção.

Preservados:

- 5 usuários;
- 1 empresa;
- 5 vínculos usuário–empresa;
- perfis, permissões e configuração fiscal;
- tabelas mestres de país, UF, NCM e ICMS;
- migrations.

Removidos:

- item de teste `CHAPA AÇO CARBONO 3 mm`;
- classificações/máscaras vinculadas ao item;
- centros de custo cadastrados;
- funcionários, calendário e parâmetros operacionais de teste;
- máquinas/tipos, conta bancária, grupos, modificadores e auditoria anterior.

Backup verificado antes da operação:

```text
/var/backups/venturerp/releases/pre-cleanup-20260812T025900Z.dump
```

Após a limpeza, a API permaneceu saudável.
