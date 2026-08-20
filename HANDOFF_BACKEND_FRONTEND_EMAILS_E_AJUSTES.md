# VentureERP — handoff Backend/Desktop: e-mails, alertas e ajustes operacionais

> Estado do trabalho em 16/08/2026, no worktree `panossoerp-ajustes`.
>
> Este documento é o contrato de integração para o Desktop. Ele consolida o que foi
> criado no backend, o comportamento esperado, as migrations necessárias, os testes
> realizados e o que ainda precisa ser implementado ou decidido no frontend.

## 1. Escopo entregue

Foram implementadas duas frentes relacionadas:

1. uma central enterprise de alertas internos e e-mails automáticos, isolada por
   empresa, com outbox transacional, worker, retry, resumo diário, histórico,
   dead letter, templates profissionais e administração por tenant;
2. correções encontradas na varredura das 211 telas do Desktop: plano de corte sem
   configuração, referências PDM, validações HTTP, isolamento de orçamento, criação
   de item, unidade de medida e consultas operacionais do worker.

Não há envio automático para clientes ou fornecedores. Destinatários automáticos são
somente usuários internos, papéis ou departamentos associados à empresa autenticada.
Qualquer envio externo deve continuar sendo uma ação manual, explícita e auditada.

## 2. Migrations e ordem de instalação

Aplicar rigorosamente nesta ordem:

| Migration | Finalidade |
|---|---|
| `000308_enterprise_notifications` | catálogo, settings, assinaturas, outbox, alertas, entregas, tentativas, dead letters, departamentos e contagem cíclica |
| `000309_notification_fiscal_stock_producers` | produtores fiscais e operacionais de estoque |
| `000310_notification_pending_daily_reminders` | lembretes diários de pendências enquanto a causa continuar aberta |
| `000311_desktop_integrity_tenant_fixes` | isolamento de configurações de corte e referências PDM por empresa |
| `000312_notification_runtime_query_repairs` | corrige o trigger de item para `engineering_item_base_code` |
| `000313_notification_catalog_producer_status` | expõe destinatários elegíveis, status real dos produtores e consolida o contrato textual da contagem |
| `000314_cycle_count_policy_scheduler` | materializa a política permanente do item em ocorrências de contagem idempotentes e auditadas |

Não modificar migrations já aplicadas. Qualquer evolução posterior deve receber novo
número. Os arquivos `down` existem para validação técnica; não fazer rollback da
central em uma base com histórico real sem política explícita de retenção/exportação.

## 3. Arquitetura da central de alertas

### 3.1 Fluxo

```text
operação de negócio
    └─ grava evento na notification_outbox na mesma transação
          └─ worker faz claim com FOR UPDATE SKIP LOCKED
                ├─ cria/atualiza alerta e resolve destinatários internos
                ├─ gera entrega com Message-ID idempotente
                ├─ envia MIME HTML + texto pelo SMTP central
                └─ registra sucesso, tentativa, retry ou dead letter
```

Falha no e-mail nunca desfaz venda, nota, estoque, cadastro ou outra operação já
confirmada. O worker suporta múltiplas réplicas, lease recuperável e shutdown com
drain. A entrega é pelo menos uma vez, com deduplicação lógica e Message-ID estável.

### 3.2 Evento e pendência

- `EVENTO`: fato pontual, enviado uma vez. Exemplo: orçamento convertido em pedido.
- `PENDENCIA`: condição que permanece aberta. Exemplo: estoque abaixo do mínimo.
  Pode gerar o primeiro aviso e um lembrete diário por destinatário até a causa ser
  corrigida. Quando normalizada, é marcada `RESOLVIDO`; nova violação abre novo ciclo.

Cadências públicas:

- `IMEDIATO`;
- `RESUMO_DIARIO`;
- `IMEDIATO_E_RESUMO_DIARIO`.

Estados relevantes:

- alerta: `ABERTO`, `RESOLVIDO`, `IGNORADO`;
- entrega: `PENDENTE`, `PROCESSANDO`, `ENVIADO`, `FALHOU`, `DESCARTADO`, `CANCELADO`;
- severidade: `INFORMATIVO`, `ATENCAO`, `CRITICO`.

### 3.3 SMTP e segurança

O provedor é central e configurado somente no ambiente do backend. O frontend nunca
deve solicitar, armazenar ou exibir usuário, senha ou token SMTP.

