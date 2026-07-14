# Integração fiscal & Exportação de Relatórios

> Referência técnica para integrações fiscais e relatórios em formatos de
> escritório (Excel/PDF).

---

## 1. Integração fiscal

O endpoint autenticado `GET /api/cnpj/{cnpj}` consulta o cadastro empresarial
para preencher razão social, inscrição estadual e endereço. A seleção do
provedor não é configurável no `.env`. Emissão, consulta e cancelamento de
documentos fiscais usam exclusivamente a integração Focus NFe e as credenciais
fiscais armazenadas por empresa. O token nunca deve ser versionado.

---

## 2. Exportação de relatórios (Excel / PDF / CSV)

Qualquer dado consultável pode ser baixado em **XLSX (Excel)**, **PDF** ou
**CSV**. O pacote `internal/infrastructure/export` é o motor único; todos os
encoders são escritos à mão (sem dependências):

| Formato | Implementação |
|---|---|
| CSV | `encoding/csv`, delimitador `;` + BOM UTF-8 (Excel pt-BR abre certo). |
| XLSX | OOXML montado via `archive/zip` (inline strings, cabeçalho em negrito). |
| PDF | Escritor PDF 1.4 próprio: Courier monoespaçada, paginação, acentos em WinAnsi. |

### Modelo único — `export.Table`

```go
type Table struct {
    Title, Subtitle string
    Columns         []string
    Rows            [][]string   // cada linha deve ter len(Columns) células
    GeneratedAt     time.Time
}
```

### Duas formas de uso

**(a) Endpoint genérico — o front-end manda as linhas que já exibe:**

```
POST /api/reports/export?format=xlsx|pdf|csv      (papéis: ADMIN, USER)
{ "title": "...", "subtitle": "...", "columns": [...], "rows": [[...]] }
```

Devolve o arquivo com `Content-Disposition: attachment`. Serve para **qualquer**
relatório sem precisar de rota dedicada por módulo.

**(b) `?format=` em endpoints de lista — exporta a consulta real do servidor:**

```
GET /api/customers?format=xlsx
GET /api/suppliers?format=pdf
```

Habilitar em um handler de lista é uma linha, reaproveitando o próprio DTO:

```go
if done, _ := export.WriteSlice(w, r, "Clientes", "clientes", result); done {
    return // exportou arquivo; sem o parâmetro cai no JSON normal
}
```

`export.TableFromSlice` reflete o slice de structs usando as tags `json` como
cabeçalho (ignora `json:"-"`, achata `*T`, formata `time.Time` em `dd/mm/aaaa` e
booleanos como `Sim/Não`). Para adotar em uma nova lista, basta o guard acima.

### Arquivos

```
infrastructure/export/table.go     Table, Format, ContentType/Extension, normalize
infrastructure/export/csv.go       EncodeCSV
infrastructure/export/xlsx.go      EncodeXLSX (zip OOXML)
infrastructure/export/pdf.go       EncodePDF (paginação + WinAnsi)
infrastructure/export/reflect.go   TableFromSlice (structs/maps → Table)
infrastructure/export/http.go      Requested / WriteHTTP / WriteSlice
interfaces/http/handler/report_export_handler.go   POST /api/reports/export
```

### Testes

`export_test.go` valida BOM/CSV, XLSX como ZIP válido com as partes OOXML
obrigatórias, estrutura do PDF (header/xref/EOF) e a reflexão de structs.
`cnpj_test.go` cobre parsing de cada provedor, fallback do modo `auto` e
propagação de `ErrNotFound`, tudo via `httptest` (sem rede real).
