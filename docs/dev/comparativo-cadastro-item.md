# Comparativo do cadastro de item (VENT0200)

Verificação feita em 2026-11-06, antes da reforma da VENT0200. O objetivo foi achar o
que o nosso cadastro ainda não cobre e o que pode ir além dos ERPs de mercado.

## Até onde a evidência vai

Nem todo fabricante publica o que precisávamos ler. Esta tabela separa o que foi lido
do que não foi. Sem fonte não há afirmação de que um ERP "não tem" algo.

| ERP | Fonte | Nível |
|---|---|---|
| Oracle Fusion | Guia *Using Product Master Data Management* 26D, PDF lido por inteiro | Alto |
| Microsoft Dynamics 365 | Páginas `product-information` e `product-lifecycle` do Learn | Alto |
| SAP S/4HANA | Curso SAP Learning de material master e trechos do SAP Help (API do produto). As páginas do SAP Help não abrem sem JavaScript | Médio |
| Odoo 19 | Páginas de tipo de produto e de unidade, em parte por resumo | Médio |
| TOTVS Protheus | Trechos indexados da TDN e artigos da Central. A TDN recusou a leitura direta (403) | Baixo |
| FOCCO | Só o site comercial: cita "Item – PDM", "Engenharia" e "Configurador de produto". Não há ajuda pública | Muito baixo |

Fontes fiscais lidas por inteiro: Guia Prático EFD-ICMS/IPI 3.1.9 e NT 2025.002-RTC
v1.50 (cópia não oficial). Campos da NF-e na API: `campos.focusnfe.com.br`.

## Lacunas encontradas e o que foi feito

| Tema | Referência | Nosso cadastro antes | Agora |
|---|---|---|---|
| GTIN comercial e tributável | SAP (GTIN por unidade), Oracle (relação GTIN), NF-e cEAN/cEANTrib, EFD COD_BARRA | Não havia. A nota saía sem código | Campos validados pelo dígito GS1, únicos na empresa, enviados à NF-e e ao 0200, usados na entrada de nota |
| Unidade tributável | NF-e uTrib/qTrib/vUnTrib | Sempre igual à comercial | Unidade e fator por item |
| Tipo do item na EFD | Guia EFD, campo TIPO_ITEM | Deduzido, nunca 03, 05, 06 ou 10 | Informado no cadastro, vence a dedução |
| Validade e vida útil mínima | Oracle (Shelf Life Days) | Só por lote, digitada | Validade preenche o lote; vida mínima recusa no recebimento |
| Controle de lote, endereço e série | TOTVS (B1_RASTRO, B1_LOCALIZ), Odoo, Oracle | Em outras telas | Na própria VENT0200 |
| Tolerância de recebimento | SAP (sub e sobre-entrega) | Só por empresa e fornecedor | Por item; o excesso vence as outras regras |
| Roteiro de recebimento | Oracle (Receipt Routing) | Só a rota de inspeção | Direto, padrão ou inspeção obrigatória |
| Prazo de compra e fabricação | SAP (prazo de entrega planejado) | Só do fornecedor ou do roteiro | Prazos do item, usados pelo MRP na falta dos outros |
| Comprador e planejador | SAP (grupo de compradores) | Planejador sem campo na tela; comprador inexistente | Os dois |
| Produto perigoso e armazenagem | Oracle (Hazard Class, UN Number), SAP (condições) | Não havia | Campos, com ONU e classe no romaneio |
| Referências | Oracle (cross-reference, MPN) | Não havia | Código anterior, fabricante e MPN, com busca |
| Revisão e descrição longa | Oracle (Item Revisions, Long Description) | Não havia | Revisão vigente e descrição longa |
| Política por status e processo | Dynamics (Enabled, Blocked, Enabled with warning) | Todo status bloqueava tudo | Matriz na VENT0250, aplicada no gatilho dos documentos |
| PDM | Oracle (grupos de atributos), FOCCO ("Item – PDM") | Grupo e modificador soltos, atributo livre e nunca gravado | Vínculo grupo×modificador, modelo de atributos, descrição gerada e assinatura contra duplicado |

## Além do que as fontes lidas mostram

Nenhum destes foi localizado nas fontes acima. Isso não prova que os ERPs não têm: a
leitura do SAP e da TOTVS foi parcial.

