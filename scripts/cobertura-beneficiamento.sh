#!/usr/bin/env bash
# Cobertura das regras do beneficiamento pedidas pela Usimac (documento 6 do
# levantamento complementar) contra o código deste worktree.
#
# Por que existe: a lista é longa e mora num PDF. Sem uma conferência que rode, a
# resposta "está tudo atendido" vira memória, e memória sobre requisito de cliente
# é o que produz surpresa na homologação.
#
# Os dois CONTROLES NEGATIVOS no fim TÊM de dar ✗. Eles provam que a conferência
# sabe dizer não — uma varredura em que tudo passa não mediu nada. Se um controle
# passar a dar ✓, o mecanismo quebrou (ou o item foi implementado e o controle
# precisa mudar).
#
# Uso: bash scripts/cobertura-beneficiamento.sh    (na raiz do worktree)
M=migrations/000370_beneficiamento_estoque_de_terceiros.up.sql
A=migrations/000371_beneficiamento_auditoria.up.sql
E=migrations/000372_beneficiamento_estorno_da_nota.up.sql
R=internal/infrastructure/repository/customer_material/repository.go
U=internal/application/usecase/customer_material_uc
ok=0; fail=0
v() { # v "rótulo" comando...
  local rot="$1"; shift
  if "$@" >/dev/null 2>&1; then printf "  ✓ %s\n" "$rot"; ok=$((ok+1));
  else printf "  ✗ %s\n" "$rot"; fail=$((fail+1)); fi
}
echo "── documento 6: regras do beneficiamento ──"
v "retorno total zera o saldo (coluna gerada)"            grep -q "GENERATED ALWAYS AS (qty_received - qty_returned - qty_leftover - qty_scrapped) STORED" "$M"
v "encerra sozinha quando o saldo zera"                   grep -q "THEN 'ENCERRADA'::customer_material_remittance_status_enum" "$R"
v "vínculo com o pedido de venda"                         grep -q "sales_order_code" "$M"
v "vínculo com a ordem de produção"                       grep -q "production_order_id" "$M"
v "vínculo com a NF-e de remessa (número/série/chave)"    grep -q "nfe_key" "$M"
v "vínculo com a nota de retorno"                         grep -q "fiscal_exit_id" "$M"
v "retorno parcial baixa só o processado"                 grep -q "TestBaixaPorNotaNaoPassaDoSaldo" internal/infrastructure/repository/customer_material/repository_integration_test.go
v "sobra devolvida com CFOP 5903"                         grep -q 'CFOPSobra *= *"5903"' "$U/nota_de_retorno.go"
v "sucata exige destinação"                               grep -q "customer_material_movements_sucata_com_destino" "$M"
v "as 4 destinações de sucata do documento"               grep -q "CLIENTE.*DESCARTE.*RETENCAO.*OUTRA" "$M"
v "recebimento divergente registra nota/físico/diferença" grep -q "divergence_qty .*GENERATED" "$M"
v "divergência exige motivo"                              grep -q "qty_received = qty_invoiced OR divergence_reason IS NOT NULL" "$M"
v "divergência registra o responsável"                    grep -q "divergence_settled_by" "$M"
v "material divergente entra bloqueado"                   grep -q "divergente" "$U/usecase.go"
v "material sem pedido entra bloqueado"                   grep -q "sem pedido" "$U/usecase.go"
v "bloqueio exige motivo"                                 grep -q "customer_material_remittances_bloqueio_com_motivo\|blocked = FALSE OR block_reason IS NOT NULL" "$M"
v "várias remessas para um pedido (sem unique no pedido)" grep -q "idx_customer_material_remittances_pedido" "$M"
v "uma remessa em vários faturamentos (idempotência)"     grep -q "customer_material_movements_idempotencia" "$M"
v "prazo fiscal de 30 dias"                               grep -q "PrazoFiscalPadraoDias = 30" "$U/usecase.go"
v "histórico de descrição e NCM"                          grep -q "record_customer_material_audit" "$A"
v "trilha guarda anterior, novo, usuário e motivo"        grep -q "before_state.*JSONB" "$A"
v "trilha imutável"                                       grep -q "prevent_customer_material_audit_mutation" "$A"
v "cancelamento estorna as movimentações"                 grep -q "EstornarNotaCancelada" internal/application/usecase/fiscal_uc/cancel_fiscal_exit_uc.go
v "estorno marca o movimento e devolve o saldo"           grep -q "reversed_at" "$E"
v "encerramento com saldo exige motivo e autor"           grep -q "closed_by" "$M"
v "sem operação interestadual (6.124/6.902/6.903)"        sh -c "! grep -qE '\"6124\"|\"6902\"|\"6903\"' $U/nota_de_retorno.go"
echo "── controle negativo (têm de dar ✗) ──"
v "CONTROLE: conversão kg↔peça (documento diz não se aplica)" grep -q "conversaoDeUnidade" "$R"
v "CONTROLE: campo fechado de destinação do saldo no encerramento" grep -q "close_destination" "$M"
echo
echo "confirmados: $ok | não confirmados: $fail  (2 dos não confirmados são os controles negativos)"
