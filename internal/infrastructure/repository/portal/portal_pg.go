// Package portal guarda o Portal do Cliente, fase 0: links de acesso (só o hash
// do token), auditoria e as leituras que o cliente pode ver — sempre filtradas
// pela empresa E pelo cliente do link, nunca por código vindo do navegador.
package portal

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/FelipePn10/panossoerp/internal/application/usecase/portal_uc"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"
)

type Repo struct{ pool *pgxpool.Pool }

func New(pool *pgxpool.Pool) *Repo { return &Repo{pool: pool} }

var _ portal_uc.Repo = (*Repo)(nil)

func (r *Repo) CriarLink(ctx context.Context, cliente int64, hash []byte, expira time.Time, descricao string, autor uuid.UUID) (int64, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return 0, err
	}
	var id int64
	err = r.pool.QueryRow(ctx, `INSERT INTO portal_links (enterprise_id, customer_code, token_hash, descricao, expires_at, created_by)
		SELECT $1, c.code, $3, NULLIF($4,''), $5, $6 FROM customers c WHERE c.enterprise_id = $1 AND c.code = $2 AND c.is_active
		RETURNING id`, empresa, cliente, hash, descricao, expira, autor).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, portal_uc.ErrNaoEncontrado
	}
	if err != nil {
		return 0, fmt.Errorf("criando o link do portal: %w", err)
	}
	return id, nil
}

func (r *Repo) ListarLinks(ctx context.Context, cliente int64) ([]portal_uc.Link, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, `SELECT id, customer_code, COALESCE(descricao,''), expires_at, revoked_at, last_access_at, access_count, created_at
		FROM portal_links WHERE enterprise_id = $1 AND customer_code = $2 ORDER BY created_at DESC`, empresa, cliente)
	if err != nil {
		return nil, fmt.Errorf("listando os links do portal: %w", err)
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (portal_uc.Link, error) {
		var l portal_uc.Link
		err := row.Scan(&l.ID, &l.CustomerCode, &l.Descricao, &l.ExpiraEm, &l.RevogadoEm, &l.UltimoAcesso, &l.Acessos, &l.CriadoEm)
		return l, err
	})
}

func (r *Repo) RevogarLink(ctx context.Context, id int64) error {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	tag, err := r.pool.Exec(ctx, `UPDATE portal_links SET revoked_at = COALESCE(revoked_at, now()) WHERE id = $1 AND enterprise_id = $2`, id, empresa)
	if err != nil {
		return fmt.Errorf("revogando o link do portal: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return portal_uc.ErrNaoEncontrado
	}
	return nil
}

// SessaoPorToken: o link vigente pelo hash do token (sem empresa no contexto:
// é o token que diz a empresa), com o ADMIN que assina o que o portal grava.
func (r *Repo) SessaoPorToken(ctx context.Context, hash []byte) (*portal_uc.Sessao, error) {
	s := &portal_uc.Sessao{}
	var ator *uuid.UUID
	err := r.pool.QueryRow(ctx, `
		UPDATE portal_links l SET last_access_at = now(), access_count = access_count + 1
		  FROM enterprise e, customers c
		 WHERE l.token_hash = $1 AND l.revoked_at IS NULL AND l.expires_at > now()
		   AND e.id = l.enterprise_id AND c.enterprise_id = l.enterprise_id AND c.code = l.customer_code AND c.is_active
		RETURNING l.id, l.enterprise_id, e.code, COALESCE(e.name,''), l.customer_code,
		          COALESCE(NULLIF(c.trade_name,''), c.name, ''), l.expires_at,
		          (SELECT u.id FROM users u JOIN user_enterprises ue ON ue.user_id = u.id AND ue.enterprise_id = e.id
		            WHERE u.is_active AND (UPPER(u.role) = 'ADMIN' OR UPPER(COALESCE(ue.role,'')) = 'ADMIN') ORDER BY u.created_at LIMIT 1)`, hash).
		Scan(&s.LinkID, &s.EnterpriseID, &s.EnterpriseCode, &s.Empresa, &s.CustomerCode, &s.Cliente, &s.ExpiraEm, &ator)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, portal_uc.ErrNaoEncontrado
	}
	if err != nil {
		return nil, fmt.Errorf("lendo o link do portal: %w", err)
	}
	if ator != nil {
		s.Ator = *ator
	}
	return s, nil
}

