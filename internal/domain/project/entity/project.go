// Package entity: Projeto (obra, contrato, desenvolvimento) ao qual pedidos de
// venda, demandas e ordens de fabricação pertencem. Por empresa; o código é
// único na empresa sem diferenciar maiúsculas.
package entity

import (
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/FelipePn10/panossoerp/internal/domain/domainerr"
)

const (
	StatusAtivo     = "ATIVO"
	StatusEncerrado = "ENCERRADO"
	StatusCancelado = "CANCELADO"
)

type Project struct {
	ID           int64
	EnterpriseID int64
	Code         string
	Name         string
	CustomerCode *int64
	CustomerName string
	Status       string
	StartDate    *time.Time
	EndDate      *time.Time
	Notes        string
	CreatedBy    *uuid.UUID
	CreatedAt    time.Time
	UpdatedAt    time.Time
	// Ordens de fabricação ligadas (abertas e no total), só na leitura.
	OpenOrders  int
	TotalOrders int
}

// Normalizar limpa os campos e confere as regras do cadastro.
func (p *Project) Normalizar() error {
	p.Code = strings.TrimSpace(p.Code)
	p.Name = strings.TrimSpace(p.Name)
	p.Notes = strings.TrimSpace(p.Notes)
	p.Status = strings.ToUpper(strings.TrimSpace(p.Status))
	if p.Status == "" {
		p.Status = StatusAtivo
	}
	if p.Code == "" {
		return domainerr.NewValidation("informe o código do projeto")
	}
	if len(p.Code) > 50 {
		return domainerr.NewValidation("o código do projeto tem no máximo 50 caracteres")
	}
	if p.Name == "" {
		return domainerr.NewValidation("informe o nome do projeto")
	}
	if len(p.Name) > 200 {
		return domainerr.NewValidation("o nome do projeto tem no máximo 200 caracteres")
	}
	switch p.Status {
	case StatusAtivo, StatusEncerrado, StatusCancelado:
	default:
		return domainerr.NewValidation("situação do projeto inválida: use ATIVO, ENCERRADO ou CANCELADO")
	}
	if p.CustomerCode != nil && *p.CustomerCode <= 0 {
		p.CustomerCode = nil
	}
	if p.StartDate != nil && p.EndDate != nil && p.EndDate.Before(*p.StartDate) {
		return domainerr.NewValidation("o término do projeto não pode ser antes do início")
	}
	return nil
}
