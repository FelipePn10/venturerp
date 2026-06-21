package export

import (
	"encoding/csv"
	"io"
)

// EncodeCSV writes the table as UTF-8 CSV. A BOM is emitted first so Excel on
// Windows opens accented Portuguese text correctly without manual import.
func EncodeCSV(w io.Writer, t *Table) error {
	if err := t.Validate(); err != nil {
		return err
	}
	t.normalize()

	if _, err := w.Write([]byte{0xEF, 0xBB, 0xBF}); err != nil {
		return err
	}

	cw := csv.NewWriter(w)
	cw.Comma = ';' // pt-BR Excel default delimiter

	if err := cw.Write(t.Columns); err != nil {
		return err
	}
	for _, row := range t.Rows {
		if err := cw.Write(row); err != nil {
			return err
		}
	}
	cw.Flush()
	return cw.Error()
}