Configuração operacional usada pelo backend:

- `SMTP_HOST`;
- `SMTP_PORT`;
- `SMTP_USERNAME`;
- `SMTP_PASSWORD`;
- `SMTP_FROM`;
- `SMTP_FROM_NAME`.

Porta 465 usa TLS implícito; as demais usam STARTTLS. TLS 1.2 ou superior é exigido.
Credenciais, MIME completo e payload sensível não são registrados em logs nem
retornados pelas APIs.

### 3.4 Conteúdo e identidade visual

Os e-mails são horizontais, responsivos e usam identidade verde do VentureERP, com:

- cabeçalho de empresa, módulo e severidade;
- resumo executivo e recomendação operacional;
- blocos de indicadores e tabelas com dados úteis;
- unidade de estoque/compra, item, máscara, classe ABC e criticidade quando aplicável;
- almoxarifado, saldos, mínimo, segurança, cobertura e última movimentação;
- fornecedor recomendado, lead time, embalagem e compras em aberto;
- alternativa em texto puro e rodapé automático.

Como o VentureERP é Desktop, o e-mail não depende de botão web. Quando aplicável,
informa módulo/tela e identificadores para localização autenticada dentro do app.
Conteúdo dinâmico é escapado e cabeçalhos rejeitam CR/LF.

Branding usa a configuração fiscal vinculada em `fiscal_config_id`. Sem logo ou
identidade configurada, utiliza fallback profissional VentureERP.

## 4. API que o Desktop deve consumir

Todas as rotas exigem JWT. O tenant vem do token/contexto; não enviar ou confiar em
`enterprise_id` para selecionar empresa.

### 4.1 Administração de notificações

| Método | Rota | Papel | Uso no Desktop |
|---|---|---|---|
| `GET` | `/api/notifications/events` | ADMIN, USER | catálogo de eventos, módulos, severidade e cadências permitidas |
| `GET` | `/api/notifications/recipients/users` | ADMIN, USER | usuários vinculados ao tenant; retorna somente `id`, `name`, `role`, `active` |
| `GET` | `/api/notifications/recipients/departments` | ADMIN, USER | departamentos do tenant; retorna `code`, `description`, `active` |
| `GET` | `/api/notifications/settings` | ADMIN | carregar configuração da empresa |
| `PUT` | `/api/notifications/settings` | ADMIN | salvar horário, fuso, retenção, limites e branding |
| `GET` | `/api/notifications/subscriptions` | ADMIN | listar assinaturas |
| `POST` | `/api/notifications/subscriptions` | ADMIN | criar assinatura e destinatários |
| `PUT` | `/api/notifications/subscriptions/{id}` | ADMIN | alterar assinatura |
| `DELETE` | `/api/notifications/subscriptions/{id}` | ADMIN | excluir/desativar assinatura |
| `POST` | `/api/notifications/test-email` | ADMIN | enfileirar teste; retorna `202`, não significa entrega imediata |
| `GET` | `/api/notifications/deliveries?limit=&offset=` | ADMIN | histórico paginado |
| `POST` | `/api/notifications/deliveries/{id}/retry` | ADMIN | reenvio auditado de falha/descartado |
| `GET` | `/api/notifications/dead-letters?limit=&offset=` | ADMIN | falhas esgotadas |
| `GET` | `/api/notifications/alerts?limit=&offset=` | ADMIN, USER | alertas visíveis do tenant |
| `GET` | `/api/notifications/alerts/{id}` | ADMIN, USER | detalhe do alerta |

Exemplo de settings:

```json
{
  "enabled": true,
  "digest_time": "08:00",
  "timezone": "America/Sao_Paulo",
  "retention_days": 365,
  "max_attachment_bytes": 10485760,
  "max_emails_per_minute": 60,
  "fiscal_config_id": 1
}
```

Exemplo de assinatura:

```json
{
  "event_key": "COMERCIAL_ORCAMENTO_CONVERTIDO_PEDIDO",
  "event_version": 1,
  "enabled": true,
  "cadence": "IMEDIATO_E_RESUMO_DIARIO",
  "thresholds": {},
  "recipients": [
    {"recipient_type": "USUARIO", "user_id": "<uuid-interno>"},
    {"recipient_type": "PAPEL", "recipient_key": "ADMIN"}
  ]
}
```

