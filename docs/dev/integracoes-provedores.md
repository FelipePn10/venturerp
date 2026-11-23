# Integrações externas — análise de provedores (22/11/2026)

Escopo: as três portas abertas na rodada de validação (`ports.CobrancaBancaria`,
`ports.Mensageiro`, `ports.RastreioDeCarga`), para indústria metalúrgica e moveleira
(B2B, carga fracionada, cobrança por boleto).

Critérios, nesta ordem: (1) API oficial e estável, com sandbox; (2) cobre o caso B2B
do cliente sem trocar o fluxo dele; (3) custo por operação; (4) esforço de integração
e de operação (certificado, homologação); (5) não prende o ERP — a porta continua
genérica, o adaptador é trocável.

## 1. Cobrança (boleto + PIX com baixa automática)

| Provedor | O que entrega | Custo (fontes de nov/2026) | Observações |
|---|---|---|---|
| **Banco Inter — API Cobrança v3** | boleto com Pix na mesma cobrança, webhook com a forma de recebimento, PDF, sandbox | liquidação de boleto isenta para PME (confirmar na tabela PJ) | OAuth2 + mTLS (certificado do Internet Banking, 1 ano); emissão assíncrona (`codigoSolicitacao`) |
| Sicoob | boleto híbrido (código de barras + QR Pix), CNAB 240 com Pix | sem tarifa Pix via API em relatos antigos — confirmar | exige certificado A1; Pix na remessa só no CNAB 240 |
| Efí (Gerencianet) | Pix, boleto, Pix Automático | 1,19% por Pix recebido via API | bom para varejo; caro para ticket B2B alto |
| Asaas | Pix e boleto sem conta em banco tradicional | R$ 1,99 por Pix recebido | intermediador: o dinheiro passa pela conta Asaas |

**Escolha: Banco Inter, API Cobrança v3.** É o único com cobrança híbrida
(boleto + Pix) numa chamada só, webhook dizendo como foi pago, sandbox e custo de
liquidação zero para PME — o perfil das empresas atendidas. Percentual (Efí) não serve
para nota de R$ 30 mil; intermediador (Asaas) muda o caixa do cliente. Quem já tem
convênio com outro banco segue no **CNAB 240** (remessa e retorno já existem no ERP) —
Sicoob é a segunda opção quando o cliente já é cooperado.

Implementado: `internal/infrastructure/integracoes/inter` (adaptador da porta) e
`POST /api/integracoes/inter/webhook/{token}` (o aviso só dispara a baixa depois de o
ERP consultar a cobrança no Inter — o corpo do aviso não é tomado como verdade).

Ciclo completo da cobrança no Inter:
1. **Emitir** (VFIN0210 › Cobrar): `POST /cobranca/v3/cobrancas` com o "seu número"
   `CR<id do título>`; o `codigoSolicitacao` fica em `contas_receber.cobranca_externa_id`.
   A emissão é assíncrona: o ERP espera alguns segundos pelos códigos (linha digitável,
   PIX copia e cola) e, se o banco ainda processa, devolve "pendente".
2. **Emitir de novo** não cria outra cobrança: consulta a mesma (`GET .../{codigo}`) e só
   emite outra se a anterior foi cancelada, expirou ou falhou no banco.
3. **Pagamento**: o Inter chama o webhook; o ERP consulta a cobrança e baixa o título
   (valor pago acima do saldo vira encargo). Dois avisos simultâneos não baixam duas vezes.
4. **Cancelar o título** cancela a cobrança no Inter antes (`POST .../{codigo}/cancelar`);
   se o banco falhar, o título NÃO é cancelado (o cliente ainda poderia pagar).
   **Baixa manual que quita** (recebido por outro meio) também cancela a do banco.
5. **Cadastro do webhook**: VFIN0100 › Integrações › "Cadastrar webhook no Inter"
   (`PUT /cobranca/v3/cobrancas/webhook`), informando o endereço público HTTPS da API.
6. Quem não usa Inter: **CNAB 240** (remessa na VFIN0610 e retorno na aba Retorno).

