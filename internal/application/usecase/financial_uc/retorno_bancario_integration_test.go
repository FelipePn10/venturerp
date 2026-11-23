//go:build integration

package financial_uc_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/application/ports"

	"github.com/FelipePn10/panossoerp/internal/application/usecase/financial_uc"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/auth"
	financialpg "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/financial"
	fiscalpg "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/fiscal"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
)

func registro(campos map[int]string) string {
	b := []byte(strings.Repeat(" ", 240))
	for pos, v := range campos {
		copy(b[pos-1:], v)
	}
	return string(b)
}

// Melhoria 10: o boleto pago no retorno CNAB 240 é baixado sozinho; repetir o
// arquivo não baixa de novo; a empresa B não acha o título da A.
func TestRetornoBancarioBaixaOBoleto(t *testing.T) {
	pool := testutil.Pool(t)
	bg := context.Background()
	ator := testutil.Actor(t, pool)
	a, b := testutil.DuasEmpresas(t, pool, func(a, b testutil.Empresa) {
		for _, e := range []int64{a.ID, b.ID} {
			pool.Exec(bg, `DELETE FROM fluxo_caixa WHERE enterprise_id=$1`, e)
			pool.Exec(bg, `DELETE FROM contas_receber WHERE enterprise_id=$1`, e)
			pool.Exec(bg, `DELETE FROM contas_bancarias WHERE enterprise_id=$1`, e)
		}
	})
	hoje := time.Now()
	var conta, contaB, titulo int64
	for _, x := range []struct {
		e   testutil.Empresa
		dst *int64
	}{{a, &conta}, {b, &contaB}} {
		if err := pool.QueryRow(bg, `INSERT INTO contas_bancarias(banco,agencia,conta,descricao,saldo_inicial,is_active,created_by,enterprise_id) VALUES('341','1','1','Itaú',0,TRUE,$1,$2) RETURNING id`, ator, x.e.ID).Scan(x.dst); err != nil {
			t.Fatal(err)
		}
	}
	if err := pool.QueryRow(bg, `INSERT INTO contas_receber(numero_documento,data_emissao,data_vencimento,valor_bruto,valor_recebido,status,criado_por,enterprise_id,nosso_numero)
		VALUES('NF 10/1',$3::date,$3::date + 30,360.50,0,'PENDENTE',$1,$2,'0012345') RETURNING id`, ator, a.ID, hoje).Scan(&titulo); err != nil {
		t.Fatal(err)
	}
	fin := financialpg.NewFinancialRepositoryPG(pool)
	uc := &financial_uc.RetornoBancarioUseCase{Titulos: fin.(*financialpg.FinancialRepositoryPG),
		Baixa: &financial_uc.BaixarContaReceberUseCase{Repo: fin, Auth: &auth.AuthService{}, FiscalRepo: fiscalpg.NewFiscalRepositoryPG(pool)}}
	arq := strings.Join([]string{
		registro(map[int]string{8: "3", 14: "T", 16: "06", 38: "00000000000000012345", 82: "000000000036050"}),
		registro(map[int]string{8: "3", 14: "U", 16: "06", 18: "000000000000150", 78: "000000000036200", 138: hoje.Format("02012006"), 146: hoje.Format("02012006")}),
	}, "\r\n")
	if r, err := uc.Processar(b.Ctx, financial_uc.RetornoDTO{ContaBancariaID: contaB, Conteudo: arq}); err != nil || len(r.NaoEncontrados) != 1 {
		t.Fatalf("empresa B: %+v err=%v", r, err)
	}
	r, err := uc.Processar(a.Ctx, financial_uc.RetornoDTO{ContaBancariaID: conta, Conteudo: arq})
	if err != nil || len(r.Baixados) != 1 {
		t.Fatalf("retorno: %+v err=%v", r, err)
	}
	var recebido, juros decimal.Decimal
	var status string
	if err := pool.QueryRow(bg, `SELECT valor_recebido, COALESCE(juros,0) + COALESCE(multa,0), UPPER(status) FROM contas_receber WHERE id=$1`, titulo).Scan(&recebido, &juros, &status); err != nil {
		t.Fatal(err)
	}
	// Pago 362,00 = 360,50 do título + 1,50 de encargos que o banco cobrou.
	if !recebido.Equal(decimal.RequireFromString("360.50")) || !juros.Equal(decimal.RequireFromString("1.5")) || status != "RECEBIDO" {
		t.Fatalf("título: recebido %s, encargos %s, status %s", recebido, juros, status)
	}
	if r, err := uc.Processar(a.Ctx, financial_uc.RetornoDTO{ContaBancariaID: conta, Conteudo: arq}); err != nil || len(r.JaBaixados) != 1 || len(r.Baixados) != 0 {
		t.Fatalf("repetir o arquivo: %+v err=%v", r, err)
	}

	// O mesmo nosso número em DOIS bancos: o retorno do banco 2 baixa só o título
	// do banco 2, nunca o do banco 1.
	var conta2, doBanco1, doBanco2 int64
	if err := pool.QueryRow(bg, `INSERT INTO contas_bancarias(banco,agencia,conta,descricao,saldo_inicial,is_active,created_by,enterprise_id) VALUES('077','1','9','Inter',0,TRUE,$1,$2) RETURNING id`, ator, a.ID).Scan(&conta2); err != nil {
		t.Fatal(err)
	}
	for _, x := range []struct {
		conta int64
		dst   *int64
	}{{conta, &doBanco1}, {conta2, &doBanco2}} {
		if err := pool.QueryRow(bg, `INSERT INTO contas_receber(numero_documento,data_emissao,data_vencimento,valor_bruto,valor_recebido,status,criado_por,enterprise_id,nosso_numero,conta_bancaria_id)
			VALUES('NF 11/1',$3::date,$3::date + 30,360.50,0,'PENDENTE',$1,$2,'0077777',$4) RETURNING id`, ator, a.ID, hoje, x.conta).Scan(x.dst); err != nil {
			t.Fatal(err)
		}
	}
	arq2 := strings.Replace(arq, "00000000000000012345", "00000000000000077777", 1)
	if r, err := uc.Processar(a.Ctx, financial_uc.RetornoDTO{ContaBancariaID: conta2, Conteudo: arq2}); err != nil || len(r.Baixados) != 1 {
		t.Fatalf("retorno do banco 2: %+v err=%v", r, err)
	}
	var st1, st2 string
	_ = pool.QueryRow(bg, `SELECT UPPER(status) FROM contas_receber WHERE id=$1`, doBanco1).Scan(&st1)
	_ = pool.QueryRow(bg, `SELECT UPPER(status) FROM contas_receber WHERE id=$1`, doBanco2).Scan(&st2)
	if st1 == "RECEBIDO" || st2 != "RECEBIDO" {
		t.Fatalf("o retorno do banco 2 baixou o título errado: banco 1 %s, banco 2 %s", st1, st2)
	}
	// Dois títulos abertos da mesma conta com o mesmo número: não escolhe sozinho.
	for range 2 {
		testutil.Exec(t, pool, `INSERT INTO contas_receber(numero_documento,data_emissao,data_vencimento,valor_bruto,valor_recebido,status,criado_por,enterprise_id,nosso_numero,conta_bancaria_id)
			VALUES('NF 12/1',$3::date,$3::date + 30,100,0,'PENDENTE',$1,$2,'0088888',$4)`, ator, a.ID, hoje, conta2)
	}
	arq3 := strings.Replace(arq, "00000000000000012345", "00000000000000088888", 1)
	if r, err := uc.Processar(a.Ctx, financial_uc.RetornoDTO{ContaBancariaID: conta2, Conteudo: arq3}); err != nil || len(r.Falhas) != 1 || len(r.Baixados) != 0 {
		t.Fatalf("nosso número ambíguo precisa ir para falhas: %+v err=%v", r, err)
	}
}

