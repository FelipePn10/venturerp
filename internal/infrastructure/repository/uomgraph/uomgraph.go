// Package uomgraph carrega do banco as arestas de conversão de unidade de um
// item — e só isso. A decisão (caminho, precedência, fração) é do domínio
// (item_conversion/service).
//
// Existiam cinco SQLs diferentes para "qual o fator de X para Y deste item":
// a compra olhava a máscara e não a conversão padrão; a nota de entrada olhava
// a padrão e ignorava a máscara; a sucata da OP não olhava nenhuma das duas; o
// saldo em segunda unidade só achava conversão que tocasse a unidade de
// estoque. O mesmo item convertia diferente conforme a tela. Agora todos leem
// por aqui.
package uomgraph

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/domain/item_conversion/service"
	uomentity "github.com/FelipePn10/panossoerp/internal/domain/unit_of_measure/entity"
	uomrepo "github.com/FelipePn10/panossoerp/internal/domain/unit_of_measure/repository"
)

// Querier é o que pool e transação têm em comum.
type Querier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

// arestasSQL: lote, item (com e sem máscara, vigente em $5), a segunda
// unidade e as embalagens do cadastro do item (VENT0200) e padrão da empresa. A empresa do
// item confere por EXISTS: o código do item é único no
// banco, mas a conversão dele só vale dentro da empresa dele.
const arestasSQL = `
SELECT c.item_code, c.id, c.from_uom, c.to_uom, c.from_quantity, c.to_quantity,
       CASE WHEN c.mask <> '' THEN 'ITEM_MASCARA' ELSE 'ITEM' END,
       c.rounding_percent, c.tolerance_value, c.tolerance_type
  FROM item_unit_conversions c
 WHERE c.item_code = ANY($2) AND c.is_active
   AND (c.mask = '' OR c.mask = $3)
   AND c.valid_from <= COALESCE($5::date, CURRENT_DATE)
   AND (c.valid_to IS NULL OR c.valid_to >= COALESCE($5::date, CURRENT_DATE))
   AND EXISTS (SELECT 1 FROM items i WHERE i.code = c.item_code AND i.enterprise_id = $1)
UNION ALL
SELECT l.item_code, l.id, l.from_uom, l.to_uom, l.from_quantity, l.to_quantity, 'LOTE', 0, 0, 'VALUE'
  FROM item_lot_unit_conversions l
 WHERE l.item_code = ANY($2) AND l.is_active AND $4 <> '' AND l.lot = $4
   AND EXISTS (SELECT 1 FROM items i WHERE i.code = l.item_code AND i.enterprise_id = $1)
UNION ALL
SELECT i.code, 0, upper(i.warehouse_unit_of_measurement::text), upper(trim(i.warehouse_secondary_uom)), 1, i.warehouse_secondary_factor,
       'SEGUNDA_UNIDADE', 0, 0, 'VALUE'
  FROM items i
 WHERE i.code = ANY($2) AND i.enterprise_id = $1
   AND COALESCE(trim(i.warehouse_secondary_uom), '') <> '' AND i.warehouse_secondary_factor > 0
   AND upper(trim(i.warehouse_secondary_uom)) <> upper(i.warehouse_unit_of_measurement::text)
UNION ALL
SELECT g.item_code, 0, upper(trim(g.uom)), upper(i.warehouse_unit_of_measurement::text), 1, g.package_quantity,
       'EMBALAGEM', 0, 0, 'VALUE'
  FROM item_gtins g
  JOIN items i ON i.code = g.item_code AND i.enterprise_id = g.enterprise_id
 WHERE g.item_code = ANY($2) AND g.enterprise_id = $1 AND g.package_quantity > 0
   AND upper(trim(g.uom)) <> upper(i.warehouse_unit_of_measurement::text)
UNION ALL
SELECT 0, g.id, g.from_uom, g.to_uom, g.from_quantity, g.to_quantity, 'PADRAO', 0, 0, 'VALUE'
  FROM global_unit_conversions g
 WHERE g.enterprise_id = $1 AND g.is_active`

// Carregar devolve as arestas de um item. `at` zero = hoje no relógio do banco.
func Carregar(ctx context.Context, db Querier, empresa, itemCode int64, mascara, lote string, at time.Time) ([]service.Aresta, error) {
	m, err := CarregarItens(ctx, db, empresa, []int64{itemCode}, mascara, lote, at)
	if err != nil {
		return nil, err
	}
	return m[itemCode], nil
}

