package aps

import (
	"context"
	"fmt"
	"strconv"

	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"
)

// FamiliaDeSetup é um agrupamento de itens que custam o mesmo para trocar na
// máquina, com quantos itens carrega hoje.
type FamiliaDeSetup struct {
	Nome  string `json:"family"`
	Itens int64  `json:"items"`
}

// ListarFamiliasDeSetup devolve as famílias já usadas na empresa. É o que
// permite à tela oferecer seleção em vez de texto livre — família digitada
// errado vira uma família nova, silenciosamente sem regra nenhuma.
func (r *APSRepositorySQLC) ListarFamiliasDeSetup(ctx context.Context) ([]FamiliaDeSetup, error) {
	e, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, `SELECT setup_family, COUNT(*)
		FROM items WHERE enterprise_id=$1 AND setup_family IS NOT NULL AND setup_family <> ''
		GROUP BY setup_family ORDER BY setup_family`, e)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []FamiliaDeSetup{}
	for rows.Next() {
		var f FamiliaDeSetup
		if err := rows.Scan(&f.Nome, &f.Itens); err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

// ItensDaFamiliaDeSetup lista os itens de uma família, para conferir a cobertura.
func (r *APSRepositorySQLC) ItensDaFamiliaDeSetup(ctx context.Context, familia string) ([]string, error) {
	e, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, `SELECT COALESCE(NULLIF(business_code,''), code::text) FROM items
		WHERE enterprise_id=$1 AND setup_family=$2 ORDER BY 1`, e, familia)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var c string
		if err := rows.Scan(&c); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// DefinirFamiliaDeSetup atribui (ou limpa, com família vazia) a família dos
// itens informados. Uma chamada por lote: atribuir quarenta chapas de uma vez é
// o caso normal, e quarenta requisições seriam quarenta chances de parar no meio.
func (r *APSRepositorySQLC) DefinirFamiliaDeSetup(ctx context.Context, familia string, itens []int64) (int64, error) {
	e, err := tenant.ID(ctx)
	if err != nil {
		return 0, err
	}
	if len(itens) == 0 {
		return 0, nil
	}
	var valor any
	if familia != "" {
		valor = familia
	}
	textos := make([]string, 0, len(itens))
	for _, c := range itens {
		textos = append(textos, strconv.FormatInt(c, 10))
	}
	tag, err := r.pool.Exec(ctx, `UPDATE items SET setup_family=$4
		WHERE enterprise_id=$1 AND (code = ANY($2) OR business_code = ANY($3))`, e, itens, textos, valor)
	if err != nil {
		return 0, fmt.Errorf("gravando família de preparação: %w", err)
	}
	return tag.RowsAffected(), nil
}

// ── Parada em aberto (cronômetro do operador) ──────────────────────────────

// AbrirParadaDeMaquina registra que a máquina parou AGORA, sem fim definido. A
// operação é idempotente por máquina: se já houver uma parada aberta, devolve a
// que existe em vez de abrir outra — tocar duas vezes no botão é o normal num
// terminal de chão de fábrica.
func (r *APSRepositorySQLC) AbrirParadaDeMaquina(ctx context.Context, machineID int64, motivo, descricao string) (int64, bool, error) {
	e, err := tenant.ID(ctx)
	if err != nil {
		return 0, false, err
	}
	var id int64
	err = r.pool.QueryRow(ctx, `SELECT id FROM machine_downtimes
		WHERE enterprise_id=$1 AND machine_id=$2 AND ends_at IS NULL`, e, machineID).Scan(&id)
	if err == nil {
		return id, true, nil
	}
	err = r.pool.QueryRow(ctx, `INSERT INTO machine_downtimes(enterprise_id,machine_id,starts_at,ends_at,downtime_type,reason)
		SELECT $1,$2,NOW(),NULL,$3,$4 WHERE EXISTS(SELECT 1 FROM machines WHERE id=$2 AND enterprise_id=$1)
		RETURNING id`, e, machineID, motivo, descricao).Scan(&id)
	return id, false, err
}

// FecharParadaDeMaquina encerra a parada aberta da máquina no instante atual e
// devolve quantos minutos ela durou.
func (r *APSRepositorySQLC) FecharParadaDeMaquina(ctx context.Context, machineID int64) (int64, float64, error) {
	e, err := tenant.ID(ctx)
	if err != nil {
		return 0, 0, err
	}
	var id int64
	var minutos float64
	err = r.pool.QueryRow(ctx, `UPDATE machine_downtimes SET ends_at=NOW()
		WHERE enterprise_id=$1 AND machine_id=$2 AND ends_at IS NULL
		RETURNING id, EXTRACT(EPOCH FROM (ends_at-starts_at))/60`, e, machineID).Scan(&id, &minutos)
	return id, minutos, err
}

// ParadaAbertaDaMaquina devolve a parada em curso, se houver.
func (r *APSRepositorySQLC) ParadaAbertaDaMaquina(ctx context.Context, machineID int64) (int64, string, string, float64, bool, error) {
	e, err := tenant.ID(ctx)
	if err != nil {
		return 0, "", "", 0, false, err
	}
	var id int64
	var motivo, descricao string
	var minutos float64
	err = r.pool.QueryRow(ctx, `SELECT id, downtime_type, reason, EXTRACT(EPOCH FROM (NOW()-starts_at))/60
		FROM machine_downtimes WHERE enterprise_id=$1 AND machine_id=$2 AND ends_at IS NULL`,
		e, machineID).Scan(&id, &motivo, &descricao, &minutos)
	if err != nil {
		return 0, "", "", 0, false, nil
	}
	return id, motivo, descricao, minutos, true, nil
}