type bancoQueConfirma struct{ seu string }

func (bancoQueConfirma) GerarPix(context.Context, ports.CobrancaPix) (*ports.PixGerado, error) {
	return nil, nil
}
func (bancoQueConfirma) ConsultarCobranca(context.Context, string) (*ports.PixGerado, error) {
	return nil, nil
}
func (bancoQueConfirma) CancelarCobranca(context.Context, string, string) error { return nil }
func (bancoQueConfirma) RegistrarWebhook(context.Context, string) error         { return nil }
func (b bancoQueConfirma) ConsultarPagamento(context.Context, string) (*ports.PagamentoConsultado, error) {
	return &ports.PagamentoConsultado{SeuNumero: b.seu, Pago: true, Valor: decimal.RequireFromString("101.30"), Data: time.Now(), Forma: "PIX", Situacao: "RECEBIDO"}, nil
}

// Aviso do banco: a consulta confirma o pagamento e o título baixa (o excedente
// vira encargo); repetir o aviso não baixa de novo; a empresa B não acha o título.
func TestAvisoDoBancoBaixaPelaConsulta(t *testing.T) {
	pool := testutil.Pool(t)
	bg := context.Background()
	ator := testutil.Actor(t, pool)
	a, b := testutil.DuasEmpresas(t, pool, func(a, b testutil.Empresa) {
		for _, e := range []int64{a.ID, b.ID} {
			pool.Exec(bg, `DELETE FROM contas_receber WHERE enterprise_id=$1`, e)
			pool.Exec(bg, `DELETE FROM contas_bancarias WHERE enterprise_id=$1`, e)
		}
	})
	hoje := time.Now()
	var conta, titulo int64
	if err := pool.QueryRow(bg, `INSERT INTO contas_bancarias(banco,agencia,conta,descricao,saldo_inicial,is_active,created_by,enterprise_id) VALUES('077','1','1','Inter',0,TRUE,$1,$2) RETURNING id`, ator, a.ID).Scan(&conta); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(bg, `INSERT INTO contas_receber(numero_documento,data_emissao,data_vencimento,valor_bruto,valor_recebido,status,criado_por,enterprise_id)
		VALUES('NF 2/1',$3::date,$3::date + 5,100,0,'PENDENTE',$1,$2) RETURNING id`, ator, a.ID, hoje).Scan(&titulo); err != nil {
		t.Fatal(err)
	}
	fin := financialpg.NewFinancialRepositoryPG(pool)
	pg := fin.(*financialpg.FinancialRepositoryPG)
	uc := &financial_uc.AvisoDoBancoUseCase{Cobranca: bancoQueConfirma{seu: financial_uc.SeuNumeroDoTitulo(titulo)}, Titulos: pg, ContaBancariaID: conta,
		Baixa: &financial_uc.BaixarContaReceberUseCase{Repo: fin, Auth: &auth.AuthService{}, FiscalRepo: fiscalpg.NewFiscalRepositoryPG(pool)}}
	if r, err := uc.Processar(b.Ctx, []string{"x"}); err != nil || len(r.Baixados) != 0 {
		t.Fatalf("empresa B: %+v err=%v", r, err)
	}
	if r, err := uc.Processar(a.Ctx, []string{"x"}); err != nil || len(r.Baixados) != 1 {
		t.Fatalf("aviso: %+v err=%v", r, err)
	}
	var recebido, encargos decimal.Decimal
	if err := pool.QueryRow(bg, `SELECT valor_recebido, COALESCE(juros,0)+COALESCE(multa,0) FROM contas_receber WHERE id=$1`, titulo).Scan(&recebido, &encargos); err != nil {
		t.Fatal(err)
	}
	if !recebido.Equal(decimal.NewFromInt(100)) || !encargos.Equal(decimal.RequireFromString("1.3")) {
		t.Fatalf("recebido %s, encargos %s", recebido, encargos)
	}
	if r, _ := uc.Processar(a.Ctx, []string{"x"}); len(r.Baixados) != 0 {
		t.Fatalf("o aviso repetido baixou de novo: %+v", r)
	}
	if code, ator2, err := pg.SessaoDoSistema(bg, a.ID); err != nil || code != a.Code || ator2 == uuid.Nil {
		t.Logf("sessão do sistema da empresa de teste: code=%d err=%v (sem ADMIN vinculado é esperado aqui)", code, err)
	}
}