## 2. WhatsApp (aviso de cobrança, pedido de compra)

| Provedor | Custo por mensagem (além da Meta) | Observações |
|---|---|---|
| **Meta — WhatsApp Cloud API (direto)** | nenhum: só a tarifa Meta da categoria *utility* (≈ US$ 0,007–0,008 no Brasil; faturamento em BRL desde jul/2026) | API oficial, sem intermediário; exige modelo (template) aprovado para iniciar conversa |
| Gupshup | ≈ US$ 0,001 | painel pouco amigável |
| Twilio | US$ 0,005 enviada e recebida | caro para volume |
| Zenvia | modelo de R$ 0,47–0,55 por mensagem | caro; plano com pacote |

**Escolha: Cloud API da Meta, direto.** O ERP manda mensagens transacionais (utility)
iniciadas pela empresa, em volume baixo: intermediário só soma custo e mais um contrato.
Requisitos do cliente: conta Meta Business verificada, número dedicado e um modelo
utility aprovado com uma variável no corpo (o ERP preenche com o texto do aviso).

Implementado: `internal/infrastructure/integracoes/whatsapp` (Graph API,
`POST /{versão}/{phone-number-id}/messages`, tipo `template`). O número do cadastro
(DDD + número) ganha o DDI 55 antes de ir à Meta. Usos: aviso de cobrança (VFIN0210 ›
Cobrar) e resumo do **pedido de compra** ao fornecedor (envio do pedido, `POST
/api/purchase-order/{code}/enviar-whatsapp`, registrado no mesmo histórico do e-mail; o
cadastro do fornecedor não tem celular, então o número é digitado no envio).

## 3. Transportadora (cotação e rastreio)

| Provedor | Perfil | Observações |
|---|---|---|
| **Frete Rápido** | hub B2B: carga fracionada e lotação, 600+ transportadoras, contratos do próprio embarcador | API de cotação v3, ocorrências por NF-e (chave) e webhook com 12 retentativas; já integrada a ERPs (Olist/Tiny, ERPFlex) |
| Melhor Envio | etiquetas com desconto, pequeno volume | voltado a e-commerce de encomenda |
| Frenet | gateway de cotação no checkout | foco em loja virtual |
| Intelipost | TMS para grande operação | a partir de R$ 299/mês; dimensionado para milhares de pedidos/dia |

**Escolha: Frete Rápido.** É o único da lista pensado para carga fracionada B2B
(chapa, perfil, móvel montado), com rastreio pela chave da NF-e — que o ERP já tem em
todo romaneio faturado — e usando o contrato que a empresa já negociou com a
transportadora. Kangu foi descartada (encerrada, segundo uma das fontes).

Implementado: `internal/infrastructure/integracoes/freterapido` (cotação v3 e
ocorrências por NF-e) e as rotas `POST /api/shipments/{code}/cotacao-online` e
`GET /api/shipments/{code}/rastreio`.

## Configuração (por empresa, no `.env` da instância)

```
# Inter (cobrança) — certificado e chave do Internet Banking › Integrar
INTER_CLIENT_ID=  INTER_CLIENT_SECRET=  INTER_CERT_FILE=  INTER_KEY_FILE=
INTER_CONTA_CORRENTE=        # só se a integração cobre mais de uma conta
INTER_SANDBOX=false
INTER_CONTA_BANCARIA_ID=     # conta do ERP (contas_bancarias.id) que recebe a baixa
INTER_WEBHOOK_TOKEN=         # segredo do caminho do webhook (32+ caracteres)
# WhatsApp Cloud API
WHATSAPP_TOKEN=  WHATSAPP_PHONE_NUMBER_ID=  WHATSAPP_TEMPLATE=aviso_erp  WHATSAPP_TEMPLATE_LANG=pt_BR
# Frete Rápido
FRETERAPIDO_TOKEN=  FRETERAPIDO_PLATFORM_CODE=
# Portal do cliente: endereço público onde o cliente abre o link
PORTAL_BASE_URL=
```

