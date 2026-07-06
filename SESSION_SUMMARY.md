# Session Summary

## Estado Atual

O módulo Comercial foi evoluído em 14 fases, com a fase 11 deliberadamente
pulada. A documentação final descreve somente o VentureERP, suas APIs, suas
regras e seus fluxos internos.

## Fases Comerciais Entregues

1. **Precificação**
   - Formação de preço de venda, tabelas de venda, preços por tabela, políticas
     de formação de preço, margens, markup, tolerâncias e validações.
   - Docs: `docs/dev/custos.md`, `docs/dev/vendas.md`,
     `docs/apresentacao/custos.md`, `docs/apresentacao/vendas.md`.
   - Script: `scripts/test-comercial-pricing.sh`.

2. **Política Comercial**
   - Políticas de descontos, acréscimos, fretes, comissões, regras comerciais,
     classificações específicas e relatórios de política.
   - Docs: `docs/dev/vendas.md`, `docs/apresentacao/vendas.md`.
   - Script: `scripts/test-comercial-politicas.sh`.

3. **Orçamentos**
   - Cadastro, itens, consulta, cancelamento, descancelamento, atendimento,
     relatório e conversão de orçamento em pedido.
   - Docs: `docs/dev/vendas.md`, `docs/apresentacao/vendas.md`.
   - Script: `scripts/test-comercial-orcamentos.sh`.

4. **Pedidos de Venda**
   - Pedido completo com crédito, reserva, análise comercial/financeira,
     liberação, conferência, cancelamento, atendimento e relatórios.
   - Docs: `docs/dev/vendas.md`, `docs/apresentacao/vendas.md`.
   - Script: `scripts/test-comercial-pedido-venda.sh`.

5. **Representantes**
   - Tipos de representantes, cadastro, comissões, ficha de acompanhamento e
     relatórios.
   - Docs: `docs/dev/vendas.md`, `docs/apresentacao/vendas.md`.
   - Script: `scripts/test-comercial-representantes.sh`.

6. **Metas de Vendas**
   - Períodos, metas por representante, metas por grupo comercial, clientes
     vinculados, saldos e relatórios planejado versus realizado.
   - Docs: `docs/dev/vendas.md`, `docs/apresentacao/vendas.md`.
   - Script: `scripts/test-comercial-metas.sh`.

7. **Previsão de Vendas**
   - Cadastro e geração de previsões, blocos, apropriações/rateios, geração por
     histórico e integração com planejamento.
   - Docs: `docs/dev/vendas.md`, `docs/apresentacao/vendas.md`.
   - Script: `scripts/test-comercial-previsao-vendas.sh`.

8. **Promessa de Entrega**
   - Parâmetros, ATP, CTP, calendário industrial, manutenção de promessa,
     reprogramação e comprometimento de tanque.
   - Docs: `docs/dev/vendas.md`, `docs/apresentacao/vendas.md`.
   - Script: `scripts/test-comercial-promessa-entrega.sh`.

9. **Assistência Técnica**
   - Grupos/motivos de defeito, responsáveis de garantia, chamados, notas de
     devolução, geração de ordens, consulta e relatório.
   - Docs: `docs/dev/vendas.md`, `docs/apresentacao/vendas.md`.
   - Script: `scripts/test-comercial-assistencia-tecnica.sh`.

10. **Atendimento ao Consumidor**
    - Consumidores, contatos, chamados, histórico, etiquetas/base de etiquetas,
      checklist, retornos, anexos, consulta e relatório.
    - Docs: `docs/dev/vendas.md`, `docs/apresentacao/vendas.md`.
    - Script: `scripts/test-comercial-sac.sh`.

11. **Central de Vendas**
    - Fase pulada por decisão do usuário.

12. **Vendas Recorrentes**
    - Parâmetros, console de contratos recorrentes, geração de pedidos, reajustes,
      receita mensal recorrente e comissões futuras.
    - Docs: `docs/dev/vendas.md`, `docs/apresentacao/vendas.md`.
    - Script: `scripts/test-comercial-vendas-recorrentes.sh`.

13. **Expedição**
    - Romaneio, cargas, vínculo carga-romaneio, notas fiscais por carga,
      controle de carregamento, liberação, orientações de entrega, boxes,
      reserva para carga, monitor de expedição, monitor de separação e painel
      logístico.
    - Docs: `docs/dev/romaneio.md`, `docs/apresentacao/romaneio.md`.
    - Script: `scripts/test-comercial-expedicao.sh`.

14. **Faturamento**
    - NF-e de saída, NFS-e, emissão por carga, rastreabilidade de cupom/NFC-e/CF-e,
      DANFE/XML, autorização, cancelamento, carta de correção, vínculo financeiro,
      baixa de estoque e baixa de reserva.
    - Docs: `docs/dev/fiscal-financeiro.md`,
      `docs/apresentacao/fiscal-financeiro.md`.
    - Script: `scripts/test-comercial-faturamento.sh`.

## Implementação Mais Recente

- Adicionada rastreabilidade de origem em `fiscal_exits`:
  - `source_type`
  - `shipment_load_code`
  - `shipment_code`
  - `fiscal_coupon_number`
  - `fiscal_coupon_date`
  - `fiscal_coupon_ecf_serial`
- Criado o endpoint `POST /api/fiscal/exits/from-load`.
- Criado o caso de uso
  `internal/application/usecase/fiscal_uc/create_fiscal_exit_from_load_uc.go`.
- Criado teste unitário
  `internal/application/usecase/fiscal_uc/create_fiscal_exit_from_load_uc_test.go`.
- Criada migration `000197_fiscal_exit_sources`.
- Atualizadas as docs fiscal/financeiro e os exemplos em
  `docs/dev/API_REQUEST_BODIES.txt`.

## Validações Recentes

- `scripts/test-comercial-faturamento.sh`
- `env GOCACHE=/tmp/panossoerp-go-build go test ./internal/application/usecase/fiscal_uc ./internal/application/usecase/shipment_uc`
- `env GOCACHE=/tmp/panossoerp-go-build go test ./internal/domain/fiscal/... ./internal/domain/shipment/...`
- `env GOCACHE=/tmp/panossoerp-go-build go test ./...`

O cache Go padrão em `/home/felipepanosso/.cache/go-build` é somente leitura no
ambiente atual. Use `GOCACHE=/tmp/panossoerp-go-build` nos testes locais.

## Observações Para Próximos Agentes

- Não reverter alterações não commitadas sem pedido explícito do usuário.
- Manter a documentação final centrada no VentureERP, com linguagem de produto
  própria.
- Rotas, tabelas e nomes de código devem seguir linguagem interna do VentureERP.
- A fase 11 permanece pulada até novo comando do usuário.
