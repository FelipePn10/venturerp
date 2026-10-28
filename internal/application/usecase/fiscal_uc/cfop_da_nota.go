package fiscal_uc

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/focusnfe"
)

// CatalogoCFOP: o cadastro de CFOP com a tabela oficial (IT 2023.002), para
// conferir o CFOP de cada linha antes da transmissão.
type CatalogoCFOP interface {
	GetCFOPByCode(ctx context.Context, code int32) (*entity.CFOP, error)
}

// conferirCFOPs: a SEFAZ recusa CFOP inexistente, fora da vigência ou não
// permitido em NF-e, e a devolução sem o documento referenciado. Vale para
// toda finalidade (inclusive a devolução, que a conferência tributária pula).
func conferirCFOPs(ctx context.Context, cat CatalogoCFOP, exit *entity.FiscalExit, items []*entity.FiscalExitItem, payload *focusnfe.NFEPayload) ([]response.PreviaPendenciaNFe, error) {
	if cat == nil || exit == nil {
		return nil, nil
	}
	var p []response.PreviaPendenciaNFe
	add := func(campo, msg, como string) {
		p = append(p, response.PreviaPendenciaNFe{Nivel: NivelImpede, Campo: campo, Mensagem: msg, ComoResolver: como})
	}
	dia := exit.DataEmissao
	if dia.IsZero() {
		dia = time.Now()
	}
	referenciada := exit.NFeReferenciada != nil && strings.TrimSpace(*exit.NFeReferenciada) != ""
	if payload != nil && len(payload.NotasReferenciadas) > 0 {
		referenciada = true
	}

	linhas := map[string][]string{}
	for i, it := range items {
		c := strings.TrimSpace(it.Cfop)
		if c == "" {
			continue // a falta do CFOP já é pendência da prévia
		}
		porItem := payload != nil && i < len(payload.Items) && payload.Items[i].DFeRefChave != ""
		if porItem {
			linhas[c+"|ref"] = append(linhas[c+"|ref"], strconv.Itoa(it.Sequence))
		} else {
			linhas[c] = append(linhas[c], strconv.Itoa(it.Sequence))
		}
	}
	chaves := make([]string, 0, len(linhas))
	for k := range linhas {
		chaves = append(chaves, k)
	}
	sort.Strings(chaves)
	vistos := map[string]*entity.CFOP{}
	for _, k := range chaves {
		codigo, comRef := strings.CutSuffix(k, "|ref")
		seq := linhasPorExtenso(linhas[k])
		campo := "itens.cfop." + codigo
		cf, ja := vistos[codigo]
		if !ja {
			n, err := strconv.Atoi(codigo)
			if err != nil || len(codigo) != 4 {
				add(campo, fmt.Sprintf("o CFOP %q de %s não tem 4 dígitos", codigo, seq), "informe um CFOP da tabela (VFIS0300) e gere a nota novamente")
				continue
			}
			cf, err = cat.GetCFOPByCode(ctx, int32(n))
			if err != nil && !errorsuc.EhNaoEncontrado(err) {
				return nil, fmt.Errorf("lendo o CFOP %s: %w", codigo, err)
			}
			vistos[codigo] = cf
		}
		switch {
		case cf == nil:
			add(campo, fmt.Sprintf("o CFOP %s de %s não existe na tabela de CFOP", codigo, seq),
				"use um CFOP da tabela oficial (VFIS0300) e gere a nota novamente")
		case !cf.IsActive:
			add(campo, fmt.Sprintf("o CFOP %s de %s está inativo no cadastro", codigo, seq),
				"reative o CFOP na VFIS0300 ou use outro")
		case cf.Oficial.Oficial && !cf.VigenteEm(dia):
			add(campo, fmt.Sprintf("o CFOP %s de %s está fora da vigência na data de emissão %s", codigo, seq, dia.Format("02/01/2006")),
				"use o CFOP vigente (VFIS0300 mostra o início e o fim de cada código)")
		case cf.Oficial.Oficial && !cf.Oficial.NFe:
			add(campo, fmt.Sprintf("o CFOP %s de %s não pode ser usado em NF-e (tabela oficial)", codigo, seq),
				"use um CFOP permitido em NF-e")
		case cf.Oficial.Devolucao && !referenciada && !comRef:
			add(campo, fmt.Sprintf("o CFOP %s de %s é de devolução e a nota não referencia a nota devolvida", codigo, seq),
				"informe a chave da nota de origem (devolução de venda na VVND0700, de compra na devolução de compra) ou use outro CFOP")
		}
	}
	return p, nil
}

// recusarCFOPs: a autorização não transmite com CFOP que a SEFAZ recusa.
func recusarCFOPs(ctx context.Context, cat CatalogoCFOP, exit *entity.FiscalExit, items []*entity.FiscalExitItem, payload *focusnfe.NFEPayload) error {
	pend, err := conferirCFOPs(ctx, cat, exit, items, payload)
	if err != nil || len(pend) == 0 {
		return err
	}
	msgs := make([]string, 0, len(pend))
	for _, x := range pend {
		msgs = append(msgs, x.Mensagem+" — "+x.ComoResolver)
	}
	return errorsuc.NewValidationError("o CFOP da nota tem pendências: " + strings.Join(msgs, "; "))
}