O frontend não deve oferecer campo de e-mail livre. Para `USUARIO`, selecionar
usuário ativo da empresa; para `PAPEL`, usar papel válido; para `DEPARTAMENTO`, usar
departamento interno cadastrado.

DTOs de destinatários elegíveis:

```json
{"id":"<uuid>","name":"Nome para exibição","role":"USER","active":true}
{"code":"COMERCIAL","description":"Comercial","active":true}
```

Usuários e departamentos inativos permanecem identificáveis por `active: false`, mas
não podem ser usados em uma nova assinatura. Nenhuma dessas rotas retorna e-mail,
hash de senha, token ou credencial.

Cada evento do catálogo agora contém `producer_status` (`ATIVO` ou `FUTURO`) e
`producer_description`. O Desktop deve permitir configuração normal de `ATIVO` e
desabilitar ou destacar claramente `FUTURO`, explicando que ainda não existe produtor
operacional conectado.

### 4.2 Contagem cíclica

| Método | Rota | Uso |
|---|---|---|
| `POST` | `/api/stock/cycle-counts` | programar contagem por empresa/almoxarifado/item/máscara/lote; `item_code` é string comercial |
| `GET` | `/api/stock/cycle-counts` | consultar contagens |
| `GET` | `/api/stock/cycle-counts/{id}` | detalhe e auditoria |
| `POST` | `/api/stock/cycle-counts/{id}/transition` | iniciar, concluir, aprovar ou alterar estado |

Aprovação exige `ADMIN`. Quantidades devem ser tratadas como decimal, nunca `float`
arredondado pela interface. Datas operacionais vêm em UTC e devem ser exibidas no
fuso da empresa.

O contrato público de contagem sempre usa código comercial textual:

```json
{"warehouse_id":1,"item_code":"0007-A","scheduled_for":"2026-08-20T12:00:00Z"}
```

Criação, listagem e detalhe devolvem `item_code` exatamente como cadastrado. A chave
interna aparece apenas como `legacy_item_code` durante a compatibilidade e não deve
ser apresentada ou reenviada pelo Desktop.

Toda ocorrência devolve `origin`. Programação manual retorna `"origin":"MANUAL"`
e omite `policy_days`; a automática retorna, por exemplo,
`"origin":"POLITICA_ITEM","policy_days":30`. Esses campos são preservados em
criação, listagem e detalhe.

O bloco `warehouse.cyclical_count_config` do cadastro do item é uma política
permanente, não uma ocorrência. O contrato oficial de entrada e saída é:

```json
{"warehouse":{"cyclical_count_config":{"days_interval":30}}}
```

Zero, negativos e chaves não documentadas, como `days`, retornam `422`. A grafia
legada `DaysInterval` é aceita temporariamente somente na entrada; toda resposta é
normalizada para `days_interval`. Quando o intervalo está ativo, o worker cria
automaticamente a próxima contagem no `warehouse_code` do item. A primeira data é
`ativação da política + intervalo`; depois de uma aprovação, é
`approved_at + intervalo`. Se a data calculada já passou, a ocorrência é criada
vencida para não esconder o atraso.

Existe no máximo uma contagem não encerrada no escopo item/almoxarifado sem
endereço/máscara/lote. Execuções concorrentes são protegidas por lock transacional e
índice único. Alterar intervalo ou almoxarifado, ou desativar a política, cancela
somente ocorrências automáticas ainda `PROGRAMADA`; contagens iniciadas e todo o
histórico permanecem intactos. O Desktop deve exibir `POLITICA_ITEM` como
"Automática — política do item" e `MANUAL` como "Manual"; não deve tentar criar a
ocorrência automática por conta própria.

## 5. Eventos já conectados a produtores reais

- orçamento convertido em pedido;
- item configurado criado sem perguntas/características ativas;
- exceções do MRP;
- NF-e de saída criada, aguardando autorização, autorizada, rejeitada e cancelada;
- NF-e de entrada importada, aprovada, cancelada, com item não identificado ou
  divergência fiscal/quantitativa;
- contagem próxima, vencida, divergente, concluída e aprovada;
- estoque abaixo do mínimo e negativo;
- lote próximo do vencimento;
- movimentação incomum conforme limiares configurados.

O catálogo contém eventos adicionais de Comercial, Compras, Produção, APS,
Manutenção, Qualidade, Financeiro, Segurança e Operação. Estar no catálogo significa
que pode ser configurado; não significa que todos os produtores futuros já foram
ligados ao respectivo processo de negócio.

