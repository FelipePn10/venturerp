# Cadastro de Transportadora (VSUP0140)

A transportadora é um **fornecedor** (o frete é pago pelo contas a pagar de
sempre) com um perfil de transporte ao lado: `shipping_carriers` e filhas.
Não confundir com `carriers`, que é o **portador** financeiro.

Convenção de empresa: `enterprise_id` em todas as tabelas (a do pai, `suppliers`).

## O que o cadastro guarda

| Grupo | Campos |
|---|---|
| Habilitação | RNTRC (8 dígitos) e validade, categoria ANTT (ETC/CTC/TAC), modal, emite CT-e |
| Tabela de frete | piso, R$/kg, ad valorem %, GRIS %, pedágio por 100 kg, **fator de cubagem** (kg/m³), **taxa de despacho** por embarque |
| Restrições | **aceita produto perigoso** |
| Seguro | seguradora, apólice, validade, cobertura |
| Frota | placa + **UF**, **RENAVAM** (dígito conferido), **tara**, tipo de veículo (apoio), **rodado/carroceria** (códigos tpRod/tpCar do MDF-e), eixos, capacidade kg/m³, **tipo do proprietário** (tpProp; TAC exige RNTRC do proprietário), motorista com **CPF** (conferido), CNH, **categoria e validade da CNH** |
| Regiões | UF/cidade/faixa de CEP com prazo e tabela própria |
| Ocorrências | data, tipo (apoio), pedido de venda e **romaneio** (pelo código, conferidos na empresa), atraso, custo |

Migração: `000501_transportadora_completa`.

## Cadastros de apoio (na própria VSUP0140)

- **Tipos de veículo** (`shipping_vehicle_types`): código, descrição, rodado,
  carroceria, eixos e capacidade. Escolhido na frota, completa o que o veículo
  deixou em branco.
- **Tipos de ocorrência** (`shipping_occurrence_types`): código, descrição e
  `counts_as_failure` (pesa contra a transportadora na avaliação). A empresa sem
  nenhum tipo recebe os padrões na primeira consulta ou no primeiro cadastro.

O código não muda depois de criado; tipo em uso não se exclui (inativa-se).

## Travas

| Ação | Bloqueada quando |
|---|---|
| Excluir transportadora | está em qualquer documento: cliente (padrão), romaneio, carga, pedido de compra (inclusive redespacho), aviso de recebimento, nota fiscal ou ocorrência |
| Inativar (botão ou formulário) | há documento **em aberto** (romaneio/carga não expedidos, pedido não recebido, aviso não liberado) ou cliente ativo com ela como padrão |
| Trocar o fornecedor | já está em qualquer documento |
| Remover/inativar veículo | a placa está em carga ou romaneio não expedido desta transportadora |
| Reativar | o fornecedor está inativo |
| Inativar o fornecedor (VSUP0500) | ele tem perfil de transportadora ativo |
| Registrar ocorrência | tipo não cadastrado ou inativo; pedido de venda ou romaneio que não existe na empresa |
| Gravar cliente com transportadora padrão (VCLI0500) | a transportadora nova é inexistente na empresa ou inativa |

A ocorrência deixou de ser apagada em cascata com a transportadora: o histórico
de avaliação é preservado.

`GET /api/shipping-carriers/{id}/usage` devolve onde ela está gravada
(`documents[]` com `total` e `open`, `can_delete`, `can_deactivate`).

## Pendências (alertas)

Não impedem gravar; aparecem na tela e na cotação: RNTRC, seguro ou CNH vencidos
**ou a vencer em 30 dias**, rodoviária sem RNTRC, sem seguro, tabela zerada, sem
veículo ativo, veículo sem UF da placa ou tara (o MDF-e não sai), sem região.
"Hoje" é a data no fuso da empresa (`enterprise_notification_settings.timezone`,
padrão America/Sao_Paulo) — vale para vencimentos, previsão de entrega da cotação
e para recusar ocorrência com data futura.

## Cotação de frete

- **Destino:** UF, cidade e/ou CEP. Com CEP, a UF é deduzida pelas faixas dos
  Correios; CEP de outro estado, UF inexistente e peso zero são recusados.
- **Região usada:** a mais específica que atende — faixa de CEP > cidade (sem
  acento/caixa) > UF. Região de cidade não vale para o estado inteiro.
- **Valor:** peso cobrado = maior entre peso real e volume × fator de cubagem;
  sobre ele incidem R$/kg e pedágio (por fração de 100 kg). Total =
  max(piso, peso + ad valorem + GRIS + pedágio) + taxa de despacho. **Não inclui
  ICMS.**
- **Prazo:** em dias úteis (seg–sex, sem feriados), a partir de hoje no fuso da
  empresa.
- **Impedimentos** (RNTRC ou seguro vencido, carga acima da cobertura, nenhum
  veículo comporta): a cotação aparece, vai para o fim e não recebe "mais
  barata"/"mais rápida". O resto (inclusive "sem fator de cubagem" em carga
  volumosa) é aviso.
- `dangerous=true` tira quem não aceita produto perigoso.
- Transportadora sem tabela (total zero) não é cotada: vai para "não atendem"
  com o motivo, em vez de aparecer como frete grátis.

## Rotas

`/api/shipping-carriers`: `GET /`, `POST /`, `GET|PUT /{id}`, `DELETE /{id}`
(ADMIN), `PATCH /{id}/status`, `GET /{id}/usage`, `GET|POST /{id}/occurrences`,
`GET /quote`, `GET /supplier/{supplierCode}`, `GET|POST /vehicle-types`,
`PUT|DELETE /vehicle-types/{typeId}`, `GET|POST /occurrence-types`,
`PUT|DELETE /occurrence-types/{typeId}` (DELETE só ADMIN).

## Onde é consumida

Carga, romaneio, pedido de compra (e redespacho), aviso de recebimento, cliente
(padrão) e NF-e (grupo transporta) gravam `shipping_carrier_id`; todos recusam
transportadora inativa ou de outra empresa. Como a inativação é bloqueada
enquanto houver documento aberto, nenhum documento aberto fica apontando para
transportadora inativa. Os campos de busca carregam também as inativas, marcadas:
saem da escolha, mas o documento antigo continua mostrando o nome. A carga e o
romaneio sugerem a placa a partir da frota da transportadora escolhida e trazem o
motorista. `carrier_code` continua sendo o portador.
