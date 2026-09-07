# Treinamento Prático VentureERP — Tecnofer

Material de capacitação operacional em dois dias, construído sobre as fichas técnicas reais
da Tecnofer (RN 01001, RN 01007 e SU 02046).

## Entregáveis

| Arquivo | Conteúdo | Páginas |
|---|---|---|
| `dia-um.pdf` | Do cadastro do produto à saída do acabado — classificação, itens em 4 níveis, estrutura (BOM) com fórmulas e histórico, máquinas, operações com modelo de tempo completo, roteiros com tempo e custo por lote, configurador de produto, orçamento → pedido de venda, MRP, CRP, APS, ordens, plano de corte, estoque, inspeção (final e de recebimento) e romaneio | 68 |
| `dia-dois.pdf` | Fiscal, financeiro, custos e contabilidade — configuração fiscal, CFOPs, NCM, NF-e de saída e entrada, industrialização em terceiro, contas a pagar/receber, fluxo de caixa, custo padrão, formação de preço, plano de contas, apuração e SPED | 26 |

Cada dia é de **4 horas (08h00–12h00)**, com agenda cronometrada, passo a passo campo a campo,
caixas de destaque, exercícios práticos, checklists e anexos com todos os dados de cadastro prontos.

### Atualização de setembro/2026

O Dia 1 foi revisto para acompanhar as telas de engenharia:

- **Roteiro de fabricação** mudou de código: `VPRO0100` deixou de existir e a tela é a **VENT0202**
  (roteiro por item) / **VENT0115** (roteiros padrão).
- **Modelo de tempo completo** na operação — preparação por lote, tempo de máquina, mão de obra,
  peças por ciclo, operadores, fila, espera e movimentação — e a unidade dos tempos passou a aceitar
  minutos, o que dispensa converter a ficha para horas.
- **Tempo e custo do lote**: novo bloco que simula o roteiro para um tamanho de lote e mostra o custo
  por peça (seção 4.4.8).
- **Terceirização na própria operação** (fornecedor, item de serviço, custo por peça, prazo, remessa).
- **Estrutura de produto**: painel de detalhe com vigência, alternativos, consumo e custo; simulação
  de fórmula; histórico de alterações; botão Conferir.
- **Cadastro de item**: natureza virou marcadores combináveis (um item pode ser base *e* configurado)
  e passou a ser possível **abrir um item para alterar** (seção 3.4.2).
- **Configurador de Produto** (VCFG0100) documentado na seção 4.6, com restrições e auditoria.
- **Roteiro de inspeção de recebimento** (VINS0200) na seção 6.9.1, com plano de amostragem.

## Estrutura de itens adotada

Decisão: **manter os códigos Truckparts existentes** e criar apenas os que faltavam, no mesmo padrão.

```
NÍVEL 0  PA  RN 01001   Balança Asa Delta Carreta Pino Ø 50 mm
 NÍVEL 2 PF  TP 01001A  Lateral 5/16"                × 2
          └─ MP 10794   Chapa de Aço 5/16"      3,710 kg · perda 26,95 %
 NÍVEL 1 CJ  BU 0120/50 Bucha da Balança 120 mm      × 1
          ├─ PF BU 0120E   Bucha externa 120 mm      × 1
          └─ PF BU 0050I   Bucha interna acabada     × 2
                └─ PF BU 0050IB Bucha interna semi-acabada
                       └─ MP 20603 Tubo Ø 60,3 × 4,75
```

Códigos criados neste material (não existiam nas fichas):

- `BU 0050I` — bucha interna 50,4 mm acabada (cementada)
- `BU 0050IB` — bucha interna 50,4 mm semi-acabada (antes da cementação)
- `MP 1xxxx` — chapas de aço, dígitos = espessura em centésimos de mm (`MP 10794` = 7,94 mm = 5/16")
- `MP 2xxxx` — tubos e perfis, dígitos = diâmetro em décimos de mm (`MP 20603` = Ø 60,3 mm)
- `MP 3xxxx` — consumíveis (solda, tinta, abrasivos)

## Achados nas fichas — pendem de decisão da Engenharia

Estão documentados na seção 2.6 do Dia 1. Resumo:

1. **TP 01001C** — dobra descrita como "Dobradeira" mas custeada com a tarifa da Guilhotina.
   Custo correto sobe de R$ 0,95 para R$ 1,26.
2. **BU 0120E aparece duas vezes** — o bloco com "Embuchar na Prensa" é, na leitura deste material,
   o roteiro do conjunto `BU 0120/50`, não da peça. **Confirmar.**
3. **"Endireitar na Morça"** — recurso sem hora-máquina cadastrada. Inferido em ~R$ 22,40/h.
   **Definir tarifa oficial.**
4. **"Montagem"** — custeada a R$ 37,09/h (tarifa da solda) sem centro próprio.
   Proposto criar o centro **Mesa de Montagem**.
5. **Cementação** — o valor 1,74 está na coluna de tempo, mas é preço de serviço
   (0,300 kg × R$ 5,80/kg). Vira preço de terceiro, não tempo de máquina.
6. **RN 01007 / TP 01022A** — quantidade 2 custeada como 1. Custo correto R$ 10,91, não R$ 5,45.
7. **Bloco do BU 0120E** — custos arredondados de uma tabela de hora-máquina antiga.

## Números-chave reconstruídos

| | RN 01001 | SU 02046 |
|---|---:|---:|
| Material | R$ 78,63 | R$ 11,94 |
| Operação | R$ 23,03 | R$ 11,64 |
| Terceiro | R$ 4,88 | — |
| Horas de máquina | 0,4513 h | 0,1856 h |
| Overhead (exemplo, R$ 45/h) | R$ 20,31 | R$ 8,35 |
| **Custo padrão** | **R$ 126,85** | **R$ 31,93** |
| Preço com divisor 0,4975 | R$ 254,97 | R$ 64,18 |

Todas as alíquotas fiscais e o percentual de overhead são **exemplos didáticos** — o enquadramento
real precisa ser confirmado com a contabilidade da Tecnofer antes de qualquer uso em produção.

## Como reconstruir os PDFs

```bash
pip install weasyprint
python3 gerar.py            # gera os dois
python3 gerar.py dia-um     # gera só um
```

## Arquivos

```
dia-um.html / dia-dois.html   fonte editável do conteúdo (HTML puro)
estilo.css                    folha de estilo compartilhada (A4, cabeçalho, rodapé, componentes)
gerar.py                      renderiza os HTML em PDF via WeasyPrint
fichas/                       as três planilhas originais da Tecnofer
```

Para alterar o conteúdo, edite o HTML e rode `gerar.py` de novo. O CSS já traz os componentes prontos:
`.passo`, `.campos`, `.dados`, `.nota`, `.atencao`, `.porque`, `.achado`, `.dica`, `.exercicio`,
`.arvore`, `.fluxo`, `ul.check` e `table.agenda`.
