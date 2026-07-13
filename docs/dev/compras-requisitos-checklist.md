# Compras — matriz de requisitos

Rastreabilidade da implementação da task `.ai/tasks/compras.md`.

## Diagnóstico anterior à implementação

| Área | Situação inicial | Evidência e impacto identificado |
|---|---|---|
| Tabelas de preços | Parcialmente implementado | Entidade, use case, handler, queries e tabelas já existiam, porém sem tenant/fornecedor no cabeçalho, precisão decimal, candidatos, ajustes e validação de fornecedor preferencial. |
| Preços por pedido/nota | Não implementado | Não havia consulta nem aplicação das duas fontes; exigiu repositório, DTOs, endpoints e transação de atualização. |
| Tolerâncias de pedido | Não implementado | Não havia domínio, persistência ou avaliação; exigiu migration, domínio, port, use case, repositório, handler e integração com recebimento/fiscal. |
| Itens por fornecedor | Parcialmente implementado | Existiam código, descrição, UM, ranking e prazo; faltavam tenant, ocorrências, máscara, classificação, regras fiscais/comerciais, validade e qualidade. |
| Configurador/PDM | Já implementado | Os módulos existentes foram preservados; compras referencia a máscara/configuração e não duplicou essas capacidades. |
| Interface cliente | Fora do escopo | A task é implementada no backend; nomes de botões foram traduzidos em operações HTTP, sem criar uma camada visual inexistente neste repositório. |

Ordem de dependência aplicada: migration e contratos de domínio; casos de uso;
repositórios e transações; handlers e composição da API; integração com recebimento
e fiscal; testes; documentação. Os principais pontos afetados foram
`migrations/000237_*`, `internal/domain/{purchase_price,item_supplier,purchase_tolerance}`,
os respectivos DTOs/use cases/repositórios/handlers e `api/api.go`.

| Processo / requisito | Situação | Implementação |
|---|---|---|
| Tabela por empresa e fornecedor, vigência e moeda | Concluído | `purchase_price_tables`, `/api/purchase-price-tables` |
| Item único por tabela/fornecedor; somente preço positivo | Concluído | índice `ux_purchase_price_item` e validação de domínio |
| UM, quantidade mínima e precisão monetária | Concluído | `NUMERIC(18,6)` e `decimal.Decimal` |
| Atualização do valor de reposição somente para preferencial | Concluído | `update_replacement_value` + validação no use case |
| Pesquisa interna/fornecedor, fallback de descrição e ordenação | Concluído | `GET /{code}/candidates` |
| Inclusão por classificação sem itens já precificados | Concluído | filtro `classification_id` nos candidatos |
| Descontos/acréscimos e copiar/colar | Concluído | ajustes e `copy-adjustments` (`REPLACE`/`ADD`) |
| Atualização por pedido, nota ou ambos, por período | Concluído | `GET /sources` e `POST /sources/apply` |
| Excluir serviço das fontes | Concluído | pedidos `OSL` e linhas fiscais sem item não são elegíveis |
| Atualizar UM e respeitar sobreposição | Concluído | aplicação transacional com `overwrite` explícito |
| Tolerâncias por tipo, aplicação, intervalo, limite e ação | Concluído | `purchase_order_tolerances` e API própria |
| Regra específica substitui a genérica | Concluído | resolução por fornecedor |
| Aplicação no aviso/recebimento físico | Concluído | `ReceivePurchaseOrderUseCase` |
| Aplicação na nota e importação XML | Concluído | entrada fiscal e XML vinculados ao pedido |
| Código, descrição, UM, UM XML e embalagem do fornecedor | Concluído | cadastro item–fornecedor ampliado |
| Ocorrências por máscara e classificação sincronizada | Concluído | chave por máscara e atualização transacional |
| Preferencial, faturamento direto e pedido de terceiros | Concluído | validações de dependência no domínio |
| Custo médio, e-commerce, barcode, observação e validade | Concluído | campos e barcode único por tenant/fornecedor |
| UF do fornecedor | Concluído | derivada do endereço preferencial |
| Conversão legada somente para item genérico | Concluído | validação pela natureza do item |
| Dados/laudo de qualidade | Concluído | tabela e endpoints de qualidade |
| Configurador e PDM | Reutilizado | módulos existentes; compras referencia `mask` |
| Isolamento por empresa | Concluído | tenant da autenticação em leituras e escritas |

Os nomes de botões da especificação pertencem à interface cliente. A API entrega
candidatos, operações em lote e modos de cópia sem colocar lógica de apresentação
nos handlers.