## 6. Tela administrativa que o frontend precisa criar

Recomendação: uma rotina “Central de Alertas” com quatro abas.

### Configuração

- habilitar/desabilitar central para a empresa;
- horário do resumo diário;
- timezone IANA, com `America/Sao_Paulo` como sugestão;
- retenção e limites permitidos pela API;
- configuração fiscal/branding selecionável;
- botão “Enviar e-mail de teste”, exibindo “enfileirado” após `202` e orientando o
  usuário a acompanhar o histórico.

### Assinaturas

- agrupar catálogo por módulo e severidade;
- mostrar descrição em português e distinguir evento de pendência;
- permitir somente cadências declaradas no catálogo;
- configurar thresholds específicos do evento;
- selecionar usuários, papéis e departamentos internos;
- exigir confirmação antes de ativar uma assinatura;
- nunca permitir endereço externo digitado manualmente.

### Histórico

- paginação por `limit`/`offset`;
- estado, destinatário mascarado quando necessário, evento, tentativas e horários;
- não exibir payload/MIME integral nem erros com dados sensíveis;
- botão de reenvio somente para estados autorizados pela API;
- atualizar a linha depois do retry, sem apagar a tentativa anterior.

### Dead letters

- explicar que a entrega esgotou as tentativas;
- exibir erro sanitizado e tentativa mais recente;
- permitir reenvio auditado;
- não oferecer “excluir histórico”.

## 7. Correções de integração descobertas na varredura Desktop

### 7.1 Plano de corte — `VCUT0100`

`GET /api/cutting-settings` agora é tenant-aware. Empresa sem configuração recebe
`200` com padrões tratáveis, incluindo `default_consumption_mode: "AUTOMATIC"`.
O frontend não deve interpretar ausência de registro como falha nem exigir criação
prévia. Ao salvar, o upsert afeta apenas o tenant autenticado.

### 7.2 Cadastro de item

- grupo e modificador PDM são validados dentro da empresa autenticada;
- grupo/modificador inexistente ou pertencente a outro tenant retorna `422`;
- nenhuma parte do item é persistida quando essa validação falha;
- unidade de medida inexistente retorna `422` com JSON compreensível;
- item alfanumérico válido retorna `201` e `data.code` continua sendo string;
- não converter o código comercial para número nem remover zeros à esquerda;
- `PUT /api/items/{code}` e `GET /api/items/{code}/activation-readiness` resolvem
  nativamente `(enterprise_id, business_code)` no use case/repositório. O Desktop
  deve enviar sempre o código comercial textual, inclusive `234139`, `0007` e
  `TEA452-0`; IDs legados não fazem parte dessas rotas públicas;
- o `PUT` é parcial: blocos e campos omitidos são mesclados com o item persistido e
  não devem ser reenviados pelo Desktop. Para os value objects JSONB opcionais
  `engineering_dimensions`, `planning_reorder_point` e
  `warehouse_cyclical_count_config`, registros históricos com SQL `NULL`, JSON
  `null` ou `{}` significam ausência. Um objeto não vazio continua sujeito à
  validação de domínio e não é descartado silenciosamente;
- usar `engineering_item_base_code`; o nome legado `engineering_item_base_cod` não
  deve aparecer em payloads ou integrações novas.

### 7.3 Funcionário e prioridade

O frontend deve exibir a mensagem retornada em `error`, sem trocar por mensagem
genérica:

- funcionário com código zero/negativo ou nome vazio: `422`;
- funcionário duplicado: `409`;
- prioridade com início maior/igual ao fim: `422`;
- faixa de prioridade sobreposta: `409`.

`500` fica reservado para falhas inesperadas.

### 7.4 Orçamento e tenant

A regra escolhida é compatibilidade com clientes instalados: o middleware substitui
qualquer `enterprise_code` do JSON pelo código da empresa autenticada. Portanto uma
requisição pode retornar `201`, mas nunca grava em outra empresa.

### 7.5 Rotas operacionais corrigidas após a varredura

- `/api/items/classifications/masks/` é uma rota estática e não passa pela
  resolução de código comercial de item;
- criação de tipo de máquina e máquina usa `enterprise_id` e `created_by` do
  contexto autenticado; esses campos do payload não são fonte de verdade;