type bancoContador struct {
	emissoes, cancelamentos int
	situacao                string
}

func (b *bancoContador) GerarPix(context.Context, ports.CobrancaPix) (*ports.PixGerado, error) {
	b.emissoes++
	return &ports.PixGerado{IDExterno: fmt.Sprintf("sol-%d", b.emissoes), LinhaDigitavel: "0779", CopiaECola: "0002"}, nil
}
func (b *bancoContador) ConsultarCobranca(_ context.Context, id string) (*ports.PixGerado, error) {
	return &ports.PixGerado{IDExterno: id, LinhaDigitavel: "0779", Situacao: b.situacao}, nil
}
func (b *bancoContador) ConsultarPagamento(context.Context, string) (*ports.PagamentoConsultado, error) {
	return nil, nil
}
func (b *bancoContador) CancelarCobranca(context.Context, string, string) error {
	b.cancelamentos++
	return nil
}
func (b *bancoContador) RegistrarWebhook(context.Context, string) error { return nil }

// Emitir de novo consulta a mesma cobrança (não cria outra no banco); só emite
// outra se a do banco foi cancelada. Cancelar o título cancela a do banco; a
// empresa B não emite cobrança do título da A.
func TestCobrancaNaoDuplicaECancelaNoBanco(t *testing.T) {
	pool := testutil.Pool(t)
	bg := context.Background()
	ator := testutil.Actor(t, pool)
	a, b := testutil.DuasEmpresas(t, pool, func(a, b testutil.Empresa) {
		for _, e := range []int64{a.ID, b.ID} {
			pool.Exec(bg, `DELETE FROM contas_receber WHERE enterprise_id=$1`, e)
		}
	})
	var titulo int64
	if err := pool.QueryRow(bg, `INSERT INTO contas_receber(numero_documento,data_emissao,data_vencimento,valor_bruto,valor_recebido,status,criado_por,enterprise_id)
		VALUES('NF 9/1',$3::date,$3::date + 10,250,0,'PENDENTE',$1,$2) RETURNING id`, ator, a.ID, time.Now()).Scan(&titulo); err != nil {
		t.Fatal(err)
	}
	fin := financialpg.NewFinancialRepositoryPG(pool)
	pg := fin.(*financialpg.FinancialRepositoryPG)
	banco := &bancoContador{situacao: "A_RECEBER"}
	pix := &financial_uc.CobrancaPixUseCase{Repo: fin, Auth: &auth.AuthService{}, Cobranca: banco, Gravar: pg}
	if _, err := pix.Gerar(b.Ctx, titulo); err == nil {
		t.Fatal("a empresa B emitiu a cobrança do título da A")
	}
	if _, err := pix.Gerar(a.Ctx, titulo); err != nil {
		t.Fatal(err)
	}
	if _, err := pix.Gerar(a.Ctx, titulo); err != nil || banco.emissoes != 1 {
		t.Fatalf("segunda emissão criou outra cobrança: emissões=%d err=%v", banco.emissoes, err)
	}
	banco.situacao = "CANCELADO"
	if g, err := pix.Gerar(a.Ctx, titulo); err != nil || banco.emissoes != 2 || g.IDExterno != "sol-2" {
		t.Fatalf("cobrança cancelada no banco deveria ser emitida de novo: %+v emissões=%d err=%v", g, banco.emissoes, err)
	}
	var ext string
	_ = pool.QueryRow(bg, `SELECT COALESCE(cobranca_externa_id,'') FROM contas_receber WHERE id=$1`, titulo).Scan(&ext)
	if ext != "sol-2" {
		t.Fatalf("cobrança guardada no título: %q", ext)
	}
	cancelar := &financial_uc.CancelContaReceberUseCase{Repo: fin, Auth: &auth.AuthService{}, Banco: &financial_uc.CobrancaNoBanco{Cobranca: banco, Titulos: pg}}
	if err := cancelar.Execute(a.Ctx, titulo); err != nil || banco.cancelamentos != 1 {
		t.Fatalf("cancelar o título: cancelamentos no banco=%d err=%v", banco.cancelamentos, err)
	}
}
