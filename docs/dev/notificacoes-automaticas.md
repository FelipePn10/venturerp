# Central de alertas e e-mails automáticos

## Segurança e arquitetura

A central é exclusivamente interna e isolada pelo `enterprise_id` do JWT. Nenhuma
API aceita endereço livre: destinatários são usuários ativos associados à empresa,
papéis da associação `user_enterprises` ou departamentos internos. E-mails a
clientes/fornecedores continuam em fluxo manual, explícito e auditado.

Os casos de negócio gravam `notification_outbox` na própria transação. O worker faz
claim concorrente com `FOR UPDATE SKIP LOCKED`, lease recuperável, deduplicação por
empresa/evento/chave, entregas pelo menos uma vez e `Message-ID` estável. As esperas
de retry são 1, 5, 15 e 60 minutos, 6 e 24 horas; a sexta falha gera dead letter.
Falha do provedor nunca desfaz uma operação já confirmada. Logs e APIs exibem somente
códigos e erros sanitizados, nunca MIME, payload integral, credenciais ou anexos.

SMTP é configuração central da aplicação. TLS 1.2+ é obrigatório, com TLS implícito
na porta 465 ou STARTTLS nas demais. O conteúdo tem HTML responsivo e alternativa
texto; valores dinâmicos passam por escape de template e cabeçalhos rejeitam CR/LF.
Como o VentureERP é um aplicativo desktop, os e-mails não dependem de botões ou
deep links web: informam o caminho do módulo/tela para consulta autenticada no app.
Branding usa o
`fiscal_config_id` explicitamente vinculado aos settings da empresa; uma mesma
configuração fiscal não pode ser vinculada a dois tenants. Sem vínculo, aplica-se o
nome da empresa e a identidade visual profissional padrão.

## Enums públicos

| Conceito | Valores |
|---|---|
| severidade | `INFORMATIVO`, `ATENCAO`, `CRITICO` |
| cadência | `IMEDIATO`, `RESUMO_DIARIO`, `IMEDIATO_E_RESUMO_DIARIO` |
| alerta | `ABERTO`, `RESOLVIDO`, `IGNORADO` |
| entrega | `PENDENTE`, `PROCESSANDO`, `ENVIADO`, `FALHOU`, `DESCARTADO`, `CANCELADO` |
| destinatário | `USUARIO`, `PAPEL`, `DEPARTAMENTO` |
| canal | `EMAIL`, `NOTIFICACAO_INTERNA`, `WEBHOOK` |

## API e permissões

Todas as rotas exigem JWT e usam implicitamente a empresa selecionada. As rotas de
configuração/histórico são `ADMIN`; catálogo e alertas aceitam `ADMIN` e `USER`.

| Método e rota | Papel |
|---|---|
| `GET /api/notifications/events` | `ADMIN`, `USER` |
| `GET`, `PUT /api/notifications/settings` | `ADMIN` |
| `GET`, `POST /api/notifications/subscriptions` | `ADMIN` |
| `PUT`, `DELETE /api/notifications/subscriptions/{id}` | `ADMIN` |
| `POST /api/notifications/test-email` | `ADMIN` |
| `GET /api/notifications/deliveries?limit=50&offset=0` | `ADMIN` |
| `POST /api/notifications/deliveries/{id}/retry` | `ADMIN` |
| `GET /api/notifications/dead-letters?limit=50&offset=0` | `ADMIN` |
| `GET /api/notifications/alerts?limit=50&offset=0` | `ADMIN`, `USER` |
| `GET /api/notifications/alerts/{id}` | `ADMIN`, `USER` |
| `POST /api/stock/cycle-counts` | `ADMIN`, `USER` autenticado |
| `GET /api/stock/cycle-counts` | `ADMIN`, `USER` autenticado |
| `GET /api/stock/cycle-counts/{id}` | `ADMIN`, `USER` autenticado |
| `POST /api/stock/cycle-counts/{id}/transition` | autenticado; aprovação somente `ADMIN` |

Exemplo de settings (`snake_case`):

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

Exemplo de assinatura por usuários e papel:

```json
{
  "event_key": "COMERCIAL_ORCAMENTO_CONVERTIDO_PEDIDO",
  "event_version": 1,
  "enabled": true,
  "cadence": "IMEDIATO_E_RESUMO_DIARIO",
  "thresholds": {},
  "recipients": [
    {"recipient_type": "USUARIO", "user_id": "c56a4180-65aa-42ec-a945-5fd21dec0538"},
    {"recipient_type": "PAPEL", "recipient_key": "ADMIN"}
  ]
}
```

O backend ignora qualquer `enterprise_id` recebido no corpo e usa o tenant do JWT.
Usuário inexistente, inativo ou fora da empresa é rejeitado. Reenvio preserva o
histórico de tentativas e marca a dead letter original como reenviada.

## Catálogo e gatilhos

A migration `000308` contém a matriz versionada completa para Comercial, Estoque,
Fiscal, Cadastro, Compras, Produção/MRP/APS/Manutenção, Qualidade, Financeiro,
Segurança e Operação. O catálogo é ativável por empresa; nenhuma assinatura é criada
sem confirmação do administrador.

Gatilhos transacionais entregues nesta onda:

- `COMERCIAL_ORCAMENTO_CONVERTIDO_PEDIDO`: gravado exatamente uma vez na transação
  de conversão, com pedido, orçamento, cliente, representante, datas, moeda,
  pagamento, transportadora, totais decimais, itens, observações e caminho no desktop;
- `CADASTRO_ITEM_CONFIGURADO_SEM_PERGUNTAS`: abre após item de natureza configurada
  sem característica ativa, cancela pendência ainda não processada ou resolve o
  alerta ao associar a primeira válida e abre novo ciclo auditável se todas forem
  removidas novamente;
- `MRP_EXCECAO`: a rota legada `/api/mrp-calculation/exceptions/notify` permanece e
  passa a publicar na central quando e-mail é solicitado. Os endereços informados no
  payload legado não são usados pela automação; valem os destinatários da assinatura.

A migration `000309` liga produtores persistidos das transições de NF-e de saída e
entrada, incluindo rejeição, cancelamento, item não identificado e divergências
fiscal/quantitativa. O scheduler liga os nove eventos de estoque: contagens próximas,
vencidas, divergentes, concluídas e aprovadas, saldo abaixo do mínimo/negativo, lote
próximo do vencimento e movimentação incomum por regras explícitas. Os eventos dos
demais módulos permanecem ativáveis no catálogo para evolução incremental dos seus
produtores, sem acoplar SMTP aos casos de uso.

## Contagem cíclica

`stock_cycle_counts` representa programação, execução, divergência, conclusão e
aprovação por empresa, almoxarifado, endereço, item/máscara e lote. Quantidades usam
`NUMERIC(20,6)` e datas operacionais são `TIMESTAMPTZ` em UTC. Cada transição deve ser
registrada em `stock_cycle_count_audit`; thresholds de antecedência/tolerância ficam
na assinatura JSONB. `tolerancia_quantidade` é o padrão da empresa;
`tolerancia_por_almoxarifado` e `tolerancia_por_item` têm precedência nessa ordem de
especificidade. Valores são decimais não negativos. A movimentação incomum aceita
`quantidade_limite`, `valor_limite`, `horario_inicio` e `horario_fim`.

## Operação e observabilidade

### Evento pontual x pendência recorrente

- `EVENTO` é uma mudança consumada, como um orçamento convertido em pedido. O e-mail é emitido uma única vez e não reaparece no dia seguinte.
- `PENDENCIA` é uma condição que continua exigindo ação, como estoque abaixo do mínimo. Se a assinatura for imediata, há o primeiro aviso e os lembretes começam no dia local seguinte. Na cadência de resumo, o primeiro aviso pode sair no próprio dia, após o horário configurado.
- Cada pendência aberta gera no máximo um e-mail individual por destinatário, empresa e data local. A chave contém alerta, usuário e data, mantendo múltiplas réplicas do worker idempotentes.
- O lembrete respeita `digest_time` e `timezone` da empresa e usa os dados completos mais recentes do alerta.
- Quando a condição é normalizada, o alerta é resolvido e os lembretes cessam. Uma violação futura abre um novo ciclo auditável.

O endpoint Prometheus existente inclui tamanho/idade da outbox, alertas abertos por
módulo/severidade, entregas por estado, tentativas, digestos, dead letters, volume
recente do provedor e latência evento→entrega, sem tenant, destinatário ou payload em
labels. O worker cria spans OpenTelemetry, aplica limites global/tenant, circuit
breaker persistido e aguarda drain no shutdown. A retenção preserva dead letters e
remove somente registros enviados/resolvidos após a janela configurada.

## Handoff frontend

A tela ADMIN deve ter abas Configuração, Assinaturas, Histórico e Dead letters.
Carregue a matriz por `events`, mostre horário/fuso IANA, permita selecionar somente
usuários/papéis/departamentos retornados pelos cadastros internos e exija confirmação
antes de salvar. Histórico deve paginar, exibir estado/tentativas sem corpo integral
e oferecer reenvio apenas para `FALHOU`/`DESCARTADO`. O teste de e-mail retorna
`202 {"status":"PENDENTE"}` e deve ser acompanhado pelo histórico.

Alertas do usuário são somente leitura nesta versão; marcar como lido futuramente não
resolverá a causa. Não oferecer endereço livre nem opção de automatizar destinatário
externo em qualquer tela desta central.

Aplicar as migrations `000308` a `000314` em ordem. A `000314` transforma
`items.warehouse_cyclical_count_config` em programações operacionais: usa o
`warehouse_code` do item, agenda a partir da ativação/última aprovação, impede
duplicidade aberta com lock e índice parcial e registra auditoria com origem
`POLITICA_ITEM`. Mudança ou desativação cancela apenas programação automática ainda
não iniciada. A API do item recebe e devolve
`warehouse.cyclical_count_config.days_interval`; `DaysInterval` é aceito apenas como
compatibilidade temporária de entrada. A API de ocorrências expõe `origin` e omite
`policy_days` somente nas programações `MANUAL`. O rollback remove central,
processo de contagem e triggers, mas deve ser usado somente em ambiente sem histórico real; retenção normal
é controlada por empresa e não apaga histórico silenciosamente.