- criação de plano de corte gera o código dentro do tenant e persiste o
  `enterprise_id` autenticado no plano e em todos os registros filhos: peças,
  estoque de corte, padrões, posicionamentos, retalhos e consumos;
- registro de lote aceita `mask` e sua identidade é
  `(enterprise_id, item_code, mask, lot)`;
- reserva de estoque exige saldo disponível real. Falta de disponibilidade
  retorna `422` e não deixa reserva parcial nem cria saldo artificial.

O frontend deve preferencialmente omitir `enterprise_code`; quando mantido por
compatibilidade, não deve permitir que o usuário o edite. Não interpretar `201` como
aceitação de troca de tenant.

Validações de configuração do orçamento agora retornam `422` para rótulos vazios,
comissão desbalanceada, motivo vazio ou motivo de cancelamento inexistente.

## 8. Pedidos funcionais levantados e situação atual do backend

### Previsão de vendas `VPRE0201`

O backend atual registra previsão por item/máscara e período, cria previsão mensal,
distribui por semanas e gera por histórico de pedidos/faturamento. Ainda não existe
um contrato aprovado para “tipo” livre (`Radiadores`, `Forma`, `Tanque`) nem geração
por curva ABC.

Antes de implementar, definir se “tipo” será:

1. cadastro livre por empresa associado aos itens; ou
2. reutilização de linha de produto/classificação já existente.

O frontend não deve criar uma lista fixa local. Curva ABC também deve vir do backend
e respeitar empresa, período e critério de classificação.

### Condições de pagamento de vendas

O backend já permite cadastro livre de condição e qualquer quantidade de parcelas.
Exemplos suportáveis: `30/60/90`, `21/42/63`, `14`, `7` e entrada/antecipado mais
parcelas. A tela precisa permitir cadastrar a condição pai e suas parcelas com:

- número sequencial;
- dias para vencimento;
- descrição;
- tipo de documento;
- movimento e portador opcionais.

Não limitar a seleção a uma lista fixa codificada no Desktop.

### Status do orçamento

O orçamento já possui status e endpoint de mudança, porém os códigos atuais são
técnicos. Ainda é necessário aprovar o contrato para estados explícitos como
`AGUARDANDO_APROVACAO_CLIENTE` e `APROVADO_AGUARDANDO_MATERIA_PRIMA`, incluindo
transições permitidas e efeito sobre conversão, MRP e expiração.

### DIFAL

O motor calcula DIFAL apenas em venda interestadual para destinatário não
contribuinte ou pessoa física:

```text
DIFAL = (alíquota interna da UF destino − alíquota interestadual) × base de ICMS
FCP   = alíquota FCP da UF destino × base de ICMS
```

Destinatário contribuinte interestadual não gera DIFAL. Produto importado nas origens
`3`, `4`, `5` ou `8` usa alíquota interestadual de 4%. A tela de conferência deve
mostrar base, alíquotas, DIFAL e FCP por item e totais, sem recalcular no frontend.

### Prévia da nota

`POST /api/fiscal/exits/create` já cria a NF-e em `RASCUNHO` e calcula ICMS, IPI,
PIS, COFINS, DIFAL e FCP antes de qualquer transmissão. A autorização é operação
separada. O Desktop deve tratar o rascunho como prévia fiscal, permitir conferência e
somente chamar `/api/fiscal/exits/{code}/authorize` após confirmação explícita.

### Financeiro manual na entrada de nota

O pedido exige alterar parcelas geradas pela entrada antes da confirmação financeira,
por exemplo transformar uma parcela da NF em três boletos. Esse fluxo ainda precisa
de task própria definindo soma obrigatória, vencimentos, auditoria, permissões e ponto
exato de confirmação. O frontend não deve editar títulos confirmados diretamente.

### Gráficos de fluxo de caixa

O backend já entrega fluxo realizado e projetado. Gráficos são responsabilidade do
Desktop: agregar por dia/semana/mês, separar entrada/saída e apresentar saldo
acumulado. Valores retornados são a fonte de verdade; não persistir os pontos do
gráfico nem usar `float` para somas monetárias.

### Adiantamentos de clientes e fornecedores

Já existem:

- `POST /api/financial/adiantamentos/create`;
- `GET /api/financial/adiantamentos/list` com filtros `tipo` e `parceiro_id`;
- `GET /api/financial/adiantamentos/{id}`;
- `POST /api/financial/adiantamentos/{id}/aplicar`.

