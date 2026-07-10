package restriction_uc

import (
	"testing"

	"github.com/FelipePn10/panossoerp/internal/domain/restriction/entity"
)

func strp(s string) *string { return &s }

func TestDeterminantsSatisfied(t *testing.T) {
	answers := map[int64]string{10: "AZ", 20: "QUAD"} // COR=AZ, TAMPA=QUAD

	// EQUAL dependency: TAMPA must be RED → violated (QUAD) → invalid.
	eq := []*entity.RestrictionDeterminant{{QuestionID: 20, Operator: entity.OperatorEqual, AnswerValue: strp("RED")}}
	if determinantsSatisfied(eq, answers) {
		t.Error("EQUAL RED com TAMPA=QUAD deveria invalidar")
	}
	// EQUAL satisfied when TAMPA=RED.
	if !determinantsSatisfied(eq, map[int64]string{10: "AZ", 20: "RED"}) {
		t.Error("EQUAL RED com TAMPA=RED deveria validar")
	}
	// INVALID determinant always forbids.
	inv := []*entity.RestrictionDeterminant{{QuestionID: 20, Operator: entity.OperatorInvalid}}
	if determinantsSatisfied(inv, answers) {
		t.Error("INVALID deveria proibir a combinação")
	}
	// DIFFERENT: TAMPA must differ from QUAD → violated.
	diff := []*entity.RestrictionDeterminant{{QuestionID: 20, Operator: entity.OperatorDifferent, AnswerValue: strp("QUAD")}}
	if determinantsSatisfied(diff, answers) {
		t.Error("DIFFERENT QUAD com TAMPA=QUAD deveria invalidar")
	}
	// BELONGS: TAMPA ∈ {RED,QUAD} → satisfied.
	bel := []*entity.RestrictionDeterminant{{QuestionID: 20, Operator: entity.OperatorBelongs, AnswerValue: strp("RED,QUAD")}}
	if !determinantsSatisfied(bel, answers) {
		t.Error("BELONGS {RED,QUAD} com TAMPA=QUAD deveria validar")
	}
	// NOT_BELONGS: TAMPA ∉ {RED} → satisfied (QUAD not in list).
	nb := []*entity.RestrictionDeterminant{{QuestionID: 20, Operator: entity.OperatorNotBelongs, AnswerValue: strp("RED")}}
	if !determinantsSatisfied(nb, answers) {
		t.Error("NOT_BELONGS {RED} com TAMPA=QUAD deveria validar")
	}
}
