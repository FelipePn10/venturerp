# Contabilidade e Complementos Fiscais — Documentação técnica

Complementa [`fiscal-financeiro.md`](fiscal-financeiro.md) com módulos servidos por
rotas próprias: **Contabilidade** (plano de contas, lançamentos, balancete,
demonstrativos, SPED ECD), **NFS-e** (nota de serviço) e **Operações de Entrada**
(natureza fiscal das compras). Versão de negócio:
[`../apresentacao/fiscal-financeiro.md`](../apresentacao/fiscal-financeiro.md).

> Convenções: `Authorization: Bearer <JWT>`, papel `ADMIN`/`USER` (salvo indicado).

---

## 1. Contabilidade (`/api/accounting`)

| Recurso | Rotas |
|---|---|
| Planos de conta | `POST /plans` · `GET /plans` |
| Contas | `POST /accounts` · `GET /accounts` |
| Lançamentos (partidas) | `POST /journal-entries` · `GET /journal-entries` |
| **Balancete** | `GET /balancete` |
| Demonstrativos | `POST /demonstratives` |
| **SPED ECD** | `POST /sped/ecd` (geração do arquivo) |

Cobre a escrituração contábil: plano de contas, lançamentos de débito/crédito,
balancete de verificação, demonstrativos e a obrigação **SPED Contábil (ECD)**.

---

## 2. NFS-e — Nota Fiscal de Serviço (`/api/fiscal/nfse`)

| Método | Rota | Permissão |
|---|---|---|
| POST | `/create` | ADMIN/USER |
| POST | `/{code}/authorize` | `PermFiscalAuthorize` |
| POST | `/{code}/cancel` | `PermFiscalAuthorize` |
| GET | `/list` | ADMIN/USER |
| GET | `/{code}` | ADMIN/USER |

Emissão e cancelamento de notas de serviço junto à prefeitura/integração; mesmo padrão
de ciclo (rascunho → autorizada → cancelada) da NF-e.

---

## 3. Operações de Entrada (`/api/entry-operations`)

Natureza fiscal aplicada nas compras (CFOP, tributação, destino), com grupos de UF.

| Recurso | Rotas |
|---|---|
| Operação | `POST /` · `PUT /` · `GET /` · `GET /{code}` · `GET /{code}/validate` |
| Grupos de estado (UF) | `POST /state-groups` · `GET /state-groups` · `GET /state-groups/{code}` · `POST /state-groups/{code}/ufs` |

A `validate` confere a consistência da operação antes do uso no recebimento.

---

> Demais módulos fiscais (NF-e saída/entrada, CT-e, IBPT, apuração ICMS/IPI/PIS/COFINS,
> ICMS-ST, tabelas NCM/ICMS, CNAB 240, conciliação OFX, relatórios gerenciais,
> adiantamentos) estão em [`fiscal-financeiro.md`](fiscal-financeiro.md).
