package handler

import (
	"bytes"
	"context"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/consumer_service_uc"
	"github.com/FelipePn10/panossoerp/internal/domain/consumer_service/entity"
	csrepo "github.com/FelipePn10/panossoerp/internal/domain/consumer_service/repository"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type attachmentAuth struct{ ports.AuthService }

func (attachmentAuth) CanManageTechnicalAssistance(context.Context) bool { return true }
func (attachmentAuth) EnterpriseID(context.Context) (int64, error)       { return 7, nil }
func (attachmentAuth) UserID(context.Context) (uuid.UUID, error) {
	return uuid.MustParse("00000000-0000-0000-0000-000000000042"), nil
}

type attachmentRepo struct {
	csrepo.Repository
	attachment *entity.CallAttachment
}

func (r *attachmentRepo) AddCallAttachment(_ context.Context, tenant int64, v *entity.CallAttachment) (*entity.CallAttachment, error) {
	if tenant != 7 {
		return nil, csrepo.ErrAttachmentNotFound
	}
	v.Code = 12
	r.attachment = v
	return v, nil
}
func (r *attachmentRepo) GetCallAttachment(_ context.Context, tenant, call, code int64) (*entity.CallAttachment, error) {
	if tenant != 7 || r.attachment == nil || r.attachment.CallCode != call || r.attachment.Code != code {
		return nil, csrepo.ErrAttachmentNotFound
	}
	return r.attachment, nil
}
func (r *attachmentRepo) DeleteCallAttachment(_ context.Context, tenant, call, code int64) error {
	if tenant != 7 || r.attachment == nil {
		return csrepo.ErrAttachmentNotFound
	}
	r.attachment = nil
	return nil
}

func TestConsumerServiceMultipartUploadDownloadAndDelete(t *testing.T) {
	repo := &attachmentRepo{}
	handler := NewConsumerServiceHandler(&consumer_service_uc.UseCase{Repo: repo, Auth: attachmentAuth{}})
	router := chi.NewRouter()
	router.Post("/calls/{code}/attachments", handler.AddCallAttachment)
	router.Get("/calls/{code}/attachments/{attachmentCode}/download", handler.DownloadCallAttachment)
	router.Delete("/calls/{code}/attachments/{attachmentCode}", handler.DeleteCallAttachment)
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, _ := writer.CreateFormFile("file", "evidencia.pdf")
	_, _ = part.Write([]byte("%PDF-1.7 evidencia"))
	_ = writer.Close()
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/calls/5/attachments", &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusCreated || !strings.Contains(recorder.Body.String(), `"file_size":18`) {
		t.Fatalf("upload: %d %s", recorder.Code, recorder.Body.String())
	}
	recorder = httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/calls/5/attachments/12/download", nil))
	if recorder.Code != http.StatusOK || recorder.Body.String() != "%PDF-1.7 evidencia" {
		t.Fatalf("download: %d %q", recorder.Code, recorder.Body.String())
	}
	recorder = httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodDelete, "/calls/5/attachments/12", nil))
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("delete: %d", recorder.Code)
	}
}