// PendentesDoPortal: rascunhos do portal deste cliente ainda não confirmados
// (total) e, se houver, o número do que repete o mesmo pedido de origem.
func (r *Repo) PendentesDoPortal(ctx context.Context, s *portal_uc.Sessao, origem int64) (int, int64, error) {
	var total int
	var mesmo int64
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*),
		       COALESCE(MAX(so.order_number) FILTER (WHERE EXISTS (SELECT 1 FROM portal_events ev
		            WHERE ev.enterprise_id = $3 AND ev.acao = 'PEDIDO_REPETIDO' AND ev.sales_order_code = so.code AND ev.pedido_origem = $4)), 0)
		  FROM sales_orders so
		 WHERE so.enterprise_code = $1 AND so.customer_code = $2 AND so.status = 'R' AND so.origin = 'PORTAL'`,
		s.EnterpriseCode, s.CustomerCode, s.EnterpriseID, origem).Scan(&total, &mesmo)
	if err != nil {
		return 0, 0, fmt.Errorf("conferindo os pedidos do portal em aberto: %w", err)
	}
	return total, mesmo, nil
}

func (r *Repo) Evento(ctx context.Context, s *portal_uc.Sessao, acao, detalhe string, pedido, origem *int64, ip, agente string) error {
	_, err := r.pool.Exec(ctx, `INSERT INTO portal_events (enterprise_id, link_id, acao, detalhe, sales_order_code, pedido_origem, ip, user_agent)
		VALUES ($1,$2,$3,NULLIF($4,''),$5,$6,NULLIF($7,''),NULLIF(left($8,300),''))`, s.EnterpriseID, s.LinkID, acao, detalhe, pedido, origem, ip, agente)
	if err != nil {
		return fmt.Errorf("registrando o evento do portal: %w", err)
	}
	return nil
}

// Pedidos: os 60 pedidos mais recentes do cliente do link, com itens, notas,
// títulos e entregas. Toda leitura filtra pela empresa E pelo cliente do link.
func (r *Repo) Pedidos(ctx context.Context, s *portal_uc.Sessao) ([]portal_uc.Pedido, error) {
	rows, err := r.pool.Query(ctx, `SELECT so.code, so.order_number, so.status, so.emission_date, so.delivery_date,
		       COALESCE(so.total_net, 0), COALESCE(so.customer_po_number,''), COALESCE(so.origin,''),
		       COALESCE(pc.description,''), COALESCE(so.freight_type,''), COALESCE(ca.description,''),
		       COALESCE(NULLIF(concat_ws(', ', NULLIF(concat_ws(' ', a.street, a.number),''), a.neighborhood, NULLIF(concat_ws('/', a.city, a.uf),'')), ''), ''),
		       COALESCE(so.obs_customer,'')
		  FROM sales_orders so
		  LEFT JOIN payment_conditions pc ON pc.code = so.payment_term_code AND pc.enterprise_id = $3
		  LEFT JOIN carriers ca ON ca.code = so.carrier_code AND ca.enterprise_id = $3
		  LEFT JOIN customers c ON c.code = so.customer_code AND c.enterprise_id = $3
		  LEFT JOIN customer_addresses a ON a.id = so.shipping_address_code AND a.customer_id = c.id
		 WHERE so.enterprise_code = $1 AND so.customer_code = $2
		   -- (o pedido do portal que a empresa recusou aparece como não confirmado,
		   -- em vez de sumir sem explicação)
		   -- pedido de verdade (orçamento é da fase 4) e, dos rascunhos, só os que o
		   -- próprio cliente pediu pelo portal: o rascunho do vendedor não é dele ainda
		   AND (so.status IN ('P','A','F') OR (so.status IN ('R','CANCELLED') AND so.origin = 'PORTAL'))
		 ORDER BY so.emission_date DESC NULLS LAST, so.code DESC LIMIT 60`, s.EnterpriseCode, s.CustomerCode, s.EnterpriseID)
	if err != nil {
		return nil, fmt.Errorf("lendo os pedidos do cliente: %w", err)
	}
	pedidos, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (portal_uc.Pedido, error) {
		var p portal_uc.Pedido
		err := row.Scan(&p.Code, &p.Numero, &p.Situacao, &p.Emissao, &p.Entrega, &p.Total, &p.PedidoDoCliente, &p.Origem,
			&p.Condicao, &p.Frete, &p.Transportadora, &p.LocalEntrega, &p.Observacao)
		return p, err
	})
	if err != nil || len(pedidos) == 0 {
		return pedidos, err
	}
	idx := map[int64]int{}
	codigos := make([]int64, len(pedidos))
	for i, p := range pedidos {
		idx[p.Code], codigos[i] = i, p.Code
	}
	lr, err := r.pool.Query(ctx, `SELECT l.sales_order_code, COALESCE(NULLIF(i.business_code,''), l.item_code::text),
		       COALESCE(NULLIF(i.name,''), i.pdm_description_technique, ''), COALESCE(l.mask,''), COALESCE(l.sales_uom,''),
		       l.requested_qty, COALESCE(l.attended_qty,0), COALESCE(l.unit_price,0), COALESCE(l.total_net,0), l.delivery_date
		  FROM sales_order_items l
		  JOIN sales_orders so ON so.code = l.sales_order_code AND so.enterprise_code = $2 AND so.customer_code = $3
		  LEFT JOIN items i ON i.code = l.item_code
		 WHERE l.sales_order_code = ANY($1) AND l.is_active AND COALESCE(l.status,'') <> 'CANCELLED'
		 ORDER BY l.sales_order_code, l.sequence`, codigos, s.EnterpriseCode, s.CustomerCode)
	if err != nil {
		return nil, fmt.Errorf("lendo os itens dos pedidos: %w", err)
	}
	for lr.Next() {
		var code int64
		var l portal_uc.Linha
		if err := lr.Scan(&code, &l.Item, &l.Descricao, &l.Mascara, &l.Unidade, &l.Quantidade, &l.Atendido, &l.PrecoUnitario, &l.Total, &l.Entrega); err != nil {
			lr.Close()
			return nil, err
		}
		pedidos[idx[code]].Linhas = append(pedidos[idx[code]].Linhas, l)
	}
	lr.Close()
	if err := lr.Err(); err != nil {
		return nil, err
	}
	// Rascunho do portal ainda sem itens montados: mostra o que o cliente pediu
	// (as linhas do pedido de origem), com o preço "a confirmar".
	var vazios []int64
	for _, p := range pedidos {
		if p.Situacao == "R" && p.Origem == "PORTAL" && len(p.Linhas) == 0 {
			vazios = append(vazios, p.Code)
		}
	}
	if len(vazios) > 0 {
		pr, err := r.pool.Query(ctx, `SELECT ev.sales_order_code, COALESCE(NULLIF(i.business_code,''), l.item_code::text),
			       COALESCE(NULLIF(i.name,''), i.pdm_description_technique, ''), COALESCE(l.mask,''), COALESCE(l.sales_uom,''), l.requested_qty
			  FROM portal_events ev
			  JOIN sales_orders o ON o.code = ev.pedido_origem AND o.enterprise_code = $3 AND o.customer_code = $4
			  JOIN sales_order_items l ON l.sales_order_code = o.code AND l.is_active AND COALESCE(l.status,'') <> 'CANCELLED'
			  LEFT JOIN items i ON i.code = l.item_code
			 WHERE ev.enterprise_id = $1 AND ev.acao = 'PEDIDO_REPETIDO' AND ev.sales_order_code = ANY($2)
			 ORDER BY ev.sales_order_code, l.sequence`, s.EnterpriseID, vazios, s.EnterpriseCode, s.CustomerCode)
		if err != nil {
			return nil, fmt.Errorf("lendo os itens pedidos pelo portal: %w", err)
		}
		for pr.Next() {
			var code int64
			l := portal_uc.Linha{APreco: true}
			if err := pr.Scan(&code, &l.Item, &l.Descricao, &l.Mascara, &l.Unidade, &l.Quantidade); err != nil {
				pr.Close()
				return nil, err
			}
			pedidos[idx[code]].Linhas = append(pedidos[idx[code]].Linhas, l)
		}
		pr.Close()
		if err := pr.Err(); err != nil {
			return nil, err
		}
	}
	nr, err := r.pool.Query(ctx, `SELECT x.sales_order_code, x.id, x.numero_nf, COALESCE(x.serie,''), COALESCE(x.chave_acesso,''), x.data_emissao, x.valor_total, x.status,
		       (x.danfe_path IS NOT NULL OR x.xml_path IS NOT NULL OR x.focus_ref IS NOT NULL)
		  FROM fiscal_exits x JOIN sales_orders so ON so.code = x.sales_order_code AND so.enterprise_code = $3 AND so.customer_code = $4
		 WHERE x.enterprise_id = $1 AND x.sales_order_code = ANY($2) AND x.status IN ('AUTHORIZED','CANCELLED')
		 ORDER BY x.data_emissao`, s.EnterpriseID, codigos, s.EnterpriseCode, s.CustomerCode)
	if err != nil {
		return nil, fmt.Errorf("lendo as notas dos pedidos: %w", err)
	}
	for nr.Next() {
		var code int64
		var n portal_uc.Nota
		if err := nr.Scan(&code, &n.ID, &n.Numero, &n.Serie, &n.Chave, &n.Emissao, &n.Valor, &n.Situacao, &n.Documentos); err != nil {
			nr.Close()
			return nil, err
		}
		n.Documentos = n.Documentos && n.Situacao == "AUTHORIZED"
		pedidos[idx[code]].Notas = append(pedidos[idx[code]].Notas, n)
	}
	nr.Close()
	if err := nr.Err(); err != nil {
		return nil, err
	}
	tr, err := r.pool.Query(ctx, `SELECT cr.sales_order_id, COALESCE(cr.numero_documento,''),
		       CASE WHEN cr.parcela_total > 1 THEN cr.parcela_numero || '/' || cr.parcela_total ELSE '' END,
		       cr.data_vencimento, cr.valor_bruto, GREATEST(cr.valor_bruto - COALESCE(cr.valor_recebido,0) - COALESCE(cr.desconto,0), 0),
		       cr.status, COALESCE(cr.linha_digitavel,''), COALESCE(cr.chave_pix_gerada,'')
		  FROM contas_receber cr JOIN sales_orders so ON so.code = cr.sales_order_id AND so.enterprise_code = $3 AND so.customer_code = $4
		 WHERE cr.enterprise_id = $1 AND cr.sales_order_id = ANY($2) AND cr.is_active AND UPPER(cr.status) <> 'CANCELADO'
		 ORDER BY cr.data_vencimento, cr.parcela_numero`, s.EnterpriseID, codigos, s.EnterpriseCode, s.CustomerCode)
	if err != nil {
		return nil, fmt.Errorf("lendo os títulos dos pedidos: %w", err)
	}
	for tr.Next() {
		var code int64
		var t portal_uc.Titulo
		if err := tr.Scan(&code, &t.Documento, &t.Parcela, &t.Vencimento, &t.Valor, &t.Saldo, &t.Situacao, &t.LinhaDigitavel, &t.PixCopiaECola); err != nil {
			tr.Close()
			return nil, err
		}
		pedidos[idx[code]].Titulos = append(pedidos[idx[code]].Titulos, t)
	}
	tr.Close()
	if err := tr.Err(); err != nil {
		return nil, err
	}
	er, err := r.pool.Query(ctx, `SELECT sh.sales_order_code, sh.code, sh.status, sh.shipped_at,
		       COALESCE(NULLIF(sup.trade_name,''), sup.name, ''), sh.nfe_number
		  FROM shipments sh JOIN sales_orders so ON so.code = sh.sales_order_code AND so.enterprise_code = $3 AND so.customer_code = $4
		  LEFT JOIN shipping_carriers sc ON sc.id = sh.shipping_carrier_id AND sc.enterprise_id = sh.enterprise_id
		  LEFT JOIN suppliers sup ON sup.code = sc.supplier_code
		 WHERE sh.enterprise_id = $1 AND sh.sales_order_code = ANY($2) AND sh.status <> 'CANCELLED'
		 ORDER BY sh.code`, s.EnterpriseID, codigos, s.EnterpriseCode, s.CustomerCode)
	if err != nil {
		return nil, fmt.Errorf("lendo as entregas dos pedidos: %w", err)
	}
	defer er.Close()
	for er.Next() {
		var code int64
		var e portal_uc.Entrega
		if err := er.Scan(&code, &e.Romaneio, &e.Situacao, &e.DespachadoEm, &e.Transportadora, &e.NotaNumero); err != nil {
			return nil, err
		}
		if e.Transportadora == "" {
			e.Transportadora = pedidos[idx[code]].Transportadora
		}
		pedidos[idx[code]].Entregas = append(pedidos[idx[code]].Entregas, e)
	}
	return pedidos, er.Err()
}

// NotaDoCliente: a nota autorizada é de um pedido do cliente do link?
func (r *Repo) NotaDoCliente(ctx context.Context, s *portal_uc.Sessao, nota int64) (int64, bool, error) {
	var numero int64
	err := r.pool.QueryRow(ctx, `SELECT x.numero_nf FROM fiscal_exits x
		JOIN sales_orders so ON so.code = x.sales_order_code AND so.enterprise_code = $3 AND so.customer_code = $4
		WHERE x.id = $1 AND x.enterprise_id = $2 AND x.status = 'AUTHORIZED'`, nota, s.EnterpriseID, s.EnterpriseCode, s.CustomerCode).Scan(&numero)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, false, nil
	}
	return numero, err == nil, err
}

// PedidoDoCliente: o pedido é do cliente do link?
// PedidoDoCliente: só pedido de verdade se repete (o que o cliente vê como
// confirmado, em análise ou faturado). Orçamento, rascunho do vendedor e o
// próprio pedido do portal ainda aberto não — antes passavam.
func (r *Repo) PedidoDoCliente(ctx context.Context, s *portal_uc.Sessao, code int64) (bool, error) {
	var ok bool
	err := r.pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM sales_orders WHERE code = $1 AND enterprise_code = $2 AND customer_code = $3
		   AND status IN ('P','A','F'))`,
		code, s.EnterpriseCode, s.CustomerCode).Scan(&ok)
	return ok, err
}