`RECEBER` representa crédito/entrada antecipada de cliente; `PAGAR` representa
débito/saída antecipada para fornecedor. Criar movimenta o caixa. Aplicar contra um
título não movimenta caixa novamente: apenas consome o saldo e baixa o título.
O relatório de adiantamentos em aberto pode usar a listagem filtrando saldo/status.

O Desktop precisa oferecer cadastro, saldo disponível, histórico de aplicações,
seleção do título compatível e relatório “aguardando nota”. Nunca gerar um segundo
movimento financeiro ao aplicar o adiantamento.

## 9. Tratamento HTTP obrigatório no frontend

| Status | Tratamento |
|---|---|
| `200` | leitura/alteração concluída |
| `201` | recurso criado |
| `202` | operação assíncrona apenas enfileirada |
| `204` | concluída sem body |
| `400` | JSON/formato inválido |
| `401` | sessão/tenant inválido; solicitar autenticação |
| `403` | usuário sem permissão |
| `404` | recurso inexistente ou estado opcional não configurado |
| `409` | conflito, duplicidade ou sobreposição |
| `422` | regra de domínio/referência inválida; mostrar `error` ao usuário |
| `500` | falha inesperada; registrar correlation ID e não ocultar recorrência |

Não repetir automaticamente requisições `POST` de negócio sem idempotência. No teste
de e-mail, `202` significa apenas que entrou na fila; o resultado final aparece no
histórico.

Nas rotas de notificação, recurso ausente no tenant retorna `404`, conflito de
estado/retry/transição retorna `409` e settings, threshold, destinatário, referência
ou quantidade inválida retornam `422`. JSON malformado continua `400`.

## 10. Testes e evidências

- suíte completa `go test ./...`: aprovada;
- `go vet ./...`: aprovado;
- `git diff --check`: aprovado;
- testes HTTP para corte, PDM, funcionário, prioridade, orçamento, cancelamento,
  unidade de medida e código alfanumérico;
- teste PostgreSQL do scheduler de alertas operacionais;
- criação real e transacional de item configurado após a `000312`, incluindo outbox;
- migrations `000311` e `000312` validadas em ciclo `down -> up` no treinamento;
- migration `000314` validada em ciclo `down -> up`, com primeira programação,
  recorrência, atraso, desativação, idempotência e isolamento multiempresa;
- teste HTTP/PostgreSQL sem cache cobre `4853`, `0007` e `TEA452-0` atravessando os
  middlewares de compatibilidade das rotas de item;
- isolamento de configuração de corte comprovado com segundo tenant em transação;
- dez e-mails profissionais de demonstração enviados ao endereço externo indicado
  pelo responsável exclusivamente como teste manual autorizado. Esse endereço não
  foi cadastrado como destinatário automático.

## 11. Checklist de entrega do Desktop

- [ ] criar Central de Alertas com Configuração, Assinaturas, Histórico e Dead letters;
- [ ] consumir catálogo e cadências do backend, sem enums locais divergentes;
- [ ] selecionar somente destinatários internos do tenant;
- [ ] tratar teste de e-mail como assíncrono (`202`);
- [ ] implementar telas de contagem cíclica e transições conforme permissão;
- [ ] aceitar defaults `AUTOMATIC` no plano de corte;
- [ ] preservar código comercial de item como string;
- [ ] exibir mensagens `409/422` do backend;
- [ ] omitir/bloquear edição de `enterprise_code` em orçamento;
- [ ] liberar cadastro flexível de condições e parcelas;
- [ ] usar rascunho de NF-e como prévia tributária;
- [ ] criar gráficos a partir do fluxo real/projetado;
- [ ] integrar cadastro, aplicação e relatório de adiantamentos;
- [ ] definir com Produto/Negócio a origem do “tipo” da previsão e os novos status;
- [ ] especificar edição financeira da entrada de NF antes de implementar.

## 12. Pendências e limites

- o código das telas Desktop não está neste repositório backend;
- previsão por tipo/ABC, novos status semânticos de orçamento e edição manual das
  parcelas de entrada ainda precisam de task funcional aprovada;
- eventos apenas catalogados precisam de produtores próprios antes de enviarem;
- nenhuma credencial SMTP deve ser copiada para documentação, frontend ou commit;
- publicação, restart de API, commit, push e release são operações separadas deste
  handoff.

Documentação técnica complementar: `docs/dev/notificacoes-automaticas.md`.
