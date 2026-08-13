# Handoff frontend — ajustes operacionais

Este documento consolida as decisões e contratos da task de ajustes operacionais.
O agente de frontend deve observar o status de cada seção antes de implementar.

## Estado da branch

- Branch backend: `fix/ajustes-operacionais`.
- Base: `main` v1.1.4, commit `5bbb759`.
- Worktree: `/home/felipepanosso/GolandProjects/panossoerp-ajustes`.
- A `develop` possui outra task local e não deve receber estas mudanças.
- As alterações descritas como implementadas abaixo estão neste worktree.

## Ambiente de treinamento validado em 13/08/2026

- imagem `venturerp/api:training` reconstruída a partir deste worktree;
- banco atualizado pela sequência combinada e compatível até a migration `000306`;
- somente `venturerp-api-training` foi recriado, sem alterar produção/demo/development;
- login do instrutor: HTTP `200`;
- calendário `generate`: rota publicada (payload inválido controlado retornou `422`);
- calendário de compatibilidade: rota publicada (`422` controlado);
- busca item-fornecedor: HTTP `200`;
- scanner: rota publicada (corpo vazio controlado retornou `422`);
- senha e token não foram impressos durante o smoke.

## Convenções confirmadas

- O código do próprio item é alfanumérico desde a criação, por exemplo `TEA452-0`.
- Empresas diferentes podem cadastrar o mesmo código de item.
- A unicidade pública é `(enterprise_id, code)`; no banco o campo público é
  `business_code` e `code` numérico permanece apenas como chave legada interna.
- Códigos são strings; nunca converter para número ou remover zeros à esquerda.
- Relacionamentos persistentes devem usar o `id` interno imutável do item.
- Rotas podem permanecer em inglês.
- Enums e valores controlados novos devem ser apresentados em português.
- JSON permanece em `snake_case`.
- Regras de aplicabilidade são validadas no backend, não apenas pela interface.

## 1. Código alfanumérico do item — IMPLEMENTADO (ROLLOUT COMPATÍVEL)

Campo alterado no contrato implementado (este trecho não é um payload completo; o
cadastro continua exigindo os blocos já usados pelo frontend):