// AvisarEmpresa registra o alerta COMERCIAL_PEDIDO_PORTAL (Central de Alertas)
// e devolve o e-mail do representante do pedido, para o aviso direto.
func (r *Repo) AvisarEmpresa(ctx context.Context, s *portal_uc.Sessao, novo, numero, original int64, observacao string) (string, error) {
	payload, _ := json.Marshal(map[string]any{
		"descricao":       fmt.Sprintf("%s pediu pelo portal a repetição do pedido %d: rascunho %d aguarda confirmação", s.Cliente, original, numero),
		"cliente":         s.CustomerCode,
		"pedido":          numero,
		"pedido_original": original,
		"observacao":      observacao,
	})
	if _, err := r.pool.Exec(ctx, `INSERT INTO notification_outbox(enterprise_id,event_key,event_version,aggregate_type,aggregate_internal_id,aggregate_public_id,payload,deduplication_key,originator_user_id)
		VALUES($1,'COMERCIAL_PEDIDO_PORTAL',1,'PEDIDO_VENDA',$2,$3,$4,$5,$6) ON CONFLICT(enterprise_id,event_key,deduplication_key) DO NOTHING`,
		s.EnterpriseID, fmt.Sprint(novo), fmt.Sprint(numero), payload, fmt.Sprintf("portal:pedido:%d", novo), s.Ator); err != nil {
		return "", fmt.Errorf("registrando o alerta do pedido do portal: %w", err)
	}
	var email *string
	err := r.pool.QueryRow(ctx, `SELECT NULLIF(TRIM(rep.main_email),'') FROM sales_orders so
		JOIN representatives rep ON rep.code = so.representative_code AND rep.enterprise_code = so.enterprise_code
		WHERE so.code = $1 AND so.enterprise_code = $2`, novo, s.EnterpriseCode).Scan(&email)
	if errors.Is(err, pgx.ErrNoRows) || email == nil {
		return "", nil
	}
	return *email, err
}
