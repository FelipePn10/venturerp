//go:build integration

package notification

import (
	"errors"
	"testing"

	notificationentity "github.com/FelipePn10/panossoerp/internal/domain/notification/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
)

// A razão social e o logo do e-mail de notificação vêm da configuração fiscal
// escolhida na VNOT0100. Antes qualquer id era aceito: a empresa B gravava o id
// da configuração da A e os e-mails de B saíam com a marca da A.
func TestConfiguracaoFiscalDaNotificacaoEhDaPropriaEmpresa(t *testing.T) {
	pool := testutil.Pool(t)
	repo := New(pool)
	a, b := testutil.DuasEmpresas(t, pool, nil)
	actor := testutil.Actor(t, pool)

	config := func(empresa int64, razao string) int64 {
		var id int64
		if err := pool.QueryRow(a.Ctx, `INSERT INTO fiscal_configs(cnpj_empresa,razao_social,uf_empresa,updated_by,enterprise_id)
			VALUES('11222333000181',$1,'PR',$2,$3) RETURNING id`, razao, actor, empresa).Scan(&id); err != nil {
			t.Fatal(err)
		}
		return id
	}
	cfgA := config(a.ID, "MARCA DA EMPRESA A")
	cfgB := config(b.ID, "MARCA DA EMPRESA B")

	base := func(empresa int64, cfg int64) notificationentity.Settings {
		return notificationentity.Settings{EnterpriseID: empresa, DigestTime: "08:00", Timezone: "America/Sao_Paulo",
			RetentionDays: 365, MaxAttachmentBytes: 1024 * 1024, MaxEmailsPerMinute: 60, FiscalConfigID: &cfg}
	}

	// A própria configuração é aceita.
	if err := repo.SaveSettings(b.Ctx, base(b.ID, cfgB), actor); err != nil {
		t.Fatalf("B gravando a própria configuração fiscal: %v", err)
	}
	// A configuração da outra empresa é recusada como erro de validação, e a de B fica como estava.
	err := repo.SaveSettings(b.Ctx, base(b.ID, cfgA), actor)
	if !errors.Is(err, notificationentity.ErrValidation) || !errors.Is(err, notificationentity.ErrConfigFiscalDeOutraEmpresa) {
		t.Fatalf("B gravando a configuração fiscal da A: esperado erro de validação, veio %v", err)
	}
	s, err := repo.GetSettings(b.Ctx, b.ID)
	if err != nil {
		t.Fatal(err)
	}
	if s.FiscalConfigID == nil || *s.FiscalConfigID != cfgB {
		t.Fatalf("a recusa alterou a configuração de B: %v", s.FiscalConfigID)
	}
	// Sem configuração continua permitido.
	semCfg := base(b.ID, 0)
	semCfg.FiscalConfigID = nil
	if err := repo.SaveSettings(b.Ctx, semCfg, actor); err != nil {
		t.Fatalf("B sem configuração fiscal: %v", err)
	}
}
