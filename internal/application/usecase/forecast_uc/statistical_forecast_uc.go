package forecast_uc

import (
	"fmt"
	"math"
)

// Model names
const (
	ModelMovingAverage = "MOVING_AVERAGE"
	ModelExpSmoothing  = "EXP_SMOOTHING"
	ModelHoltWinters   = "HOLT_WINTERS"
	ModelAuto          = "AUTO"
)

// DataPoint is a single historical observation.
type DataPoint struct {
	Period   int     `json:"period"` // e.g. 1, 2, 3 ... N
	Quantity float64 `json:"quantity"`
}

// ForecastResult contains the chosen model's forecasts and its MAPE.
type ForecastResult struct {
	Model     string    `json:"model"`
	MAPE      float64   `json:"mape_pct"`  // Mean Absolute Percentage Error
	Forecasts []float64 `json:"forecasts"` // qty per future period
}

type StatisticalForecastDTO struct {
	ItemCode  int64       `json:"item_code"`
	History   []DataPoint `json:"history"`    // historical demand, ordered by period asc
	Periods   int         `json:"periods"`    // how many future periods to forecast
	Model     string      `json:"model"`      // AUTO | MOVING_AVERAGE | EXP_SMOOTHING | HOLT_WINTERS
	MAWindow  int         `json:"ma_window"`  // for moving average (default 3)
	Alpha     float64     `json:"alpha"`      // for exp smoothing / HW (0..1, default 0.3)
	Beta      float64     `json:"beta"`       // for HW trend (0..1, default 0.1)
	Gamma     float64     `json:"gamma"`      // for HW season (0..1, default 0.1)
	SeasonLen int         `json:"season_len"` // HW season length (default 12)
}

type StatisticalForecastResponse struct {
	ItemCode int64            `json:"item_code"`
	Result   ForecastResult   `json:"result"`
	All      []ForecastResult `json:"all_models,omitempty"` // all model results for AUTO
}

// Execute picks the best model (or uses the requested one) and returns forecasts.
func Execute(dto StatisticalForecastDTO) (*StatisticalForecastResponse, error) {
	if len(dto.History) < 3 {
		return nil, fmt.Errorf("at least 3 historical data points required")
	}
	if dto.Periods <= 0 {
		dto.Periods = 3
	}

	// defaults
	if dto.MAWindow <= 0 {
		dto.MAWindow = 3
	}
	if dto.Alpha <= 0 || dto.Alpha >= 1 {
		dto.Alpha = 0.3
	}
	if dto.Beta <= 0 || dto.Beta >= 1 {
		dto.Beta = 0.1
	}
	if dto.Gamma <= 0 || dto.Gamma >= 1 {
		dto.Gamma = 0.1
	}
	if dto.SeasonLen <= 0 {
		dto.SeasonLen = 12
	}

	values := extractValues(dto.History)

	switch dto.Model {
	case ModelMovingAverage:
		result := runMovingAverage(values, dto.MAWindow, dto.Periods)
		return &StatisticalForecastResponse{ItemCode: dto.ItemCode, Result: result}, nil
	case ModelExpSmoothing:
		result := runExpSmoothing(values, dto.Alpha, dto.Periods)
		return &StatisticalForecastResponse{ItemCode: dto.ItemCode, Result: result}, nil
	case ModelHoltWinters:
		result := runHoltWinters(values, dto.Alpha, dto.Beta, dto.Gamma, dto.SeasonLen, dto.Periods)
		return &StatisticalForecastResponse{ItemCode: dto.ItemCode, Result: result}, nil
	default: // AUTO
		all := []ForecastResult{
			runMovingAverage(values, dto.MAWindow, dto.Periods),
			runExpSmoothing(values, dto.Alpha, dto.Periods),
			runHoltWinters(values, dto.Alpha, dto.Beta, dto.Gamma, dto.SeasonLen, dto.Periods),
		}
		best := all[0]
		for _, r := range all[1:] {
			if r.MAPE < best.MAPE {
				best = r
			}
		}
		return &StatisticalForecastResponse{ItemCode: dto.ItemCode, Result: best, All: all}, nil
	}
}

// ─── Moving Average ───────────────────────────────────────────────────────────

func runMovingAverage(values []float64, n, periods int) ForecastResult {
	if n > len(values) {
		n = len(values)
	}
	holdout := n
	training := values[:len(values)-holdout]
	test := values[len(values)-holdout:]

	mape := mapeOnHoldout(training, test, func(hist []float64, step int) float64 {
		window := append(hist, make([]float64, 0)...)
		if len(window) < n {
			return avg(window)
		}
		return avg(window[len(window)-n:])
	})

	// Forecast future: extend with predictions
	hist := append([]float64(nil), values...)
	forecasts := make([]float64, periods)
	for i := range forecasts {
		window := hist
		if len(window) >= n {
			window = hist[len(hist)-n:]
		}
		f := avg(window)
		forecasts[i] = math.Max(0, f)
		hist = append(hist, f)
	}

	return ForecastResult{Model: ModelMovingAverage, MAPE: mape, Forecasts: forecasts}
}

