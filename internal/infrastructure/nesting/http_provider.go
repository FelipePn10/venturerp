// Package nesting holds the external true-shape nesting adapter. It lets a
// dedicated nesting engine (e.g. a DeepNest / ProNest microservice) perform the
// expensive irregular nesting, while the ERP stays agnostic behind the
// service.CuttingOptimizer contract.
//
// The adapter speaks a small, documented JSON protocol:
//
//	POST {url}
//	  { "params": {kerf, trim},
//	    "parts":  [{id,label,qty,allow_rotation,width,height,polygon:[{x,y}...]}],
//	    "sheets": [{id,width,height,qty,is_remnant,priority}] }
//	→ 200
//	  { "sheets": [{stock_id,width,height,repeat,is_remnant,used_area,
//	                placements:[{part_id,label,x,y,width,height,rotation_deg}]}],
//	    "unplaced": [{label,width,height,qty}] }
//
// Any service implementing this contract becomes a drop-in true-shape engine.
package nesting

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/FelipePn10/panossoerp/internal/domain/cutting_plan/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/cutting_plan/service"
)

// HTTPNestingProvider calls an external nesting service over HTTP.
type HTTPNestingProvider struct {
	url    string
	client *http.Client
}

// NewHTTPProvider builds an external true-shape provider pointed at `url`.
func NewHTTPProvider(url string) *HTTPNestingProvider {
	return &HTTPNestingProvider{
		url:    url,
		client: &http.Client{Timeout: 60 * time.Second},
	}
}

func (p *HTTPNestingProvider) Type() entity.CutType { return entity.CutTypeTrueShape2D }

// ─── wire protocol ────────────────────────────────────────────────────────────

type wirePoint struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

type wirePart struct {
	ID            int64       `json:"id"`
	Label         string      `json:"label"`
	Qty           int         `json:"qty"`
	AllowRotation bool        `json:"allow_rotation"`
	Width         float64     `json:"width"`
	Height        float64     `json:"height"`
	Polygon       []wirePoint `json:"polygon,omitempty"`
}

type wireSheet struct {
	ID        int64   `json:"id"`
	Width     float64 `json:"width"`
	Height    float64 `json:"height"`
	Qty       int     `json:"qty"`
	IsRemnant bool    `json:"is_remnant"`
	Priority  int     `json:"priority"`
}

type wireRequest struct {
	Params struct {
		Kerf float64 `json:"kerf"`
		Trim float64 `json:"trim"`
	} `json:"params"`
	Parts  []wirePart  `json:"parts"`
	Sheets []wireSheet `json:"sheets"`
}

type wirePlacement struct {
	PartID      int64   `json:"part_id"`
	Label       string  `json:"label"`
	X           float64 `json:"x"`
	Y           float64 `json:"y"`
	Width       float64 `json:"width"`
	Height      float64 `json:"height"`
	RotationDeg float64 `json:"rotation_deg"`
}

type wireSheetResult struct {
	StockID    int64           `json:"stock_id"`
	Width      float64         `json:"width"`
	Height     float64         `json:"height"`
	Repeat     int             `json:"repeat"`
	IsRemnant  bool            `json:"is_remnant"`
	UsedArea   float64         `json:"used_area"`
	Placements []wirePlacement `json:"placements"`
}

type wireUnplaced struct {
	Label  string  `json:"label"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
	Qty    int     `json:"qty"`
}

type wireResponse struct {
	Sheets   []wireSheetResult `json:"sheets"`
	Unplaced []wireUnplaced    `json:"unplaced"`
}

// Optimize implements service.CuttingOptimizer by delegating to the remote engine.
func (p *HTTPNestingProvider) Optimize(demand []service.DemandPiece, stock []service.StockPiece, params service.CutParams) (*service.Solution, error) {
	req := wireRequest{Parts: make([]wirePart, 0, len(demand)), Sheets: make([]wireSheet, 0, len(stock))}
	req.Params.Kerf = params.Kerf
	req.Params.Trim = params.Trim
	for _, d := range demand {
		wp := wirePart{ID: d.PartID, Label: d.Label, Qty: d.Qty, AllowRotation: d.AllowRotation, Width: d.Width, Height: d.Height}
		for _, pt := range d.Polygon {
			wp.Polygon = append(wp.Polygon, wirePoint{X: pt.X, Y: pt.Y})
		}
		req.Parts = append(req.Parts, wp)
	}
	for _, s := range stock {
		req.Sheets = append(req.Sheets, wireSheet{
			ID: s.StockID, Width: s.Width, Height: s.Height, Qty: s.Qty, IsRemnant: s.IsRemnant, Priority: s.Priority,
		})
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("encoding nesting request: %w", err)
	}
	httpReq, err := http.NewRequestWithContext(context.Background(), http.MethodPost, p.url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("building nesting request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("calling nesting service: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("nesting service returned %d", resp.StatusCode)
	}

	var out wireResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("decoding nesting response: %w", err)
	}
	return p.toSolution(out), nil
}

func (p *HTTPNestingProvider) toSolution(out wireResponse) *service.Solution {
	sol := &service.Solution{}
	var totalDemand, totalStock float64
	for _, sh := range out.Sheets {
		repeat := sh.Repeat
		if repeat <= 0 {
			repeat = 1
		}
		pat := service.Pattern{
			StockID: sh.StockID, IsRemnant: sh.IsRemnant, Repeat: repeat,
			StockWidth: sh.Width, StockHeight: sh.Height, UsedArea: sh.UsedArea,
		}
		for _, pl := range sh.Placements {
			pat.Placements = append(pat.Placements, service.Placement{
				PartID: pl.PartID, Label: pl.Label, X: pl.X, Y: pl.Y, W: pl.Width, H: pl.Height,
				Rotated: pl.RotationDeg != 0, RotationDeg: pl.RotationDeg,
			})
		}
		sol.Patterns = append(sol.Patterns, pat)
		totalDemand += sh.UsedArea * float64(repeat)
		totalStock += sh.Width * sh.Height * float64(repeat)
		sol.StockUsed += repeat
		sol.CutCount += len(pat.Placements) * repeat
	}
	for _, u := range out.Unplaced {
		qty := u.Qty
		if qty <= 0 {
			qty = 1
		}
		sol.Unplaced = append(sol.Unplaced, service.DemandPiece{Label: u.Label, Width: u.Width, Height: u.Height, Qty: qty})
	}
	sol.TotalDemand = totalDemand
	sol.TotalStock = totalStock
	if totalStock > 0 {
		sol.Utilization = totalDemand / totalStock
	}
	return sol
}