```json
{
  "code": "TEA452-0",
  "name": "Tubo de aço"
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

`code` pode ser omitido ou enviado vazio. Nesse caso o backend gera uma sequência
numérica por empresa (`1`, `2`, `3`...), considerando somente códigos automáticos.
A reserva é atômica e pula eventual colisão com um código manual numérico.

### Referências transversais

Nas requisições autenticadas, referências como `item_code`, `parent_item_code`,
`child_item_code`, `root_item_code`, `material_item_code`, `band_item_code`,
`scrap_item_code`, `service_item_code`, `reference_item_code`, `order_item_code`,
`packaging_item_code` e listas `item_codes` aceitam códigos alfanuméricos.
O mesmo vale para `engineering.item_base_cod`; o frontend envia o código comercial
e ignora `legacy_item_base_cod`, mantido somente durante o rollout.

Isso vale para body JSON, query params e rotas identificadas como rotas de item.
O backend resolve o código dentro da empresa autenticada para o ID imutável usado
pelas FKs. Respostas devolvem o código comercial e acrescentam temporariamente
`legacy_<nome_do_campo>` com a referência antiga.

Existe teste HTTP com PostgreSQL cobrindo item-fornecedor, estoque, pedido de
venda, MRP, estrutura e ordem de produção, incluindo o mesmo código em duas
empresas e preservação de zeros à esquerda.

```json
{"item_code":"TEA452-0","quantity":"10.500"}
```

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

Observações de rollout:

- saltos de nível, prefixo inválido e ciclos já são bloqueados pelo backend;
- atualmente erros de duplicidade/hierarquia podem chegar como `422`; o frontend
  deve mostrar a mensagem retornada e não depender exclusivamente de `409`;
- preservar busca por descendentes nos filtros de MRP, Compras e Custos.

## 3. Calendário industrial — IMPLEMENTADO

`POST /api/industrial-calendar/generate`:

```json
{"year": 2027, "month": null, "weekdays": [1, 2, 3, 4, 5]}
```

Compatibilidade: `POST /api/industrial-calendar/generate/{year}/{month}`.

As duas rotas de geração estão montadas no router público e protegidas pelos
papéis `ADMIN` e `USER`, como as rotas existentes do calendário.

- segunda a sexta são úteis por padrão;
- não sobrescrever dias existentes.
- resposta: `year`, `month`, `created`, `preserved`, `ignored`.
- origem em português: `AUTOMATICO`, `FIM_DE_SEMANA`, `FERIADO`, `MANUAL`.
- calendário isolado por empresa.

## 4. Exportação PDF/VITM0100 — IMPLEMENTADO

`POST /api/reports/export?format=pdf` aceita:

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

Um item interno aceita vários fornecedores, cada qual com código, descrição,
unidade, conversão, preferência e vigência próprios. Há resolução por código
ou descrição externa e rastreabilidade na entrada fiscal.

Busca: `GET /api/item-suppliers/search?supplier_code=123&term=ABC`. O retorno
prioriza o código externo exato e depois a descrição. Estratégias fiscais:
`CODIGO_EXATO`, `DESCRICAO`, `MANUAL` e `NAO_RESOLVIDO`. A linha fiscal retorna
`item_supplier_id`, `supplier_item_code`, `resolution_strategy` e `resolved_at`.

## 6. Parametrização fiscal automática — IMPLEMENTADO

O mestre fiscal é isolado por empresa e possui vigência e padrões de NCM, CEST,
origem, UMs, IPI, ICMS e PIS/COFINS. Respostas do item incluem
`fiscal_effective.purchase` e `fiscal_effective.sale`, a referência ao mestre e o
mapa `sources`, com `HERDADO` ou `SOBRESCRITO`.

Exemplo de criação:

`POST /api/fiscal-classifications`

```json
{"description":"COMPONENTES METALICOS","ncm":"73269090","ipi_rate":5,"pis_rate":1.65,"cofins_rate":7.6,"default_origin":"0","default_icms_rate":18,"default_calculate_pis_cofins":true}
```

## 7. Código de barras da ordem de produção — IMPLEMENTADO NO BACKEND

Criar token (ADMIN): `POST /api/production-order/scanner/tokens`:

```json
{"production_order_id":123,"operation_id":456,"valid_until":"2027-01-31T23:59:59-03:00"}
```

Codifique `barcode_value` em Code 128 ou QR. Nunca use os IDs no lugar do token.
O layout backend `manufacturing.ManufacturingOrder` também aceita esse valor e
gera Code 128 vetorial no PDF, com legenda mascarada e sem revelar o token em texto.

Leitura: `POST /api/production-order/scanner/scan`:

```json
{"token":"OF1_...","action":"APONTAR","idempotency_key":"coletor-01-00042","device_id":"coletor-01","employee_id":77,"good_quantity":"12.500","scrap_quantity":"0.250","hours":"1.750","scrap_reason":"TRINCA","complete_operation":true}
```

Ações: `RESOLVER`, `INICIAR`, `APONTAR`, `CONCLUIR`. Quantidades e horas são
strings decimais. O backend valida tenant, vigência, operador, sequência e status,
e registra idempotência/auditoria persistentes.

Resposta de criação do token: `token`, `barcode_value`, `production_order_id`,
`operation_id` e `valid_until`. Resposta da leitura: `production_order_id`,
`operation_id`, `order_number`, `status`, `operation_status` e `replayed`.

## 8. Auditoria de campos duplicados — DECISÃO DE GOVERNANÇA

Não remover campos no frontend até classificá-los como fonte da verdade, derivado,
override ou conceito distinto. Prioridades: descrição, unidade, peso, classificação,
comprado/fabricado, mínimos de estoque e códigos externos de fornecedores.

- Nome, descrição técnica PDM e descrição comercial são conceitos distintos.
- UM de estoque é base; UMs de compra/venda são contextuais e usam conversão.
- Peso de engenharia é mestre; pesos fiscais/logísticos são transacionais.
- Estoque mínimo, segurança e ponto de reposição não são duplicados.
- Classificações comercial, contábil e fiscal têm finalidades diferentes.

## 9. Ordem das telas de busca — RESPONSABILIDADE DO FRONTEND

Ao abrir uma rotina que possua formulário de busca e formulário de manutenção,
mostrar primeiro a busca/listagem. O cadastro/edição deve ser aberto por ação
explícita (`Novo`, `Editar` ou seleção de resultado). O backend não controla ordem
visual de forms; nenhuma rota deve ser chamada para criar registros apenas ao abrir
a tela.

## 10. Auto-preenchimento de fornecedor por CNPJ — BACKEND DISPONÍVEL

Antes do `POST /api/suppliers`, chamar `GET /api/cnpj/{cnpj}`. A resposta compartilhada
com clientes contém razão social, nome fantasia, inscrição estadual, endereço, CNAE,
porte, Simples e MEI. CNPJ inválido retorna `400`, inexistente `404` e indisponibilidade
do provedor `502`. Após o fornecedor existir, a consulta cadastral fiscal permanece em
`POST /api/suppliers/{code}/sefaz-query`.

## 11. Laudos e certificados do fornecedor — IMPLEMENTADO NO BACKEND

- Criar/anexar: `POST /api/item-suppliers/{id}/quality-reports`;
- listar: `GET /api/item-suppliers/{id}/quality-reports`;
- baixar: `GET /api/item-suppliers/quality-reports/{reportID}/download`;
- status em português: `PENDENTE`, `APROVADO`, `REJEITADO`, `EXPIRADO`;
- limite atual do anexo: 10 MiB;
- `content` é enviado em Base64 pelo JSON (mapeamento padrão de `[]byte` do Go);
- isolamento por empresa aplicado no vínculo, listagem e download.

A associação explícita com a inspeção de recebimento está disponível em:

- `POST /api/procurement/receiving-inspection-orders/{id}/quality-reports`, corpo
  `{"quality_report_id": 123}`;
- `GET /api/procurement/receiving-inspection-orders/{id}/quality-reports`;
- `DELETE /api/procurement/receiving-inspection-orders/{id}/quality-reports/{reportID}`.

O backend só aceita o vínculo quando empresa, item e fornecedor do laudo são
compatíveis com a ordem de inspeção. Vínculos repetidos são idempotentes e ficam
auditados por usuário e data. O download continua sendo feito pelo endpoint do
laudo, usando o `quality_report_id` retornado.

### Checklist obrigatório do frontend

- nunca converter códigos de item para número nem usar `legacy_code` na UI;
- abrir rotinas de consulta na busca/listagem, não no formulário de inclusão;
- usar somente os enums em português documentados neste arquivo;
- gerar Code 128 ou QR usando exatamente `barcode_value`;
- consultar CNPJ antes do cadastro e permitir revisão humana antes de salvar;
- no VSUP0670, anexar o laudo ao vínculo item-fornecedor e depois associar o
  `quality_report_id` à ordem de inspeção;
- mostrar status, nome, data e responsável de cada laudo vinculado;
- tratar `400`, `401/403`, `404`, `409` e `422` sem ocultar a mensagem do backend.

## 12. Herança do indicador PIS/COFINS — IMPLEMENTADO NO BACKEND

No `POST /api/items`, `accounting.calculate_pis_cofins` possui três estados:

- campo ausente: não grava override e herda da classificação fiscal;
- `true`: grava sobrescrita afirmativa no item;
- `false`: grava sobrescrita negativa no item.

Quando herdado, `accounting.calculate_pis_cofins` não é enviado na resposta bruta
do item. O valor resolvido fica em
`fiscal_effective.purchase.calculate_pis_cofins` e
`fiscal_effective.sale.calculate_pis_cofins`, enquanto
`sources.calculate_pis_cofins` retorna `HERDADO`. Nos dois valores explícitos, a
fonte retorna `SOBRESCRITO`. Portanto, o frontend não deve preencher `false` por
padrão: deve omitir a propriedade enquanto o usuário não escolher sobrescrever.

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