// ─── Exponential Smoothing (Simple / Holt single) ────────────────────────────

func runExpSmoothing(values []float64, alpha float64, periods int) ForecastResult {
	holdout := 3
	if holdout >= len(values) {
		holdout = 1
	}
	training := values[:len(values)-holdout]
	test := values[len(values)-holdout:]

	mape := mapeOnHoldout(training, test, func(hist []float64, step int) float64 {
		return exponentialSmooth(hist, alpha)
	})

	// Forecast
	hist := append([]float64(nil), values...)
	forecasts := make([]float64, periods)
	for i := range forecasts {
		f := exponentialSmooth(hist, alpha)
		forecasts[i] = math.Max(0, f)
		hist = append(hist, f)
	}

	return ForecastResult{Model: ModelExpSmoothing, MAPE: mape, Forecasts: forecasts}
}

func exponentialSmooth(values []float64, alpha float64) float64 {
	if len(values) == 0 {
		return 0
	}
	s := values[0]
	for _, v := range values[1:] {
		s = alpha*v + (1-alpha)*s
	}
	return s
}

// ─── Holt-Winters (additive seasonal) ────────────────────────────────────────

func runHoltWinters(values []float64, alpha, beta, gamma float64, seasonLen, periods int) ForecastResult {
	if len(values) < seasonLen*2 {
		// Fall back to exp smoothing when not enough data for HW
		r := runExpSmoothing(values, alpha, periods)
		r.Model = ModelHoltWinters
		return r
	}

	holdout := seasonLen
	training := values[:len(values)-holdout]
	test := values[len(values)-holdout:]

	mape := mapeOnHoldout(training, test, func(hist []float64, step int) float64 {
		_, _, _, fc := holtWintersFit(hist, alpha, beta, gamma, seasonLen, step)
		if len(fc) == 0 {
			return 0
		}
		return fc[len(fc)-1]
	})

	_, _, _, forecasts := holtWintersFit(values, alpha, beta, gamma, seasonLen, periods)
	capped := make([]float64, len(forecasts))
	for i, f := range forecasts {
		capped[i] = math.Max(0, f)
	}

	return ForecastResult{Model: ModelHoltWinters, MAPE: mape, Forecasts: capped}
}

// holtWintersFit runs the additive Holt-Winters algorithm and returns
// level, trend, seasonal indices, and `forecastPeriods` ahead values.
func holtWintersFit(values []float64, alpha, beta, gamma float64, m, forecastPeriods int) (
	level float64, trend float64, seasonal []float64, forecasts []float64,
) {
	n := len(values)
	// Initial level: average of first season
	level = avg(values[:m])
	// Initial trend: difference of season averages / m
	if n >= 2*m {
		trend = (avg(values[m:2*m]) - avg(values[:m])) / float64(m)
	}
	// Initial seasonal: ratio of each period to initial level
	seasonal = make([]float64, m)
	for i := 0; i < m; i++ {
		seasonal[i] = values[i] - level
	}

	// Fit
	smoothed := make([]float64, n)
	for t, v := range values {
		prevLevel := level
		prevTrend := trend
		s := seasonal[t%m]
		level = alpha*(v-s) + (1-alpha)*(prevLevel+prevTrend)
		trend = beta*(level-prevLevel) + (1-beta)*prevTrend
		seasonal[t%m] = gamma*(v-level) + (1-gamma)*s
		smoothed[t] = level + trend + seasonal[t%m]
	}
	_ = smoothed

	// Forecast
	forecasts = make([]float64, forecastPeriods)
	for h := 1; h <= forecastPeriods; h++ {
		sIdx := (n + h - 1) % m
		forecasts[h-1] = level + float64(h)*trend + seasonal[sIdx]
	}
	return level, trend, seasonal, forecasts
}

// ─── MAPE helper ─────────────────────────────────────────────────────────────

func mapeOnHoldout(training, test []float64, predict func(hist []float64, step int) float64) float64 {
	if len(test) == 0 {
		return 0
	}
	hist := append([]float64(nil), training...)
	sum := 0.0
	count := 0
	for i, actual := range test {
		fc := predict(hist, i+1)
		if actual != 0 {
			sum += math.Abs((actual - fc) / actual)
			count++
		}
		hist = append(hist, actual)
	}
	if count == 0 {
		return 0
	}
	return (sum / float64(count)) * 100
}

// ─── utilities ────────────────────────────────────────────────────────────────

func extractValues(pts []DataPoint) []float64 {
	vs := make([]float64, len(pts))
	for i, p := range pts {
		vs[i] = p.Quantity
	}
	return vs
}

func avg(vs []float64) float64 {
	if len(vs) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range vs {
		sum += v
	}
	return sum / float64(len(vs))
}
