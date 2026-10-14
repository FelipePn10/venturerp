package handler

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"reflect"
	"regexp"
	"strings"
	"time"
)

var soData = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)

// decodeEntityBody decodifica o corpo numa entidade fiscal aceitando data sem
// hora ("2026-10-12") nos campos time.Time — é o que o campo de data da tela
// manda. Só os campos de data da struct são completados (descobertos por
// reflexão): um texto que por acaso pareça data continua texto.
func decodeEntityBody(r *http.Request, dst any) error {
	raw, err := io.ReadAll(r.Body)
	if err != nil {
		return err
	}
	var campos map[string]json.RawMessage
	if json.Unmarshal(raw, &campos) == nil && campos != nil {
		alterou := false
		for nome := range camposDeData(dst) {
			valor, ok := campos[nome]
			if !ok {
				continue
			}
			var texto string
			if json.Unmarshal(valor, &texto) != nil {
				continue
			}
			if soData.MatchString(texto) {
				campos[nome], _ = json.Marshal(texto + "T00:00:00Z")
				alterou = true
			}
		}
		if alterou {
			if raw, err = json.Marshal(campos); err != nil {
				return err
			}
		}
	}
	return json.NewDecoder(bytes.NewReader(raw)).Decode(dst)
}

var tipoTime = reflect.TypeOf(time.Time{})

func camposDeData(dst any) map[string]struct{} {
	out := map[string]struct{}{}
	t := reflect.TypeOf(dst)
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return out
	}
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		ft := f.Type
		if ft.Kind() == reflect.Pointer {
			ft = ft.Elem()
		}
		if ft != tipoTime {
			continue
		}
		nome := strings.Split(f.Tag.Get("json"), ",")[0]
		if nome == "" || nome == "-" {
			nome = f.Name
		}
		out[nome] = struct{}{}
	}
	return out
}

// respondFiscalError mantém o nome usado pelos handlers fiscais.
func respondFiscalError(w http.ResponseWriter, err error) { respondErr(w, err) }
