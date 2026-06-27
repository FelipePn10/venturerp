// Package lp is a small, dependency-free linear-programming solver used by the
// cutting-plan column-generation optimiser. It solves
//
//	minimise   cᵀx
//	subject to A x {≤,=,≥} b
//	           x ≥ 0
//
// with a two-phase primal simplex (Bland's rule for anti-cycling) over a dense
// tableau, and — crucially for column generation — it returns the optimal dual
// vector alongside the primal solution. Problem sizes here are small (a handful
// of demand and stock constraints, a few dozen pattern columns), so a dense
// tableau is both simple and fast.
//
// The solver is deliberately generic and self-contained: it has no knowledge of
// cutting plans, so it stays independently unit-testable (the tests assert strong
// duality, the property that makes the returned duals trustworthy for pricing).
package lp

import "math"

// ConstraintType is the relational operator of a constraint row.
type ConstraintType int

const (
	LE ConstraintType = iota // a·x ≤ b
	GE                       // a·x ≥ b
	EQ                       // a·x = b
)

// Status is the outcome of a solve.
type Status int

const (
	Optimal Status = iota
	Infeasible
	Unbounded
)

// Problem is a minimisation LP in canonical row form. C has one entry per
// structural variable; A/B/Type describe the constraints (A is m×n). All
// structural variables are implicitly ≥ 0.
type Problem struct {
	C    []float64
	A    [][]float64
	B    []float64
	Type []ConstraintType
}

// Result carries the primal solution, objective and the dual prices (one per
// constraint, aligned with the input rows).
type Result struct {
	Status Status
	X      []float64
	Obj    float64
	Dual   []float64
}

const (
	eps     = 1e-9
	feasTol = 1e-7
	maxIter = 200000
)

// Solve runs the two-phase simplex and returns the optimal primal/dual solution.
func Solve(p Problem) Result {
	m := len(p.A)
	n := len(p.C)

	A := make([][]float64, m)
	b := make([]float64, m)
	ctype := make([]ConstraintType, m)
	flipped := make([]bool, m)
	for i := 0; i < m; i++ {
		row := make([]float64, n)
		copy(row, p.A[i])
		bi := p.B[i]
		t := p.Type[i]
		if bi < 0 {
			for j := range row {
				row[j] = -row[j]
			}
			bi = -bi
			switch t {
			case LE:
				t = GE
			case GE:
				t = LE
			}
			flipped[i] = true
		}
		A[i], b[i], ctype[i] = row, bi, t
	}

	// Allocate logical columns: a slack (+1) for ≤, a surplus (−1) plus an
	// artificial (+1) for ≥, an artificial (+1) for =. Every row therefore owns a
	// column whose original vector is +eᵢ (slack for ≤, artificial otherwise); the
	// current tableau column under that variable is the i-th column of B⁻¹, which
	// is exactly what we need to read the duals back without any sign bookkeeping.
	cols := n
	slackIdx := make([]int, m)
	artIdx := make([]int, m)
	unitVar := make([]int, m) // the +eᵢ column for row i
	for i := range slackIdx {
		slackIdx[i], artIdx[i] = -1, -1
	}
	for i := 0; i < m; i++ {
		switch ctype[i] {
		case LE:
			slackIdx[i] = cols
			cols++
		case GE:
			cols++ // surplus
			artIdx[i] = cols
			cols++
		case EQ:
			artIdx[i] = cols
			cols++
		}
	}

	T := make([][]float64, m)
	rhs := make([]float64, m)
	basis := make([]int, m)
	isArtificial := make([]bool, cols)
	for i := 0; i < m; i++ {
		T[i] = make([]float64, cols)
		copy(T[i], A[i])
		rhs[i] = b[i]
		switch ctype[i] {
		case LE:
			T[i][slackIdx[i]] = 1
			unitVar[i] = slackIdx[i]
			basis[i] = slackIdx[i]
		case GE:
			surplus := artIdx[i] - 1 // surplus column was allocated just before the artificial
			T[i][surplus] = -1
			T[i][artIdx[i]] = 1
			unitVar[i] = artIdx[i]
			basis[i] = artIdx[i]
			isArtificial[artIdx[i]] = true
		case EQ:
			T[i][artIdx[i]] = 1
			unitVar[i] = artIdx[i]
			basis[i] = artIdx[i]
			isArtificial[artIdx[i]] = true
		}
	}

	hasArtificial := false
	for _, a := range isArtificial {
		if a {
			hasArtificial = true
			break
		}
	}

	// Phase 1: minimise the sum of artificials to reach a feasible basis.
	if hasArtificial {
		c1 := make([]float64, cols)
		for j := 0; j < cols; j++ {
			if isArtificial[j] {
				c1[j] = 1
			}
		}
		if simplexLoop(T, rhs, basis, c1, m, cols, nil) == Unbounded {
			return Result{Status: Infeasible}
		}
		obj1 := 0.0
		for i := 0; i < m; i++ {
			obj1 += c1[basis[i]] * rhs[i]
		}
		if obj1 > feasTol {
			return Result{Status: Infeasible}
		}
		// Pivot any artificial still basic out of the basis when a real column can
		// replace it; an irreducible one marks a redundant row and stays basic at 0.
		for i := 0; i < m; i++ {
			if !isArtificial[basis[i]] {
				continue
			}
			for j := 0; j < cols; j++ {
				if isArtificial[j] || math.Abs(T[i][j]) <= eps {
					continue
				}
				pivot(T, rhs, basis, i, j, m, cols)
				break
			}
		}
	}

	// Phase 2: minimise the real objective, barring artificials from re-entering.
	c2 := make([]float64, cols)
	for j := 0; j < n; j++ {
		c2[j] = p.C[j]
	}
	if simplexLoop(T, rhs, basis, c2, m, cols, isArtificial) == Unbounded {
		return Result{Status: Unbounded}
	}

	x := make([]float64, n)
	for i := 0; i < m; i++ {
		if basis[i] < n {
			x[basis[i]] = rhs[i]
		}
	}
	obj := 0.0
	for j := 0; j < n; j++ {
		obj += p.C[j] * x[j]
	}

	// Duals: y = c_Bᵀ B⁻¹, with B⁻¹'s i-th column equal to the tableau column under
	// row i's +eᵢ logical variable. Flip back the rows we normalised.
	dual := make([]float64, m)
	for i := 0; i < m; i++ {
		s := 0.0
		for r := 0; r < m; r++ {
			s += c2[basis[r]] * T[r][unitVar[i]]
		}
		if flipped[i] {
			s = -s
		}
		dual[i] = s
	}

	return Result{Status: Optimal, X: x, Obj: obj, Dual: dual}
}

