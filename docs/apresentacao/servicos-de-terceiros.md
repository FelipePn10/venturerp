# Serviços de terceiros

O controle de serviços externos acompanha pintura, tratamento térmico,
beneficiamento, usinagem e outras etapas executadas por fornecedores sem perder
o vínculo com a ordem de fabricação.

O comprador pode registrar preços por fornecedor, produto, configuração,
operação e data de vigência, indicar o fornecedor preferencial e guardar opções
com preço zero para cotação. Frete, conversão de unidade e fórmulas de produtos
configurados evitam planilhas paralelas e alimentam o custo padrão.
O custo real também apresenta os impostos recuperáveis separadamente, evitando
misturar a visão fiscal com o custo padrão de planejamento.

Durante o MRP, o PCP já enxerga as ordens de serviço planejadas por operação,
fornecedor e prazo. Ao incluir uma OF manual ou firmar a produção, o ERP cria as
ordens de serviço necessárias na mesma transação e mantém o
encadeamento entre OF, requisição e pedido de compra. O PCP consulta o que está no
fornecedor, o que está atrasado ou pendente e registra remessa, retorno e
recebimento. O sistema impede receber acima do contratado e conclui a ordem ao
atingir a quantidade total.

Conversões por item/configuração e conversões globais atendem contratos em
caixa, peça, quilo ou outra unidade. Para itens não fracionáveis, percentuais de
arredondamento e tolerâncias evitam tanto bloqueios desnecessários quanto ajustes
silenciosos fora da política da fábrica.

Toda alteração de preço exige motivo e permanece no histórico. Reajustes,
cópias e transferências de tabelas são executados em bloco: se um item falhar,
nenhum fica parcialmente atualizado. Cada movimentação possui uma chave de
idempotência, protegendo integrações contra lançamentos duplicados. Consultas e
históricos podem ser exportados em CSV, Excel, PDF ou Word para análise,
conferência e impressão.
