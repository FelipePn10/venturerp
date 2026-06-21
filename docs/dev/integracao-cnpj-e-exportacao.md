# Integração CNPJ (auto-fill) & Exportação de Relatórios

> 🛠️ Referência técnica. Duas capacidades transversais adicionadas para reduzir
> digitação no cadastro e atender à exigência de relatórios em formatos de
> escritório (Excel/PDF). Ambas são **dependency-free** (sem libs externas) e se
> apoiam apenas na stdlib + vendoring já existente.

---

## 1. Busca automática por CNPJ

Ao informar um CNPJ no cadastro de **cliente**, **fornecedor** ou **empresa**, a
tela chama um endpoint que devolve razão social, nome fantasia, **inscrição
estadual**, endereço completo, CNAE, natureza jurídica, porte e flags de Simples
Nacional / MEI — direto da base da Receita Federal.

### Endpoint

```
GET /api/cnpj/{cnpj}        (papéis: ADMIN, USER)
```

`{cnpj}` aceita com ou sem máscara. O dígito verificador é validado **antes** de
qualquer chamada externa (`validation.ValidateCNPJ`), então CNPJ inválido retorna
`400` sem custo de rede.

Resposta: `response.CNPJLookupResponse` (ver `API_REQUEST_BODIES.txt`).

| Status | Quando |
|---|---|
| `200` | Encontrado. |
| `400` | CNPJ com dígito inválido. |
| `404` | Não existe na base. |
| `502` | Provedor indisponível / rate-limit / timeout. |

### Provedores

| Provedor | Traz IE? | Observação |
|---|---|---|
| **CNPJá Open** (`open.cnpja.com`) | ✅ Sim (`registrations[]`) | Free tier com rate-limit baixo (~5/min). |
| **BrasilAPI** (`brasilapi.com.br`) | ❌ Não | Confiável, completo em endereço/CNAE/Simples. |

`CNPJ_PROVIDER=auto` (default) consulta a **CNPJá primeiro** (para trazer a IE) e,
se ela falhar por indisponibilidade/limite, **cai para a BrasilAPI** — assim o
operador sempre recebe ao menos endereço e razão social. Um `404` genuíno da
primária **não** dispara fallback. O campo `source` na resposta diz quem
respondeu; quando for `"brasilapi"`, a IE não veio e deve ser digitada.

### Configuração (`.env`)

```
CNPJ_PROVIDER=auto                                  # auto | brasilapi | cnpja
CNPJ_BRASILAPI_URL=https://brasilapi.com.br/api/cnpj/v1
CNPJ_CNPJA_URL=https://open.cnpja.com
CNPJ_TIMEOUT_SEC=8
```

### Arquitetura (Clean Architecture)

```
domain/cnpj/entity/company.go      Company, Address, Activity, StateRegistration (modelo neutro)
domain/cnpj/service/provider.go    porta Provider + ErrNotFound / ErrUnavailable
infrastructure/cnpj/               adapters BrasilAPI, CNPJá, chain "auto" + factory New(Config)
application/usecase/cnpj_uc/       LookupCNPJUseCase: valida + delega + mapeia p/ DTO
interfaces/http/handler/cnpj_handler.go   GET /api/cnpj/{cnpj}, mapeia erros → HTTP
```

O domínio não sabe qual API respondeu: cada adapter mapeia sua resposta para a
mesma `entity.Company`. Trocar/empilhar provedores = implementar `Provider`.

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