// simplexLoop runs primal simplex iterations until optimal or unbounded. Entering
// and leaving choices both use Bland's rule (smallest index) so the method cannot
// cycle on degenerate problems. forbid, when non-nil, marks columns barred from
// entering (the artificials in phase 2).
func simplexLoop(T [][]float64, rhs []float64, basis []int, c []float64, m, cols int, forbid []bool) Status {
	for iter := 0; iter < maxIter; iter++ {
		entering := -1
		for j := 0; j < cols; j++ {
			if forbid != nil && forbid[j] {
				continue
			}
			z := 0.0
			for r := 0; r < m; r++ {
				z += c[basis[r]] * T[r][j]
			}
			if c[j]-z < -eps { // negative reduced cost improves a minimisation
				entering = j
				break // Bland: first improving column
			}
		}
		if entering == -1 {
			return Optimal
		}

		leaving, best := -1, math.Inf(1)
		for i := 0; i < m; i++ {
			if T[i][entering] <= eps {
				continue
			}
			ratio := rhs[i] / T[i][entering]
			if ratio < best-eps || (math.Abs(ratio-best) <= eps && (leaving == -1 || basis[i] < basis[leaving])) {
				best, leaving = ratio, i
			}
		}
		if leaving == -1 {
			return Unbounded
		}
		pivot(T, rhs, basis, leaving, entering, m, cols)
	}
	return Optimal
}

// pivot performs a Gauss-Jordan pivot on (r, c), updating the tableau, RHS and the
// basis membership.
func pivot(T [][]float64, rhs []float64, basis []int, r, c, m, cols int) {
	pv := T[r][c]
	inv := 1 / pv
	for j := 0; j < cols; j++ {
		T[r][j] *= inv
	}
	rhs[r] *= inv
	for i := 0; i < m; i++ {
		if i == r {
			continue
		}
		f := T[i][c]
		if f == 0 {
			continue
		}
		for j := 0; j < cols; j++ {
			T[i][j] -= f * T[r][j]
		}
		rhs[i] -= f * rhs[r]
	}
	basis[r] = c
}
