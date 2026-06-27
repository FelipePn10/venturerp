package lp

import (
	"math"
	"testing"
)

func approx(a, b float64) bool { return math.Abs(a-b) < 1e-6 }

func dot(a, b []float64) float64 {
	s := 0.0
	for i := range a {
		s += a[i] * b[i]
	}
	return s
}

// strongDuality is the property that validates the returned duals: at optimality
// the primal objective equals the dual objective yᵀb.
func checkStrongDuality(t *testing.T, p Problem, r Result) {
	t.Helper()
	if r.Status != Optimal {
		t.Fatalf("status = %v, want Optimal", r.Status)
	}
	if !approx(r.Obj, dot(r.Dual, p.B)) {
		t.Errorf("strong duality violated: primal obj = %v, dual obj yᵀb = %v", r.Obj, dot(r.Dual, p.B))
	}
}

func TestSolve_GEConstraintsKnownOptimum(t *testing.T) {
	p := Problem{
		C:    []float64{1, 1},
		A:    [][]float64{{1, 2}, {2, 1}},
		B:    []float64{3, 3},
		Type: []ConstraintType{GE, GE},
	}
	r := Solve(p)
	if !approx(r.Obj, 2) {
		t.Fatalf("obj = %v, want 2", r.Obj)
	}
	if !approx(r.X[0], 1) || !approx(r.X[1], 1) {
		t.Fatalf("x = %v, want [1 1]", r.X)
	}
	if !approx(r.Dual[0], 1.0/3) || !approx(r.Dual[1], 1.0/3) {
		t.Fatalf("dual = %v, want [1/3 1/3]", r.Dual)
	}
	checkStrongDuality(t, p, r)
}

func TestSolve_LEMaximisationViaNegation(t *testing.T) {
	// maximise x1+x2  ≡  minimise -x1-x2
	p := Problem{
		C:    []float64{-1, -1},
		A:    [][]float64{{1, 1}, {1, 0}, {0, 1}},
		B:    []float64{4, 3, 3},
		Type: []ConstraintType{LE, LE, LE},
	}
	r := Solve(p)
	if !approx(r.Obj, -4) {
		t.Fatalf("obj = %v, want -4", r.Obj)
	}
	checkStrongDuality(t, p, r)
}

func TestSolve_EqualityConstraint(t *testing.T) {
	p := Problem{
		C:    []float64{1, 1},
		A:    [][]float64{{1, 1}},
		B:    []float64{5},
		Type: []ConstraintType{EQ},
	}
	r := Solve(p)
	if !approx(r.Obj, 5) {
		t.Fatalf("obj = %v, want 5", r.Obj)
	}
	if !approx(r.Dual[0], 1) {
		t.Fatalf("dual = %v, want [1]", r.Dual)
	}
	checkStrongDuality(t, p, r)
}

func TestSolve_MixedConstraintsStrongDuality(t *testing.T) {
	p := Problem{
		C:    []float64{3, 2, 4},
		A:    [][]float64{{1, 1, 1}, {2, 1, 0}, {0, 1, 3}},
		B:    []float64{10, 8, 6},
		Type: []ConstraintType{GE, LE, GE},
	}
	r := Solve(p)
	checkStrongDuality(t, p, r)
	// feasibility of the returned primal
	for i, row := range p.A {
		lhs := dot(row, r.X)
		switch p.Type[i] {
		case GE:
			if lhs < p.B[i]-1e-6 {
				t.Errorf("row %d: %v >= %v violated", i, lhs, p.B[i])
			}
		case LE:
			if lhs > p.B[i]+1e-6 {
				t.Errorf("row %d: %v <= %v violated", i, lhs, p.B[i])
			}
		}
	}
}

func TestSolve_NegativeRHSNormalisation(t *testing.T) {
	// −x1 − x2 ≤ −3  is  x1 + x2 ≥ 3 after normalisation.
	p := Problem{
		C:    []float64{1, 1},
		A:    [][]float64{{-1, -1}},
		B:    []float64{-3},
		Type: []ConstraintType{LE},
	}
	r := Solve(p)
	if !approx(r.Obj, 3) {
		t.Fatalf("obj = %v, want 3", r.Obj)
	}
	checkStrongDuality(t, p, r)
}

func TestSolve_Infeasible(t *testing.T) {
	p := Problem{
		C:    []float64{1, 1},
		A:    [][]float64{{1, 1}, {1, 1}},
		B:    []float64{1, 5},
		Type: []ConstraintType{LE, GE},
	}
	if r := Solve(p); r.Status != Infeasible {
		t.Fatalf("status = %v, want Infeasible", r.Status)
	}
}

func TestSolve_Unbounded(t *testing.T) {
	// minimise -x1 with only x1 >= 1 → unbounded below.
	p := Problem{
		C:    []float64{-1},
		A:    [][]float64{{1}},
		B:    []float64{1},
		Type: []ConstraintType{GE},
	}
	if r := Solve(p); r.Status != Unbounded {
		t.Fatalf("status = %v, want Unbounded", r.Status)
	}
}

func TestSolve_DegenerateNoCycle(t *testing.T) {
	// A degenerate problem; Bland's rule must terminate.
	p := Problem{
		C:    []float64{-0.75, 150, -0.02, 6},
		A:    [][]float64{{0.25, -60, -0.04, 9}, {0.5, -90, -0.02, 3}, {0, 0, 1, 0}},
		B:    []float64{0, 0, 1},
		Type: []ConstraintType{LE, LE, LE},
	}
	r := Solve(p)
	if r.Status != Optimal {
		t.Fatalf("status = %v, want Optimal (no cycling)", r.Status)
	}
	checkStrongDuality(t, p, r)
}
