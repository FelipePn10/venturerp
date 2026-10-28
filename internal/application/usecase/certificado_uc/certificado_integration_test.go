//go:build integration

package certificado_uc_test

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/application/security"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/certificado_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/documento_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/lote_uc"
	docdom "github.com/FelipePn10/panossoerp/internal/domain/quality/documento"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/auth"
	certpdf "github.com/FelipePn10/panossoerp/internal/infrastructure/export/certificado"
	amostragempg "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/amostragem"
	certpg "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/certificado"
	docpg "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/documento"
	lotepg "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/lote"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
	contextkey "github.com/FelipePn10/panossoerp/internal/interfaces/http/context"
)

func dp(s string) *decimal.Decimal {
	d := decimal.RequireFromString(s)
	return &d
}
func sp(s string) *string { return &s }
func bp(b bool) *bool     { return &b }

func TestCertificadoEDocumentos(t *testing.T) {
	pool := testutil.Pool(t)
	bg := context.Background()
	actor := testutil.Actor(t, pool)
	u := testutil.UniqueCode()
	item := 93_000_000 + u%1_000_000
	cliente := 8_000_000 + u%1_000_000
	revisor := uuid.New()
	testutil.Exec(t, pool, `INSERT INTO users(id,name,email,password) VALUES($1,'Revisora',$2,'x')`, revisor, revisor.String()+"@example.test")
	a, b := testutil.DuasEmpresas(t, pool, func(a, b testutil.Empresa) {
		for _, q := range []string{
			`ALTER TABLE quality_certificates DISABLE TRIGGER trg_quality_certificates_congelado`,
			`DELETE FROM quality_certificates WHERE enterprise_id=$1`,
			`ALTER TABLE quality_certificates ENABLE TRIGGER trg_quality_certificates_congelado`,
			`DELETE FROM quality_documents WHERE enterprise_id=$1`,
			`DELETE FROM quality_inspection_lots WHERE enterprise_id=$1`,
			`DELETE FROM receiving_inspection_routes WHERE enterprise_code=(SELECT code FROM enterprise WHERE id=$1)`,
			`DELETE FROM stock_lots WHERE enterprise_id=$1`, `DELETE FROM customers WHERE enterprise_id=$1`,
			`DELETE FROM items WHERE enterprise_id=$1`, `DELETE FROM warehouse WHERE enterprise_id=$1`,
		} {
			if strings.HasPrefix(q, "ALTER") {
				_, _ = pool.Exec(bg, q)
			} else if _, err := pool.Exec(bg, q, a.ID); err != nil {
				t.Logf("limpeza %s: %v", q, err)
			}
		}
		_, _ = pool.Exec(bg, `UPDATE users SET is_active=FALSE WHERE id=$1`, revisor)
	})
	var almox int64
	if err := pool.QueryRow(bg, `INSERT INTO warehouse(code,description,created_by,location,type,disposition,reservations_allowed,enterprise_id)
		VALUES($1,'Almox',$2,'INTERNO','NORMAL',TRUE,TRUE,$3) RETURNING id`, fmt.Sprintf("CQ-%d", u), actor, a.ID).Scan(&almox); err != nil {
		t.Fatal(err)
	}
	testutil.Exec(t, pool, `INSERT INTO items(code,business_code,warehouse_code,created_by,enterprise_id,name) VALUES($1,($1::bigint)::text,$2,$3,$4,'FLANGE DN50')`, item, almox, actor, a.ID)
	testutil.Exec(t, pool, `INSERT INTO customers(code,name,document_number,created_by,enterprise_id) VALUES($1,'METALÚRGICA CLIENTE','11222333000144',$2,$3)`, cliente, actor, a.ID)
	testutil.Exec(t, pool, `INSERT INTO stock_lots(item_code,lot,heat_number,certificate,created_by,enterprise_id) VALUES($1,'L-77','C-4521','USIMINAS 99812',$2,$3)`, item, actor, a.ID)
	var rota int64
	if err := pool.QueryRow(bg, `INSERT INTO receiving_inspection_routes(enterprise_code,basis,item_code,mask,inspection_warehouse_id,valid_from,is_active,created_by)
		VALUES($1,'ITEM',$2,'',$3,$4::date,TRUE,$5) RETURNING id`, a.Code, item, almox, time.Now().AddDate(0, 0, -30), actor).Scan(&rota); err != nil {
		t.Fatal(err)
	}
	testutil.Exec(t, pool, `INSERT INTO receiving_inspection_route_steps(route_id,sequence,inspection_name,kind,appointment_mode,is_required,min_value,max_value)
		VALUES($1,1,'Espessura','VALUE','ALL_MEASUREMENTS',TRUE,9.8,10.2),($1,2,'Acabamento','ATTRIBUTE','STATUS_ONLY',TRUE,NULL,NULL)`, rota)

	ctx := a.Ctx
	lotes := &lote_uc.UseCase{Repo: lotepg.New(pool), Fontes: lotepg.NewFontes(pool), Amostragem: amostragempg.New(pool), Auth: &auth.AuthService{}}
	cert := &certificado_uc.UseCase{Repo: certpg.New(pool), Fontes: certpg.New(pool), Lotes: lotepg.New(pool), Auth: &auth.AuthService{}}

	// Sem lote de inspeção decidido, não há certificado.
	if _, err := cert.Emitir(ctx, certificado_uc.EmissaoDTO{ItemCode: item, Lote: sp("L-77"), Quantidade: decimal.NewFromInt(10)}); err == nil ||
		!strings.Contains(err.Error(), "precisa existir") {
		t.Fatalf("sem inspeção: %v", err)
	}
	l, err := lotes.Abrir(ctx, lote_uc.AberturaDTO{Origem: "MANUAL", ItemCode: item, Quantidade: decimal.NewFromInt(3), Lote: sp("L-77")})
	if err != nil {
		t.Fatal(err)
	}
	esp, acab := l.Caracteristicas[0].ID, l.Caracteristicas[1].ID
	var rs []lote_uc.ResultadoDTO
	for i, v := range []string{"9.9", "10.1", "10.0"} {
		rs = append(rs, lote_uc.ResultadoDTO{CaracteristicaID: esp, AmostraIndice: i + 1, Valor: dp(v)}, lote_uc.ResultadoDTO{CaracteristicaID: acab, AmostraIndice: i + 1, Conforme: bp(true)})
	}
	if _, err := lotes.GravarResultados(ctx, l.ID, rs); err != nil {
		t.Fatal(err)
	}
	if _, err := lotes.Decidir(ctx, l.ID, lote_uc.DecisaoDTO{Aceita: decimal.NewFromInt(3)}); err != nil {
		t.Fatal(err)
	}

	if _, err := cert.Emitir(ctx, certificado_uc.EmissaoDTO{ItemCode: item, Lote: sp("L-77"), Quantidade: decimal.NewFromInt(4), CustomerCode: &cliente}); err == nil ||
		!strings.Contains(err.Error(), "passa do que os lotes") {
		t.Fatalf("quantidade acima do aprovado: %v", err)
	}
	c, err := cert.Emitir(ctx, certificado_uc.EmissaoDTO{ItemCode: item, Lote: sp("L-77"), Quantidade: decimal.NewFromInt(3), CustomerCode: &cliente, Observacoes: sp("Pedido 4512")})
	if err != nil {
		t.Fatal(err)
	}
	ct := c.Conteudo
	if c.Numero != 1 || ct.Corrida != "C-4521" || ct.CertUsina != "USIMINAS 99812" || ct.Cliente == nil || ct.Cliente.Nome != "METALÚRGICA CLIENTE" ||
		ct.ItemNome != "FLANGE DN50" || len(ct.Lotes) != 1 {
		t.Fatalf("conteúdo: %+v", ct)
	}
	e := ct.Lotes[0].Caracteristicas[0]
	if e.N != 3 || e.MedidoMinimo != "9.9" || e.MedidoMaximo != "10.1" || e.Media != "10" || e.Minimo != "9.8" || e.Maximo != "10.2" {
		t.Fatalf("resumo da espessura: %+v", e)
	}
	if _, err := cert.Obter(b.Ctx, c.ID); err == nil {
		t.Fatal("B leu o certificado de A")
	}
	pdf := certpdf.Gerar(c, time.Now())
	if !bytes.HasPrefix(pdf, []byte("%PDF")) || len(pdf) < 1500 {
		t.Fatalf("PDF inválido (%d bytes)", len(pdf))
	}
	if _, err := pool.Exec(bg, `UPDATE quality_certificates SET quantidade=99 WHERE id=$1`, c.ID); err == nil {
		t.Fatal("certificado emitido não pode mudar")
	}
	if cc, err := cert.Cancelar(ctx, c.ID, "emitido para o cliente errado"); err != nil || cc.Status != "CANCELADO" {
		t.Fatalf("cancelamento: %v", err)
	}
	if _, err := cert.Cancelar(ctx, c.ID, "de novo"); err == nil {
		t.Fatal("cancelar duas vezes")
	}
	c2, err := cert.Emitir(ctx, certificado_uc.EmissaoDTO{ItemCode: item, LotesInspecao: []int64{l.ID}, Quantidade: decimal.NewFromInt(3), CustomerCode: &cliente})
	if err != nil || c2.Numero != 2 {
		t.Fatalf("reemissão (o cancelado libera a quantidade): %v", err)
	}
	if _, err := cert.Emitir(ctx, certificado_uc.EmissaoDTO{ItemCode: item, Lote: sp("L-77"), Quantidade: decimal.NewFromInt(1)}); err == nil ||
		!strings.Contains(err.Error(), "já certificada") {
		t.Fatalf("o lote já está todo certificado: %v", err)
	}

	// Documentos: quem elabora não aprova; aprovar torna a anterior obsoleta.
	docs := &documento_uc.UseCase{Repo: docpg.New(pool), Auth: &auth.AuthService{}}
	d, err := docs.Salvar(ctx, 0, documento_uc.DocumentoDTO{Codigo: "it-solda-01", Titulo: "Soldagem MIG da flange", Tipo: "it", ItemCode: &item})
	if err != nil || d.Codigo != "IT-SOLDA-01" {
		t.Fatalf("documento: %+v %v", d, err)
	}
	if _, err := docs.GravarRascunho(ctx, d.ID, documento_uc.RascunhoDTO{DescricaoMudanca: "emissão inicial", Conteudo: sp("1. Limpar. 2. Pré-aquecer a 150 °C.")}); err != nil {
		t.Fatal(err)
	}
	if _, err := docs.Aprovar(ctx, d.ID); err == nil || !strings.Contains(err.Error(), "elaborou") {
		t.Fatalf("quem elaborou aprovando: %v", err)
	}
	ctxRev := context.WithValue(bg, contextkey.UserKey, &security.AuthUser{ID: revisor.String(), Role: "ADMIN", EnterpriseID: a.ID, EnterpriseCode: a.Code})
	d, err = docs.Aprovar(ctxRev, d.ID)
	if err != nil || d.Vigente == nil || d.Vigente.Revisao != 0 || d.Rascunho != nil {
		t.Fatalf("aprovação da rev 0: %+v %v", d, err)
	}
	if _, err := pool.Exec(bg, `UPDATE quality_document_revisions SET conteudo='alterado' WHERE id=$1`, d.Vigente.ID); err == nil {
		t.Fatal("revisão vigente não pode mudar")
	}
	if _, err := docs.GravarRascunho(ctx, d.ID, documento_uc.RascunhoDTO{DescricaoMudanca: "temperatura", Arquivo: []byte("%PDF-1.4 IT rev1"), ArquivoNome: sp("it.pdf"), ArquivoTipo: sp("application/pdf")}); err != nil {
		t.Fatal(err)
	}
	if d, err = docs.Aprovar(ctxRev, d.ID); err != nil || d.Vigente.Revisao != 1 || !d.Vigente.TemArquivo {
		t.Fatalf("rev 1: %+v %v", d, err)
	}
	obsoletas := 0
	for _, r := range d.Revisoes {
		if r.Status == "OBSOLETA" {
			obsoletas++
		}
	}
	if obsoletas != 1 {
		t.Fatalf("a rev 0 deveria estar obsoleta: %+v", d.Revisoes)
	}
	if arq, err := docs.Arquivo(ctx, d.Vigente.ID); err != nil || string(arq.Arquivo) != "%PDF-1.4 IT rev1" {
		t.Fatalf("arquivo: %v", err)
	}
	if _, err := docs.Arquivo(b.Ctx, d.Vigente.ID); err == nil {
		t.Fatal("B baixou o arquivo de A")
	}
	if lista, _ := docs.Listar(b.Ctx, docdom.Filtro{}); len(lista) != 0 {
		t.Fatal("B listou documentos de A")
	}
}
