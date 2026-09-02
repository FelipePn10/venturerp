# Matriz de publicação — ciclo comercial

Atualizada em 2026-08-31. A matriz separa evidência local de publicação: uma
rota implementada na branch não deve ser apresentada como disponível em demo ou
training antes de uma release imutável ser instalada e verificada.

| Contrato | Local `fix/ajustes-operacionais` | Demo | Training | Migration mínima |
|---|---|---|---|---|
| `PATCH /api/sales-division/{code}/status` | validado | atualizado localmente | atualizado localmente | 324 |
| `DELETE /api/sales-division/{code}` | validado | atualizado localmente | atualizado localmente | 324 |
| `GET /api/delivery-reschedule/preview/{code}` | validado | atualizado localmente | atualizado localmente | 324 |
| `POST /api/delivery-reschedule/batch` | validado | atualizado localmente | atualizado localmente | 324 |
| `GET /api/sales-order/search` | validado | atualizado localmente | atualizado localmente | 324 |
| `GET /api/recurring-sales/{code}` | validado | atualizado localmente | atualizado localmente | 324 |
| `GET /api/representatives/{code}` | validado | atualizado localmente | atualizado localmente | 324 |
| `GET /api/representatives/interest-classifications` | validado | atualizado localmente | atualizado localmente | 324 |
| `GET /api/items/classifications/` | validado | atualizado localmente | atualizado localmente | 324 |
| `PUT /api/customers/support/commercial-policies/{code}` | validado | atualizado localmente | atualizado localmente | 324 |

## Atualização operacional local em 2026-08-31

- `5070`, demo (`5072`) e training (`5073`) executam o mesmo image ID local
  `sha256:8ce77db92a1cfa4cfd75d39a9792022c0a46559d1d47403110876441a63c4369`.
- Os três bancos estão na migration `324`, sem marcador `dirty`.
- Demo e training passaram a suíte comercial autenticada completa.
- O `5070`, que não possui o dataset demo `item_code=10001`, passou o smoke
  autenticado focal com seu item real `50001`: inclusão/resolução de preço,
  catálogo de classificações e filtro de workflow.
- Esta atualização não criou release, tag ou commit e não publicou imagem em
  registry.

## Procedimento de atualização

1. Integrar os PRs e obter `main` limpa.
2. Executar `make release-check VERSION=X.Y.Z` e `make release VERSION=X.Y.Z`.
3. Instalar a mesma tag imutável no ambiente, preservando backup e rollback.
4. Conferir `GET /api/version`, migration aplicada e smoke autenticado.
5. Trocar `pendente de release` por versão/data somente depois de confirmar
   resposta, estado persistido, evento/auditoria e isolamento por empresa.

Uma publicação futura em registry continua seguindo o fluxo de release acima.
A atualização registrada nesta matriz foi apenas operacional/local, conforme
autorização explícita, sem criar uma nova release.
