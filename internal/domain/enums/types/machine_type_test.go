package types

import (
	"encoding/json"
	"testing"
)

func TestMachineTypeEnumUsesDatabaseVocabularyAndValidatesJSON(t *testing.T) {
	for _, value := range []MachineTypeEnum{MachineCut, MachineBend, MachineWeld, MachineAssemble, MachinePaint, MachineLathe, MachineMill, MachineInject, MachinePress} {
		if !value.IsValid() {
			t.Fatalf("expected %s to be valid", value)
		}
		var decoded MachineTypeEnum
		if err := json.Unmarshal([]byte(`"`+string(value)+`"`), &decoded); err != nil {
			t.Fatal(err)
		}
		if decoded != value {
			t.Fatalf("got %s, want %s", decoded, value)
		}
	}
	for _, invalid := range []string{`"CORTE"`, `"CUTTING"`, `""`} {
		var decoded MachineTypeEnum
		if err := json.Unmarshal([]byte(invalid), &decoded); err == nil {
			t.Fatalf("expected %s to be rejected", invalid)
		}
	}
}