Sem as variáveis, a porta usa o adaptador desligado e a tela diz "integração não
configurada" — nada quebra. **No treino (`DATA_ENVIRONMENT=training`) os três
adaptadores ficam desligados mesmo com as variáveis preenchidas:** a cópia tem
clientes e títulos reais, e um boleto ou WhatsApp saído de lá chegaria de verdade.

O aviso do Inter entra pela rota pública `POST /api/integracoes/inter/webhook/{token}`
(lista de livres em `catalogo_rotas.go`). A baixa sai em nome do primeiro ADMIN ativo
da empresa `INTER_EMPRESA_ID` (padrão 1), na conta `INTER_CONTA_BANCARIA_ID`; o
excedente pago sobre o saldo entra como encargo. Resposta 503 faz o Inter reenviar.

## 4. Portal do cliente (fase 0)

Não é provedor externo: é o próprio ERP aberto ao cliente da empresa, por link seguro.
O vendedor gera o link na VCLI0500 › Portal do cliente (`POST /api/customers/{code}/portal-links`,
validade de 7 a 365 dias, revogável). O token tem 32 bytes aleatórios; o banco guarda só o
hash SHA-256. O cliente abre `<PORTAL_BASE_URL>/#/portal/<token>` (rotas públicas
`GET /api/portal/{token}`, `POST /api/portal/{token}/pedidos/{code}/repetir` e
`GET /api/portal/{token}/notas/{id}/{danfe|xml}`, com limite próprio de 60/min por IP),
vê os pedidos, parcelas e notas dele, baixa DANFE/XML e pede de novo — o pedido entra como
rascunho de origem PORTAL e a empresa é avisada (pendência, alerta, e-mail do
representante, faixa na VVND0200). Cada acesso, download e pedido fica em `portal_events`.
O plano completo está em `.ai/tasks/portal-do-cliente.md`.

Regras de segurança (testadas em `portal_integration_test.go` e no ataque de caixa-preta
da rodada de validação):
- o cliente vem do token, nunca do navegador; pedido, nota e título são filtrados por
  empresa E cliente; pedido/nota de outro cliente responde 404 genérico;
- só pedido confirmado, em análise ou faturado se repete (orçamento, rascunho do
  vendedor e pedido do portal ainda aberto, não); o mesmo pedido não é repetido enquanto o
  anterior aguarda; no máximo 5 pedidos do portal aguardando por cliente;
- cliente inativado perde o acesso na hora; link revogado ou vencido responde 404;
- o token não vai para log, auditoria nem rastro (`CaminhoSemSegredo`, filtro do
  OpenTelemetry) — o mesmo vale para o segredo do webhook do Inter;
- o pedido do portal que a empresa cancela aparece ao cliente como "Não confirmado".

### Publicar o portal (pendente — a página ainda não está em endereço público)

A página é a rota `#/portal/...` do front. Hoje só existe dentro do app desktop; o site
`app.venturerp.com` (Vercel) serve só o download e o `tenants.json`. Para o cliente abrir:
1. publicar o build web do front num endereço público (ex.: `cliente.venturerp.com`,
   projeto próprio na Vercel, com `X-Frame-Options: DENY` e `Referrer-Policy: no-referrer`);
2. em cada API: `PORTAL_BASE_URL=https://cliente.venturerp.com`, `PORTAL_EMPRESA=<id no
   tenants.json>` (o link sai `…/#/portal/<empresa>/<token>` e a página procura a API da
   empresa no catálogo — nunca um endereço vindo do link) e acrescentar a origem do portal
   em `CORS_ALLOWED_ORIGINS` (hoje só `tauri://…`; sem isso o navegador barra a chamada).

Configuração pronta (não publicada) em `app-venture/portal-cliente/` (`vercel.json` com CSP,
sem referrer, sem indexação, sem iframe; README com os passos). Ensaio do build de
produção com essa CSP: página, download e empresa fora do catálogo OK, sem chamada à
produção. A página do portal fica fora da trava de versão do desktop
(`SystemUpdateGate`), que a bloqueava no build de produção.