- **Prontidão por processo.** A Oracle documenta um percentual de completude. Aqui a
  resposta é operacional: pronto para comprar, produzir, estocar, vender e faturar, com
  a pendência e a tela que resolve, olhando estrutura, roteiro, fornecedor e fiscal.
- **Prévia fiscal do item.** Antes da primeira nota, mostra o que vai ao XML e à EFD e o
  que a SEFAZ recusaria, incluindo o fabricado que é componente e talvez não seja 04.
- **Impacto antes de salvar.** A TOTVS documenta um caso (endereçamento exige saldo
  zerado). Aqui a mesma conferência cobre unidade, tipo, GTIN, tipo EFD e validade, com
  simulação na tela e recusa no backend.

## O que ficou de fora

- ANP: na NF-e o campo é do grupo de combustível, sem uso em metalúrgica e moveleira.
- Certificado na entrada por nota sem pedido: a tela de entrada não tem lote; as regras valem no recebimento do pedido.
- Situação (linha/promoção) segue como classificação: nenhuma regra faz sentido sem inventar processo.

## Auditoria dos indicadores

Cada indicador foi conferido pelo que o sistema faz com ele. Vários gravavam e nenhum
processo lia.

| Indicador | Antes | Agora |
|---|---|---|
| Ativo no MRP | O MRP ignorava | Fora do cálculo e da reposição |
| Protótipo | Sem efeito | Fora do MRP, com aviso na venda |
| Ferramenta | Sem efeito | Ficha de vida útil vinculada em VPRO1000; aviso sem ficha |
| Múltiplo e mínimo de venda | Sem efeito | Recusados na linha de pedido e de orçamento |
| Quantidade fracionada | Só na produção | Também na venda |
| Exclusivo e OEM | Sem efeito | Cliente exclusivo: outro cliente não compra; aviso sem cliente |
| Altera descrição no faturamento | Sem efeito | Desligado, a nota sai com a descrição do cadastro |
| Exige volumes no despacho | Sem efeito | Romaneio não despacha sem volumes |
| Embalagem diferenciada e de exportação | Sem efeito | Destacadas no romaneio |
| Check list de recebimento | Sem efeito | Recebimento exige a confirmação |

Indicadores novos, pensados para metalúrgica e moveleira: Sucata/retalho (subproduto na
EFD, fora do MRP), Exige certificado e corrida, Exige desenho, Sentido do material no
corte, Não girar no corte, Volumes por unidade e Cliente exclusivo.

Saíram da tela, sem apagar o valor gravado, porque nenhum processo os usa: controle de
safra, disponível no mobile, emite etiquetas, retenção de PIS/COFINS, almoxarifado de
transferência e de assistência técnica.

Ligados depois: Item de Processo sai do MRP; fator de volume calcula os volumes do romaneio;
item de embalagem é baixado no despacho, um por volume; entrega estimada vira a data padrão
da linha de pedido e de orçamento. A tolerância de falta encerra a linha, e o comprador filtra
a cobertura de materiais.

## Fontes

- Oracle: https://docs.oracle.com/en/cloud/saas/supply-chain-and-manufacturing/26d/fapim/using-product-master-data-management.pdf
- Microsoft: https://learn.microsoft.com/en-us/dynamics365/supply-chain/pim/product-lifecycle e https://learn.microsoft.com/en-us/dynamics365/supply-chain/pim/product-information
- SAP: https://learning.sap.com/courses/purchasing-in-sap-s-4hana/creating-material-master-records-for-purchasing e https://help.sap.com/docs/SAP_S4HANA_ON-PREMISE/18fe3fab96864826bfa0be0de4f65b85/74aa2b58a333a107e10000000a441470.html
- Odoo: https://www.odoo.com/documentation/19.0/applications/inventory_and_mrp/inventory/product_management/configure/type.html
- TOTVS: https://tdn.totvs.com/display/public/PROT/Produtos+-+MATA010 e https://centraldeatendimento.totvs.com/hc/pt-br/articles/360011734912
- FOCCO: https://foccoerp.com.br/sistema/gestao-para-industrias/
- EFD: https://www.confaz.fazenda.gov.br/legislacao/arquivo-manuais/06___anexoguia_pratico_da_escrituracao_fiscal_digital___efd.pdf
- NT 2025.002: https://www.reformatributaria.com/wp-content/uploads/2026/06/NT_2025.002_v1.50_RTC_NF-e_IBS_CBS_IS.pdf
- Focus NFe: https://campos.focusnfe.com.br/nfe/NotaFiscalXML.html
