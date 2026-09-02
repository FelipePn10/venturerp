# Decisões enterprise — suprimentos e compras

## Escopo e referências

Decisões para a rodada `suprimentos.md`, comparadas com documentação oficial do
[SAP](https://help.sap.com/doc/e2048712f0ab45e791e6d15ba5e20c68/1709/en-US/FSD_OP1709_latest.pdf),
[Oracle Supplier Model](https://docs.oracle.com/en/cloud/saas/procurement/26a/oaprc/oracle-supplier-model.html),
[Oracle Invoice Tolerances](https://docs.oracle.com/en/cloud/saas/financials/25d/fappp/invoice-tolerances.html),
[Dynamics 365](https://learn.microsoft.com/en-us/dynamics365/supply-chain/procurement/purchase-order-approval-confirmation)
e [Odoo Subcontracting](https://www.odoo.com/documentation/18.0/applications/inventory_and_mrp/manufacturing/subcontracting/subcontracting_basic.html).

## Contrato canônico e rastreabilidade

- `VSUP0200` é a tela canônica do pedido de compra; `VPDC0200` passa a ser apenas
  um alias de navegação para o mesmo contrato `/api/purchase-order`. `VSUP0300`
  continua sendo requisição, não pedido. O pedido possui capa e linhas próprias;
  nenhuma ação de item referencia pedido de venda como se fosse compra.
- Sugestão `PURCHASE` do MRP é firmada como pedido de origem `MRP`. A linha mantém
  o vínculo com a ordem planejada/demanda de origem e o fluxo seguinte preserva
  pedido → recebimento → movimento de estoque. A origem nunca é inferida por texto
  livre do cliente.
- Empresa e ator são claims autenticadas. Campos `enterprise_code`,
  `enterprise_id` e `created_by` não fazem parte de bodies públicos; valores
  enviados por clientes antigos são ignorados durante a compatibilidade.
- Mutações de pedido e recebimento devem usar chave idempotente e auditoria
  imutável de antes/depois. Identidade do responsável é resolvida pelo cadastro de
  usuários, mantendo o UUID como fallback técnico e nunca o texto "Usuário não localizado".

## Fornecedor único

- `VSUP0500` é o cadastro único. As funções antes apresentadas em `VSUP0510`
  tornam-se abas de parâmetros, empresas/sites, contatos, condições comerciais e
  conversões do fornecedor. `VSUP0510` deve ser removida do catálogo do frontend.
- O fornecedor é uma identidade global com configuração transacional por
  empresa/site, alinhado ao Supplier Model da Oracle. Bloqueio/homologação são
  avaliados no tenant antes de autorizar pedido; desbloqueio exige motivo e gera
  histórico.
- `VSUP0130` não mantém um segundo código de fornecedor. O identificador canônico
  é o do cadastro único e códigos externos pertencem à relação item-fornecedor.

## Conversões e terceiros

- `VSUP0110` é o cadastro canônico de conversão por item e empresa. `VTER0400`
  deixa de ser cadastro global concorrente e deve navegar para a mesma função.
  Conversões distinguem UM de estoque, compra e venda, permitem vigência/faixa e
  política de arredondamento, e rejeitam fatores não positivos e ciclos
  inconsistentes.
- O contrato público recebe `item_code` textual e resolve o ID interno dentro do
  tenant; inteiros positivos permanecem aceitos temporariamente com telemetria.
- Serviço terceirizado é tratado como compra vinculada à OF e à requisição/pedido
  de compra. Conforme o fluxo de subcontracting do Odoo, confirmação gera o
  recebimento e movimentações sem perder a origem de produção.

## Alçadas, tolerâncias e EDI

- Alçadas usam escopo `GLOBAL|SUPPLIER|COST_CENTER|CATEGORY`; `scope_ref` é nulo
  apenas em `GLOBAL`. Mudança relevante reinicia a aprovação e fica auditada.
- Tolerâncias se aplicam a quantidade, preço ou total, por valor fixo ou percentual,
  com ação `WARN|BLOCK`, faixa não sobreposta e override opcional por fornecedor.
  Isso segue o padrão Oracle de tolerâncias de quantidade/valor no recebimento e
  matching da fatura.
- Parâmetros usam catálogo fechado de domínio e `STRING|NUMBER|BOOL|JSON`.
- EDI aceita somente `PO_CONFIRMATION`, `ASN` e `INVOICE`, direção explícita e JSON
  válido. Mensagens preservam referência externa/idempotência e divergências; a
  configuração pertence ao fornecedor/site, como no Collaboration Messaging da
  Oracle.

## Compatibilidade de telas

Remover do catálogo, sem remover dados: `VSUP0510` (absorvida por `VSUP0500`) e
`VTER0400` (absorvida por `VSUP0110`). `VPDC0200` e `VSUP0200` apontam para a mesma
rotina canônica até a retirada do alias legado.