// CarregarItens: o mesmo para vários itens numa consulta só (listagens).
// Cada item recebe as suas arestas mais as padrão da empresa.
func CarregarItens(ctx context.Context, db Querier, empresa int64, itens []int64, mascara, lote string, at time.Time) (map[int64][]service.Aresta, error) {
	var data *time.Time
	if !at.IsZero() {
		d := time.Date(at.Year(), at.Month(), at.Day(), 0, 0, 0, 0, time.UTC)
		data = &d
	}
	rows, err := db.Query(ctx, arestasSQL, empresa, itens, mascara, lote, data)
	if err != nil {
		return nil, fmt.Errorf("lendo as conversões de unidade: %w", err)
	}
	defer rows.Close()
	porItem := map[int64][]service.Aresta{}
	var padrao []service.Aresta
	for rows.Next() {
		var item int64
		var a service.Aresta
		var origem string
		var arred, tol decimal.Decimal
		if err := rows.Scan(&item, &a.ID, &a.De, &a.Para, &a.QtdDe, &a.QtdPara, &origem, &arred, &tol, &a.TipoTolerancia); err != nil {
			return nil, fmt.Errorf("lendo as conversões de unidade: %w", err)
		}
		a.Origem = service.Origem(origem)
		a.ArredondamentoPct, a.Tolerancia = arred, tol
		if a.Origem == service.OrigemPadrao {
			padrao = append(padrao, a)
			continue
		}
		porItem[item] = append(porItem[item], a)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	out := make(map[int64][]service.Aresta, len(itens))
	for _, it := range itens {
		out[it] = append(append([]service.Aresta{}, porItem[it]...), padrao...)
	}
	return out, nil
}

// Resolver é o atalho Carregar + service.Resolver.
func Resolver(ctx context.Context, db Querier, empresa, itemCode int64, mascara, lote string, at time.Time, de, para string) (*service.Caminho, error) {
	if service.Normalizar(de) == service.Normalizar(para) {
		c, _ := service.Resolver(nil, de, para)
		return &c, nil
	}
	arestas, err := Carregar(ctx, db, empresa, itemCode, mascara, lote, at)
	if err != nil {
		return nil, err
	}
	c, ok := service.Resolver(arestas, de, para)
	if !ok {
		return nil, nil
	}
	return &c, nil
}

// unidadesSQL: código, código fiscal e sinônimos (do fornecedor e gerais) já
// normalizados, para casar o texto recebido.
const unidadesSQL = `
SELECT u.code, u.code, 'CODIGO', 0 FROM units_of_measure u WHERE u.enterprise_id = $1 AND u.is_active
UNION ALL
SELECT a.alias, a.unit_code, CASE WHEN a.supplier_code IS NULL THEN 'SINONIMO' ELSE 'SINONIMO_FORNECEDOR' END,
       CASE WHEN a.supplier_code IS NULL THEN 2 ELSE 1 END
  FROM unit_of_measure_aliases a
  JOIN units_of_measure u ON u.enterprise_id = a.enterprise_id AND u.code = a.unit_code AND u.is_active
 WHERE a.enterprise_id = $1 AND (a.supplier_code IS NULL OR a.supplier_code = $2)
UNION ALL
SELECT u.fiscal_code, u.code, 'CODIGO_FISCAL', 3 FROM units_of_measure u
 WHERE u.enterprise_id = $1 AND u.is_active AND u.fiscal_code <> u.code`

// Normalizar traduz textos recebidos (uCom da NF-e) para o catálogo da
// empresa. Ordem: código exato, sinônimo do fornecedor, sinônimo geral,
// código fiscal. Texto não reconhecido volta com Unidade vazia.
func Normalizar(ctx context.Context, db Querier, empresa int64, fornecedor *int64, textos []string) (map[string]uomrepo.Resolucao, error) {
	out := map[string]uomrepo.Resolucao{}
	if len(textos) == 0 {
		return out, nil
	}
	var forn int64
	if fornecedor != nil {
		forn = *fornecedor
	}
	rows, err := db.Query(ctx, unidadesSQL, empresa, forn)
	if err != nil {
		return nil, fmt.Errorf("lendo o catálogo de unidades: %w", err)
	}
	defer rows.Close()
	type cand struct {
		unidade, via string
		prio         int
	}
	mapa := map[string]cand{}
	for rows.Next() {
		var chave, unidade, via string
		var prio int
		if err := rows.Scan(&chave, &unidade, &via, &prio); err != nil {
			return nil, err
		}
		chave = uomentity.NormalizarTexto(chave)
		if atual, ok := mapa[chave]; !ok || prio < atual.prio {
			mapa[chave] = cand{unidade, via, prio}
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for _, t := range textos {
		r := uomrepo.Resolucao{Texto: t}
		if c, ok := mapa[uomentity.NormalizarTexto(t)]; ok {
			r.Unidade, r.Via = c.unidade, c.via
		}
		out[t] = r
	}
	return out, nil
}