## Situação e telas

`GET /api/integracoes/status` diz o que está ligado (só booleanos); a VFIN0100 mostra o
quadro "Integrações" e as telas desabilitam o botão da integração desligada.

## Telas (desktop)

| Tela | O que usa |
|---|---|
| VFIN0210 Contas a Receber → "Cobrar" | emitir boleto + PIX (Inter) e aviso por e-mail/WhatsApp |
| VFIN0610 Remessa e Retorno CNAB → aba Retorno | baixa pelo arquivo de retorno CNAB 240 |
| VEXP0100 Expedição → aba "Por pedido (guiado)" | cotação de frete e rastreio pela chave da NF-e (Frete Rápido) |
| VFIN0100 Contas Bancárias → Integrações | situação das integrações e cadastro do webhook do Inter |
| Envio do pedido de compra (VPDC) | resumo do pedido por WhatsApp |
| VCLI0500 Cliente → Portal do cliente | gerar, acompanhar e revogar o link do cliente |

## Antes de ligar em produção

- Inter: homologar no sandbox (fora do horário 8h–20h o sandbox cai); conferir a tabela
  de tarifas PJ; cadastrar o webhook pela VFIN0100 (ou `PUT /cobranca/v3/cobrancas/webhook`)
  com o endereço público `https://<api>`.
- WhatsApp: aprovar o modelo na Meta; a variável não aceita quebra de linha (o ERP troca
  por " · ").
- Frete Rápido: pedir o `platform_code` e o token; passar pela homologação deles.
- Pedir cotação formal: as tarifas acima vêm de comparativos publicados, não de proposta.

## Fontes

- [Comparativo de APIs bancárias para ERP (KMEE)](https://kmee.com.br/blog/comparativo-apis-bancarias-erp-brasil-2026/)
- [Sicoob — boleto híbrido (Tecnospeed)](https://atendimento.tecnospeed.com.br/hc/pt-br/articles/28671440838679-756-Sicoob)
- [Pix híbrido no boleto / CNAB 240](https://maxwbh.github.io/cobranca-api/api/pix.html)
- [Tarifas Efí](https://sejaefi.com.br/central-de-ajuda/tarifas-e-prazos/tarifas-cobradas-pela-gerencianet)
- [Plataformas para receber Pix (Asaas)](https://www.abacatepay.com/blog/melhores-plataformas-para-receber-pix)
- [Como emitir boleto — Inter isento para PME](https://nacao.digital/blog/como-emitir-boleto/)
- [Inter — changelog API Boletos (V2 descontinuada em 18/05/2026)](https://developers.inter.co/changelog/18/05/2026)
- [SDK inter.js (tipos gerados da OpenAPI do Inter)](https://github.com/unfoldingcx/inter.js)
- [Correção de campos aninhados da cobrança v3](https://github.com/eseixas/bancointer-whmcs/pull/5)
- [Preços da WhatsApp Business Platform (Meta)](https://developers.facebook.com/documentation/business-messaging/whatsapp/pricing)
- [WhatsApp Business API: preços e provedores](https://nacao.digital/blog/whatsapp-business-api/)
- [Preço da API do WhatsApp no Brasil](https://yavendio.com/br/blog/preco-api-whatsapp-brasil)
- [WhatsApp pricing explained (Courier)](https://www.courier.com/guides/whatsapp-pricing)
- [Documentação Frete Rápido](https://dev.freterapido.com.br/)
- [Frete Rápido — cotação v3](https://dev.freterapido.com.br/ecommerce/cotacao_v3/)
- [Plataformas de frete comparadas](https://nacao.digital/blog/plataformas-de-frete/)
- [Melhor Envio vs Frenet vs Intelipost](https://www.akuma.com.br/akuma-academy/melhor-envio-vs-frenet-vs-intelipost-qual-escolher-para-escalar-sua-operacao-logistica)
- [Plataformas de frete 2026 (SuperFrete)](https://superfrete.com/blog/plataformas-de-frete)
