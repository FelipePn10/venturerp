package purchase_order

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/purchase_order_uc"
	fiscalrepo "github.com/FelipePn10/panossoerp/internal/domain/fiscal/repository"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"
)

// ConsultasPG implementa as consultas de compras que cruzam outros módulos.
// Convenções de empresa: pedido usa enterprise_code (tenant.Code); itens,
// fornecedores, notas de entrada e condições de pagamento usam enterprise_id
// (tenant.ID).
type ConsultasPG struct {
	db     *pgxpool.Pool
	Fiscal fiscalrepo.FiscalRepository
}

func NewConsultasPG(pool *pgxpool.Pool, fiscal fiscalrepo.FiscalRepository) *ConsultasPG {
	return &ConsultasPG{db: pool, Fiscal: fiscal}
}

var _ purchase_order_uc.ConsultasCompras = (*ConsultasPG)(nil)

func empresas(ctx context.Context) (code, id int64, err error) {
	if code, err = tenant.Code(ctx); err != nil {
		return 0, 0, err
	}
	id, err = tenant.ID(ctx)
	return code, id, err
}

func datePtr(d pgtype.Date) *time.Time {
	if !d.Valid {
		return nil
	}
	t := d.Time
	return &t
}

func (c *ConsultasPG) LinhasEmAberto(ctx context.Context, f purchase_order_uc.FiltroAcompanhamento) ([]purchase_order_uc.LinhaEmAberto, error) {
	code, id, err := empresas(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := c.db.Query(ctx, `
		SELECT po.code, po.order_number, poi.code, poi.sequence, poi.item_code, COALESCE(it.name, ''),
		       COALESCE(po.supplier_code, 0), COALESCE(s.name, ''),
		       poi.requested_qty::float8, poi.received_qty::float8,
		       GREATEST(poi.requested_qty - poi.received_qty - poi.cancelled_qty, 0)::float8,
		       COALESCE(poi.delivery_date, po.delivery_date), poi.promised_date,
		       f.id, f.data_prometida, COALESCE(f.contato, ''), COALESCE(f.observacao, ''), f.registrado_em, COALESCE(u.name, '')
		  FROM purchase_order_items poi
		  JOIN purchase_orders po ON po.code = poi.purchase_order_code
		  LEFT JOIN items it ON it.code = poi.item_code AND it.enterprise_id = $2
		  LEFT JOIN suppliers s ON s.code = po.supplier_code
		  LEFT JOIN LATERAL (
		        SELECT x.id, x.data_prometida, x.contato, x.observacao, x.registrado_em, x.registrado_por
		          FROM purchase_order_item_followups x
		         WHERE x.purchase_order_item_code = poi.code AND x.enterprise_code = $1
		         ORDER BY x.registrado_em DESC, x.id DESC LIMIT 1) f ON TRUE
		  LEFT JOIN users u ON u.id = f.registrado_por
		 WHERE po.enterprise_code = $1 AND po.is_active AND po.status IN ('APPROVED', 'PARTIAL')
		   AND poi.is_active AND poi.status IN ('OPEN', 'PARTIAL')
		   AND poi.requested_qty - poi.received_qty - poi.cancelled_qty > 0.0001
		   AND ($3::bigint IS NULL OR po.supplier_code = $3)
		 ORDER BY COALESCE(poi.promised_date, poi.delivery_date, po.delivery_date) NULLS LAST, po.code, poi.sequence
		 LIMIT 2000`, code, id, f.SupplierCode)
	if err != nil {
		return nil, fmt.Errorf("lendo linhas em aberto: %w", err)
	}
	defer rows.Close()
	out := []purchase_order_uc.LinhaEmAberto{}
	for rows.Next() {
		var l purchase_order_uc.LinhaEmAberto
		var entrega, prometida, fData pgtype.Date
		var fID *int64
		var fContato, fObs, fUser string
		var fEm pgtype.Timestamptz
		if err := rows.Scan(&l.PurchaseOrderCode, &l.OrderNumber, &l.LineCode, &l.Sequence, &l.ItemCode, &l.ItemName,
			&l.SupplierCode, &l.SupplierName, &l.RequestedQty, &l.ReceivedQty, &l.Saldo, &entrega, &prometida,
			&fID, &fData, &fContato, &fObs, &fEm, &fUser); err != nil {
			return nil, err
		}
		l.DeliveryDate, l.PromisedDate = datePtr(entrega), datePtr(prometida)
		if fID != nil {
			l.UltimoContato = &purchase_order_uc.Followup{ID: *fID, PurchaseOrderCode: l.PurchaseOrderCode, LineCode: l.LineCode,
				DataPrometida: datePtr(fData), Contato: fContato, Observacao: fObs, RegistradoEm: fEm.Time, RegistradoPor: fUser}
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

func (c *ConsultasPG) RegistrarFollowup(ctx context.Context, f purchase_order_uc.Followup, por uuid.UUID) (*purchase_order_uc.Followup, error) {
	code, err := tenant.Code(ctx)
	if err != nil {
		return nil, err
	}
	tx, err := c.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var data pgtype.Date
	if f.DataPrometida != nil {
		data = pgtype.Date{Time: *f.DataPrometida, Valid: true}
	}
	vazio := func(s string) *string {
		if s == "" {
			return nil
		}
		return &s
	}
	err = tx.QueryRow(ctx, `
		INSERT INTO purchase_order_item_followups
		       (enterprise_code, purchase_order_code, purchase_order_item_code, data_prometida, contato, observacao, registrado_por)
		SELECT $1, poi.purchase_order_code, poi.code, $4, $5, $6, $7
		  FROM purchase_order_items poi
		  JOIN purchase_orders po ON po.code = poi.purchase_order_code AND po.enterprise_code = $1
		 WHERE poi.code = $3 AND poi.purchase_order_code = $2
		RETURNING id, registrado_em`, code, f.PurchaseOrderCode, f.LineCode, data, vazio(f.Contato), vazio(f.Observacao), por).
		Scan(&f.ID, &f.RegistradoEm)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("a linha %d não pertence ao pedido %d", f.LineCode, f.PurchaseOrderCode)
	}
	if err != nil {
		return nil, fmt.Errorf("gravando acompanhamento: %w", err)
	}
	if f.DataPrometida != nil {
		if _, err := tx.Exec(ctx, `UPDATE purchase_order_items SET promised_date = $2, updated_at = NOW() WHERE code = $1`, f.LineCode, data); err != nil {
			return nil, fmt.Errorf("gravando data prometida: %w", err)
		}
	}
	if err := tx.QueryRow(ctx, `SELECT COALESCE((SELECT name FROM users WHERE id = $1), '')`, por).Scan(&f.RegistradoPor); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return &f, nil
}

func (c *ConsultasPG) Followups(ctx context.Context, lineCode int64) ([]purchase_order_uc.Followup, error) {
	code, err := tenant.Code(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := c.db.Query(ctx, `
		SELECT f.id, f.purchase_order_code, f.purchase_order_item_code, f.data_prometida, COALESCE(f.contato, ''),
		       COALESCE(f.observacao, ''), f.registrado_em, COALESCE(u.name, '')
		  FROM purchase_order_item_followups f
		  LEFT JOIN users u ON u.id = f.registrado_por
		 WHERE f.enterprise_code = $1 AND f.purchase_order_item_code = $2
		 ORDER BY f.registrado_em DESC, f.id DESC`, code, lineCode)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []purchase_order_uc.Followup{}
	for rows.Next() {
		var f purchase_order_uc.Followup
		var d pgtype.Date
		if err := rows.Scan(&f.ID, &f.PurchaseOrderCode, &f.LineCode, &d, &f.Contato, &f.Observacao, &f.RegistradoEm, &f.RegistradoPor); err != nil {
			return nil, err
		}
		f.DataPrometida = datePtr(d)
		out = append(out, f)
	}
	return out, rows.Err()
}

func (c *ConsultasPG) NotasDoPedido(ctx context.Context, orderCode int64) ([]purchase_order_uc.NotaDaLinha, error) {
	code, id, err := empresas(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := c.db.Query(ctx, `
		SELECT fi.purchase_order_item_code, fe.id, fe.numero_nf, COALESCE(fe.serie, ''), fe.data_emissao, fe.data_entrada,
		       fe.status, fi.quantity::float8, COALESCE(fi.uom, ''), fi.unit_price::float8, fi.total_price::float8,
		       COALESCE(fe.razao_social_emitente, '')
		  FROM fiscal_entry_items fi
		  JOIN fiscal_entries fe ON fe.id = fi.fiscal_entry_id AND fe.enterprise_id = $2
		  JOIN purchase_order_items poi ON poi.code = fi.purchase_order_item_code AND poi.purchase_order_code = $3
		  JOIN purchase_orders po ON po.code = poi.purchase_order_code AND po.enterprise_code = $1
		 ORDER BY fe.data_entrada, fe.id, fi.sequence`, code, id, orderCode)
	if err != nil {
		return nil, fmt.Errorf("lendo notas do pedido: %w", err)
	}
	defer rows.Close()
	out := []purchase_order_uc.NotaDaLinha{}
	for rows.Next() {
		var n purchase_order_uc.NotaDaLinha
		var emissao, entrada pgtype.Date
		if err := rows.Scan(&n.LineCode, &n.FiscalEntryID, &n.NumeroNF, &n.Serie, &emissao, &entrada, &n.Status,
			&n.Quantidade, &n.Unidade, &n.ValorUnitario, &n.ValorTotal, &n.SupplierName); err != nil {
			return nil, err
		}
		n.DataEmissao, n.DataEntrada = emissao.Time, entrada.Time
		out = append(out, n)
	}
	return out, rows.Err()
}

func (c *ConsultasPG) ComprasDoItem(ctx context.Context, itemCode int64, desde time.Time, limite int) ([]purchase_order_uc.CompraDoItem, error) {
	id, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := c.db.Query(ctx, `
		SELECT fe.id, fe.numero_nf, fe.data_entrada, fe.supplier_code,
		       COALESCE(s.name, fe.razao_social_emitente, ''), fi.quantity::float8, COALESCE(fi.uom, ''), fi.unit_price::float8,
		       q.qtd::float8, q.custo::float8
		  FROM fiscal_entry_items fi
		  JOIN fiscal_entries fe ON fe.id = fi.fiscal_entry_id AND fe.enterprise_id = $1
		  LEFT JOIN suppliers s ON s.code = fe.supplier_code
		  CROSS JOIN LATERAL (SELECT COALESCE(NULLIF(fi.quantidade_estoque, 0), fi.quantity * COALESCE(NULLIF(fi.fator_conversao, 0), 1)) AS qtd,
		                             COALESCE(fi.custo_aquisicao, fi.valor_contabil, fi.total_price) AS custo) q
		 WHERE fi.item_code = $2 AND fe.status IN ('APPROVED', 'WRITTEN_OFF') AND fe.data_entrada >= $3::date
		 ORDER BY fe.data_entrada DESC, fe.id DESC, fi.sequence
		 LIMIT $4`, id, itemCode, desde, limite)
	if err != nil {
		return nil, fmt.Errorf("lendo compras do item: %w", err)
	}
	defer rows.Close()
	out := []purchase_order_uc.CompraDoItem{}
	for rows.Next() {
		var x purchase_order_uc.CompraDoItem
		var entrada pgtype.Date
		if err := rows.Scan(&x.FiscalEntryID, &x.NumeroNF, &entrada, &x.SupplierCode, &x.SupplierName, &x.Quantidade, &x.Unidade,
			&x.PrecoNota, &x.QtdEstoque, &x.CustoAquisicao); err != nil {
			return nil, err
		}
		x.DataEntrada = entrada.Time
		if x.QtdEstoque > 0 {
			x.CustoEstoque = x.CustoAquisicao / x.QtdEstoque
		}
		out = append(out, x)
	}
	return out, rows.Err()
}

func (c *ConsultasPG) UltimoPedidoDoItem(ctx context.Context, itemCode int64) (*purchase_order_uc.UltimoPedidoDoItem, error) {
	code, err := tenant.Code(ctx)
	if err != nil {
		return nil, err
	}
	var u purchase_order_uc.UltimoPedidoDoItem
	err = c.db.QueryRow(ctx, `
		SELECT po.code, po.order_number, po.emission_date, COALESCE(s.name, ''), poi.unit_price::float8, COALESCE(poi.purchase_uom, '')
		  FROM purchase_order_items poi
		  JOIN purchase_orders po ON po.code = poi.purchase_order_code AND po.enterprise_code = $1
		  LEFT JOIN suppliers s ON s.code = po.supplier_code
		 WHERE poi.item_code = $2 AND poi.is_active AND po.status <> 'CANCELLED'
		 ORDER BY po.emission_date DESC, po.code DESC, poi.sequence DESC
		 LIMIT 1`, code, itemCode).Scan(&u.PurchaseOrderCode, &u.OrderNumber, &u.EmissionDate, &u.SupplierName, &u.UnitPrice, &u.PurchaseUOM)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("lendo último pedido do item: %w", err)
	}
	return &u, nil
}

// Destinatarios: contatos marcados para receber pedido de compra vêm primeiro
// e sugeridos; depois os demais contatos e os e-mails gerais do fornecedor.
// Sem contato marcado, o primeiro e-mail geral é o sugerido.
func (c *ConsultasPG) Destinatarios(ctx context.Context, supplierCode int64) ([]purchase_order_uc.Destinatario, error) {
	id, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := c.db.Query(ctx, `
		SELECT e.value, sc.name,
		       CASE WHEN COALESCE(btrim(sc.purchase_order_tag), '') <> '' THEN 'CONTATO_PEDIDO' ELSE 'CONTATO' END,
		       CASE WHEN COALESCE(btrim(sc.purchase_order_tag), '') <> '' THEN 0 ELSE 1 END AS ordem, sc.ranking, e.ranking
		  FROM supplier_contact_emails e
		  JOIN supplier_contacts sc ON sc.id = e.contact_id AND sc.is_active
		  JOIN suppliers s ON s.id = sc.supplier_id AND s.code = $1 AND s.enterprise_id = $2
		UNION ALL
		SELECT se.email, '', 'FORNECEDOR', 2, 0, se.ranking
		  FROM supplier_emails se
		  JOIN suppliers s ON s.id = se.supplier_id AND s.code = $1 AND s.enterprise_id = $2
		 WHERE se.is_active
		 ORDER BY 4, 5, 6`, supplierCode, id)
	if err != nil {
		return nil, fmt.Errorf("lendo e-mails do fornecedor: %w", err)
	}
	defer rows.Close()
	out := []purchase_order_uc.Destinatario{}
	vistos := map[string]bool{}
	temMarcado := false
	for rows.Next() {
		var d purchase_order_uc.Destinatario
		var ordem, r1, r2 int
		if err := rows.Scan(&d.Email, &d.Nome, &d.Origem, &ordem, &r1, &r2); err != nil {
			return nil, err
		}
		d.Email = strings.ToLower(strings.TrimSpace(d.Email))
		if d.Email == "" || vistos[d.Email] {
			continue
		}
		vistos[d.Email] = true
		if d.Origem == "CONTATO_PEDIDO" {
			d.Sugerido, temMarcado = true, true
		}
		out = append(out, d)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if !temMarcado {
		for i := range out {
			if out[i].Origem == "FORNECEDOR" {
				out[i].Sugerido = true
				break
			}
		}
		if len(out) > 0 && !out[0].Sugerido {
			sugerido := false
			for _, d := range out {
				sugerido = sugerido || d.Sugerido
			}
			if !sugerido {
				out[0].Sugerido = true
			}
		}
	}
	return out, nil
}

func (c *ConsultasPG) RegistrarEnvio(ctx context.Context, e purchase_order_uc.Envio) (*purchase_order_uc.Envio, error) {
	code, err := tenant.Code(ctx)
	if err != nil {
		return nil, err
	}
	var erro *string
	if e.Erro != "" {
		erro = &e.Erro
	}
	err = c.db.QueryRow(ctx, `
		INSERT INTO purchase_order_envios (enterprise_code, purchase_order_code, enviado_em, enviado_por, destinatarios, assunto, situacao, erro)
		SELECT $1, po.code, $3, $4, $5, $6, $7, $8 FROM purchase_orders po WHERE po.code = $2 AND po.enterprise_code = $1
		RETURNING id`, code, e.PurchaseOrderCode, e.EnviadoEm, e.EnviadoPor, e.Destinatarios, e.Assunto, e.Situacao, erro).Scan(&e.ID)
	if err != nil {
		return nil, fmt.Errorf("registrando envio do pedido: %w", err)
	}
	if e.EnviadoPor != nil {
		_ = c.db.QueryRow(ctx, `SELECT COALESCE((SELECT name FROM users WHERE id = $1), '')`, *e.EnviadoPor).Scan(&e.EnviadoPorNome)
	}
	return &e, nil
}

func (c *ConsultasPG) Envios(ctx context.Context, orderCode int64) ([]purchase_order_uc.Envio, error) {
	code, err := tenant.Code(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := c.db.Query(ctx, `
		SELECT e.id, e.purchase_order_code, e.enviado_em, COALESCE(u.name, ''), e.destinatarios, e.assunto, e.situacao, COALESCE(e.erro, '')
		  FROM purchase_order_envios e
		  LEFT JOIN users u ON u.id = e.enviado_por
		 WHERE e.enterprise_code = $1 AND e.purchase_order_code = $2
		 ORDER BY e.enviado_em DESC, e.id DESC`, code, orderCode)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []purchase_order_uc.Envio{}
	for rows.Next() {
		var e purchase_order_uc.Envio
		if err := rows.Scan(&e.ID, &e.PurchaseOrderCode, &e.EnviadoEm, &e.EnviadoPorNome, &e.Destinatarios, &e.Assunto, &e.Situacao, &e.Erro); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func (c *ConsultasPG) DadosDocumento(ctx context.Context, orderCode int64) (*purchase_order_uc.DadosDocumento, error) {
	code, id, err := empresas(ctx)
	if err != nil {
		return nil, err
	}
	d := &purchase_order_uc.DadosDocumento{}
	if c.Fiscal != nil {
		cfg, err := c.Fiscal.GetFiscalConfig(ctx)
		var nf *errorsuc.NotFoundError
		if err != nil && !errors.As(err, &nf) {
			return nil, fmt.Errorf("lendo dados da empresa: %w", err)
		}
		if cfg != nil {
			end := strings.TrimSpace(cfg.Logradouro)
			if cfg.Numero != "" {
				end += ", " + cfg.Numero
			}
			if cfg.Bairro != "" {
				end += " — " + cfg.Bairro
			}
			d.Empresa = purchase_order_uc.Parte{Nome: cfg.RazaoSocial, CNPJCPF: mascaraDocumento(cfg.CnpjEmpresa), IE: derefStr(cfg.IEEmpresa),
				Endereco: end, Cidade: cfg.Municipio, UF: cfg.UFEmpresa, CEP: cfg.CEP, Telefone: derefStr(cfg.Telefone), Email: cfg.Email}
			d.Logo, d.CorMarca = cfg.Logo, derefStr(cfg.BrandColor)
		}
	}
	var supplierCode, carrierCode, termCode *int64
	var createdBy uuid.UUID
	if err := c.db.QueryRow(ctx, `SELECT supplier_code, carrier_code, payment_term_code, created_by FROM purchase_orders WHERE code = $1 AND enterprise_code = $2`,
		orderCode, code).Scan(&supplierCode, &carrierCode, &termCode, &createdBy); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("pedido de compra %d não encontrado", orderCode)
		}
		return nil, err
	}
	if supplierCode != nil {
		var f purchase_order_uc.Parte
		var fantasia, ie, rua, num, comp, bairro, cidade, uf, cep, email *string
		err := c.db.QueryRow(ctx, `
			SELECT s.name, s.trade_name, s.document_number, s.state_registration,
			       a.street, a.number, a.complement, a.neighborhood, a.city, a.uf, a.zip_code,
			       (SELECT se.email FROM supplier_emails se WHERE se.supplier_id = s.id AND se.is_active ORDER BY se.ranking LIMIT 1)
			  FROM suppliers s
			  LEFT JOIN LATERAL (SELECT * FROM supplier_addresses sa WHERE sa.supplier_id = s.id ORDER BY sa.is_default DESC, sa.id LIMIT 1) a ON TRUE
			 WHERE s.code = $1 AND s.enterprise_id = $2`, *supplierCode, id).
			Scan(&f.Nome, &fantasia, &f.CNPJCPF, &ie, &rua, &num, &comp, &bairro, &cidade, &uf, &cep, &email)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("lendo fornecedor do pedido: %w", err)
		}
		f.Fantasia, f.IE, f.Email = derefStr(fantasia), derefStr(ie), derefStr(email)
		f.CNPJCPF = mascaraDocumento(f.CNPJCPF)
		end := derefStr(rua)
		if n := derefStr(num); n != "" {
			end += ", " + n
		}
		if cp := derefStr(comp); cp != "" {
			end += " " + cp
		}
		if b := derefStr(bairro); b != "" {
			end += " — " + b
		}
		f.Endereco, f.Cidade, f.UF, f.CEP = strings.TrimSpace(end), derefStr(cidade), derefStr(uf), derefStr(cep)
		d.Fornecedor = f
	}
	if carrierCode != nil {
		_ = c.db.QueryRow(ctx, `SELECT description FROM carriers WHERE code = $1`, *carrierCode).Scan(&d.Transportadora)
	}
	if termCode != nil {
		// O pedido guarda o CÓDIGO da condição (é o que a tela escolhe).
		_ = c.db.QueryRow(ctx, `SELECT description FROM payment_conditions WHERE code = $1 AND enterprise_id = $2`, *termCode, id).Scan(&d.Condicao)
	}
	_ = c.db.QueryRow(ctx, `SELECT COALESCE((SELECT name FROM users WHERE id = $1), '')`, createdBy).Scan(&d.Comprador)

	rows, err := c.db.Query(ctx, `
		SELECT poi.sequence, COALESCE(it.business_code, poi.item_code::text), COALESCE(NULLIF(it.name, ''), 'Item ' || poi.item_code),
		       COALESCE(poi.purchase_uom, it.warehouse_unit_of_measurement::text, ''), poi.requested_qty::float8, poi.unit_price::float8,
		       poi.discount_pct::float8, poi.ipi_pct::float8, poi.total_price::float8, COALESCE(poi.promised_date, poi.delivery_date),
		       COALESCE(w.description, ''), COALESCE(poi.notes, ''), poi.status, poi.cancelled_qty::float8
		  FROM purchase_order_items poi
		  LEFT JOIN items it ON it.code = poi.item_code AND it.enterprise_id = $2
		  LEFT JOIN warehouse w ON w.id = poi.warehouse_id AND w.enterprise_id = $2
		 WHERE poi.purchase_order_code = $1 AND poi.is_active
		 ORDER BY poi.sequence`, orderCode, id)
	if err != nil {
		return nil, fmt.Errorf("lendo linhas do pedido: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var l purchase_order_uc.LinhaDocumento
		var entrega pgtype.Date
		if err := rows.Scan(&l.Sequence, &l.ItemCode, &l.Descricao, &l.Unidade, &l.Quantidade, &l.PrecoUnit, &l.DescontoPct, &l.IPIPct,
			&l.Total, &entrega, &l.Almoxarifado, &l.Observacao, &l.Situacao, &l.Cancelada); err != nil {
			return nil, err
		}
		l.Entrega = datePtr(entrega)
		d.Linhas = append(d.Linhas, l)
	}
	return d, rows.Err()
}

func derefStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func mascaraDocumento(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	d := b.String()
	switch len(d) {
	case 14:
		return d[0:2] + "." + d[2:5] + "." + d[5:8] + "/" + d[8:12] + "-" + d[12:14]
	case 11:
		return d[0:3] + "." + d[3:6] + "." + d[6:9] + "-" + d[9:11]
	}
	return s
}
