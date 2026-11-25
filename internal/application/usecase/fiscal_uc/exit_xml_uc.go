package fiscal_uc

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/repository"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/focusnfe"
)

// GetExitXMLUseCase devolve o XML autorizado de uma NF-e emitida, com a chave e
// o protocolo, para o usuário conferir os impostos e repassar a quem precisar.
// A cópia fica no banco: a primeira consulta de uma nota antiga a baixa da
// Focus; as seguintes não dependem da emissora.
type GetExitXMLUseCase struct {
	Repo repository.FiscalRepository
	Auth ports.AuthService
}

type ExitXMLResult struct {
	ExitID      int64  `json:"exit_id"`
	NumeroNF    int64  `json:"numero_nf"`
	Serie       string `json:"serie"`
	ChaveAcesso string `json:"chave_acesso"`
	Protocolo   string `json:"protocolo"`
	Status      string `json:"status"`
	XML         string `json:"xml"`
	// XMLCancelamento: o evento de cancelamento, quando a nota foi cancelada.
	XMLCancelamento string `json:"xml_cancelamento,omitempty"`
}

func (uc *GetExitXMLUseCase) Execute(ctx context.Context, exitID int64) (*ExitXMLResult, error) {
	if !uc.Auth.CanGetFiscalExit(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	exit, err := uc.Repo.GetExitByID(ctx, exitID)
	if err != nil {
		return nil, err
	}
	if exit.Status != entity.ExitStatusAuthorized && exit.Status != entity.ExitStatusCancelled {
		return nil, errorsuc.NewValidationError(fmt.Sprintf("a nota não tem XML autorizado: ela está %s", situacaoDaNota(exit.Status)))
	}
	res := &ExitXMLResult{
		ExitID: exitID, NumeroNF: exit.NumeroNF, Serie: exit.Serie,
		ChaveAcesso: derefStr(exit.ChaveAcesso), Protocolo: derefStr(exit.Protocolo), Status: string(exit.Status),
	}

	if res.XML, res.XMLCancelamento, err = uc.Repo.ExitXML(ctx, exitID); err != nil {
		return nil, err
	}
	falta := res.XML == ""
	faltaCancelamento := exit.Status == entity.ExitStatusCancelled && res.XMLCancelamento == "" && exit.FocusRef != nil
	if !falta && !faltaCancelamento {
		return res, nil
	}

	if exit.FocusRef == nil && derefStr(exit.XmlPath) == "" {
		return nil, errorsuc.NewValidationError("esta nota não foi transmitida à SEFAZ por este sistema, então o XML dela não está aqui: peça-o a quem a emitiu")
	}
	cfg, err := uc.Repo.GetFiscalConfig(ctx)
	if err != nil {
		return nil, err
	}
	if cfg.FocusNfeToken == nil || *cfg.FocusNfeToken == "" {
		if !falta {
			return res, nil
		}
		return nil, errorsuc.NewValidationError(tokenFocusAusente)
	}
	cli := focusnfe.NewClient(*cfg.FocusNfeToken, cfg.FocusNfeAmbiente)

	caminho, caminhoCancelamento := derefStr(exit.XmlPath), ""
	if (falta && caminho == "") || faltaCancelamento {
		cli.WithLogger(func(endpoint, method, reqBody, respBody string, statusCode, durationMs int) {
			if errLog := uc.Repo.SaveFocusLog(ctx, exitID, endpoint, method, reqBody, respBody, statusCode, durationMs); errLog != nil {
				slog.Error("falha ao gravar o registro da comunicação com a Focus", "documento", exitID, "endpoint", endpoint, "erro", errLog)
			}
		})
		resp, errConsulta := cli.ConsultarNFe(ctx, *exit.FocusRef)
		if errConsulta != nil {
			if !falta {
				// O XML autorizado já está aqui: falta só o do cancelamento.
				slog.WarnContext(ctx, "XML do cancelamento indisponível agora", "documento", exitID, "erro", errConsulta)
				return res, nil
			}
			return nil, erroDaEmissora(ctx, "a consulta do XML", exitID, errConsulta)
		}
		if resp != nil {
			if caminho == "" {
				caminho = resp.PathXML
			}
			caminhoCancelamento = resp.PathXMLCancel
		}
	}

	if falta {
		if caminho == "" {
			return nil, errorsuc.NewValidationError("a Focus NF-e ainda não disponibilizou o XML desta nota: tente de novo em instantes")
		}
		xml, err := BaixarXMLDaNota(ctx, cli, caminho, res.ChaveAcesso)
		if err != nil {
			return nil, erroDaEmissora(ctx, "o download do XML", exitID, err)
		}
		if err := uc.Repo.SaveExitXML(ctx, exitID, xml); err != nil {
			return nil, err
		}
		res.XML = xml
	}
	if faltaCancelamento && caminhoCancelamento != "" {
		if xml, err := BaixarXMLDaNota(ctx, cli, caminhoCancelamento, res.ChaveAcesso); err != nil {
			slog.WarnContext(ctx, "XML do cancelamento não foi baixado", "documento", exitID, "erro", err)
		} else if err := uc.Repo.SaveExitCancelXML(ctx, exitID, xml); err != nil {
			return nil, err
		} else {
			res.XMLCancelamento = xml
		}
	}
	return res, nil
}

// BaixarXMLDaNota baixa o XML pelo caminho da Focus e confere que é mesmo o
// documento (com a chave, quando informada): o que vai para o banco e para o
// usuário não pode ser uma página de erro.
func BaixarXMLDaNota(ctx context.Context, cli *focusnfe.Client, caminho, chave string) (string, error) {
	b, err := cli.BaixarArquivo(ctx, caminho)
	if err != nil {
		return "", err
	}
	xml := strings.TrimPrefix(string(b), "\ufeff")
	if !strings.HasPrefix(strings.TrimSpace(xml), "<") || strings.Contains(strings.ToLower(xml[:min(len(xml), 200)]), "<html") {
		return "", fmt.Errorf("o arquivo baixado de %s não é um XML", caminho)
	}
	if chave != "" && !strings.Contains(xml, chave) {
		return "", fmt.Errorf("o XML baixado de %s não é o da chave %s", caminho, chave)
	}
	return xml, nil
}

// guardarXMLDoCancelamento guarda o XML do evento de cancelamento. Falhar aqui
// não desfaz o cancelamento: a consulta do XML (ou a rotina) tenta de novo.
func guardarXMLDoCancelamento(ctx context.Context, repo repository.FiscalRepository, cli *focusnfe.Client, exitID int64, caminho, chave string) {
	if caminho == "" {
		return
	}
	xml, err := BaixarXMLDaNota(ctx, cli, caminho, chave)
	if err == nil {
		err = repo.SaveExitCancelXML(ctx, exitID, xml)
	}
	if err != nil {
		slog.WarnContext(ctx, "XML do cancelamento da NF-e não foi guardado agora; será baixado depois", "documento", exitID, "erro", err)
	}
}

// guardarXMLDaNota guarda a cópia do XML logo na autorização. Falhar aqui não
// desfaz a autorização: a primeira consulta do XML tenta de novo.
func guardarXMLDaNota(ctx context.Context, repo repository.FiscalRepository, cli *focusnfe.Client, exitID int64, caminho, chave string) {
	if caminho == "" {
		return
	}
	xml, err := BaixarXMLDaNota(ctx, cli, caminho, chave)
	if err == nil {
		err = repo.SaveExitXML(ctx, exitID, xml)
	}
	if err != nil {
		slog.WarnContext(ctx, "XML da NF-e autorizada não foi guardado agora; será baixado na primeira consulta", "documento", exitID, "erro", err)
	}
}
