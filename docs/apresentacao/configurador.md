# VentureERP — Configurador de Produto

### Apresentação para a Engenharia e o Comercial

O Configurador de Produto permite vender e fabricar **itens que variam** (cor, medida,
acabamento, opcionais) sem cadastrar um código para cada combinação. Você descreve o item
por **perguntas** (características) e **respostas** (variáveis); ao configurar, o sistema
monta automaticamente a **máscara** que identifica exatamente aquela variação.

Esta é a **primeira entrega** (fundação). As telas de Restrições, Desenhos, Regras e
descrições especiais chegam nas próximas.

---

## 1. Conjuntos e Variáveis

Um **conjunto** agrupa respostas afins (ex.: conjunto "COR" com AZUL, VERDE, PRETO). Cada
**variável** tem:

- um **código** e uma **descrição** (ex.: `AZ` = "Azul");
- a **composição da máscara** (a sigla/valor que aparece na máscara, ex.: `AZUL`);
- marcações de **Ativo**, **Especial**, **Inclui descrição**, **Marketing** e **dados
  especiais** para exibição;
- **idiomas** (tradução por país).

## 2. Características

A **característica** é a pergunta que aparece ao configurar o produto ("Cor da tampa?").
Cada uma tem um **tipo** que define como é respondida:

- **Escolha / Escolha múltipla** — escolher uma (ou várias) variáveis de um conjunto;
- **Inf. Caractere / Inf. Numérica** — texto livre ou número (com faixa e múltiplo, ex.:
  largura de 1 a 100, de 2 em 2);
- **Opção** — sim/não;
- **Desenho** — leva um código de desenho como resposta;
- **Fórmula** — a resposta é calculada por uma fórmula;
- **Campo / Sequencial** — puxa um dado do pedido ou gera um número.

Ainda por característica: **máscara** (como visualizar), **afeta preço**, **controla
metas**, **especial** (esconde as de baixo até ser respondida) e **variável default**.

## 3. Características do Item

Aqui ligamos as características a um **item**, definindo:

- a **sequência** (de 10 em 10, para dar espaço a novas perguntas);
- a **resposta default** do item (tem prioridade sobre a default da característica);
- a **característica pai** — as "filhas" só aparecem depois que a pai é respondida,
  deixando a tela do configurador mais limpa;
- marcações **Especial**, **Desenho** e **Carga** (para copiar pedidos parecidos).

> Trava de segurança: depois que o item já tem **máscara gerada**, não é permitido mudar a
> sequência nem remover uma característica — evita quebrar itens já usados em pedidos.

## 4. Montando a máscara

Respondidas as perguntas, o sistema junta as respostas na ordem das sequências e forma a
máscara (ex.: `VERDE#50`). Essa máscara é a **mesma** usada pela estrutura de produto,
pelo pedido de venda e pelo MRP — ou seja, o configurador conversa com o resto do ERP.

## 5. Cadastro de Desenhos

Gerencia os **desenhos de engenharia** e suas **revisões** ao longo do tempo (com
vigência, aprovação, motivo e distribuição). O código de replicação é
`Desenho + Dígito + Formato + Revisão`. Um desenho pode ser a resposta de uma
característica do tipo *Desenho* no configurador.

Cada fábrica acessa somente seus desenhos. O código de engenharia pode ser
mantido no item simples ou em cada configuração. Com o parâmetro 8 habilitado,
uma revisão corrente atualiza somente configurações que ainda utilizavam
exatamente a revisão anterior; a primeira revisão é informada manualmente.

## 6. Máscara de Lotes/Séries

Define **como o código de lote é gerado automaticamente**. A máscara (até 20 caracteres)
é montada por partes: texto fixo, data, sequência numérica ou sequência de letras. A
sequência avança a cada geração e pode **zerar na virada do ano**. A máscara certa é
escolhida pelo contexto (cliente, item, classificação, módulo) — ex.: `LT26070001`,
`LT26070002`, …
