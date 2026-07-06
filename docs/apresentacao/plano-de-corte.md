# VentureERP — Plano de Corte

> Documento de apresentação

---

## O que o sistema faz nessa área?

O **Plano de Corte** decide a melhor forma de cortar a matéria-prima para
**desperdiçar o mínimo possível**. Você informa as peças que precisa (comprimento e
quantidade) e as barras/perfis/tubos disponíveis no estoque, e o sistema calcula
**como cortar cada barra** — quantas barras usar, onde fazer cada corte e quanto sobra.

É uma das ferramentas que mais economiza dinheiro numa metalúrgica: cada centímetro
de aço aproveitado é custo que não vira sucata.

---

## Como funciona, na prática

1. **Você cria um plano** para um material (ex.: cantoneira 2 polegadas).
2. **Lista as peças que precisa cortar** — ex.: 8 pernas de 720 mm e 4 travessas de 1200 mm.
3. **Lista as barras disponíveis** — ex.: 5 barras de 6000 mm e 1 ponta de 2300 mm.
   > Aqui não existe "barra padrão": cada barra comprada pode ter um tamanho diferente,
   > e o sistema trabalha com os tamanhos reais que você tem.
4. **Manda otimizar.** O sistema devolve o **plano de corte**: para cada barra, em que
   posição cortar cada peça, na ordem certa.

---

## O que o sistema leva em conta

- **Espessura da serra (kerf):** todo corte "come" alguns milímetros. O sistema desconta
  isso para o plano bater com a realidade.
- **Refile (trim):** apara a ponta da barra antes de começar, quando necessário.
- **Sobra mínima reaproveitável:** você define a partir de quanto uma ponta vale a pena
  guardar. Acima desse tamanho ela é tratada como **retalho aproveitável** (e numa
  próxima fase volta para o estoque); abaixo, é sucata.
- **Retalho primeiro:** se houver pontas/retalhos no estoque, o sistema os usa **antes**
  de abrir uma barra nova.

---

## O que você recebe

- **Padrões de corte:** quando várias barras são cortadas do mesmo jeito, elas são
  agrupadas ("corte 3 barras assim"), deixando a instrução curta para o chão-de-fábrica.
- **Indicadores:** percentual de **aproveitamento**, percentual de **refugo**, número de
  barras usadas e número de cortes.
- **Avisos:** se alguma peça for maior do que qualquer barra disponível, ela aparece como
  **"não encaixada"** para você providenciar material.

---

## Exemplo simples

> Preciso de **3 peças de 2000 mm**. Tenho **uma barra de 6000 mm**.
> O sistema corta as 3 peças nessa única barra, com aproveitamento de 100%.

> Agora preciso das mesmas 3 peças, mas a serra tem 5 mm de espessura.
> 2000 + 2000 + 2000 + os cortes passa de 6000 mm — então o sistema avisa que a terceira
> peça vai precisar de uma segunda barra.

---

## Firmar o plano (baixa de estoque + retalhos)

Quando o plano está pronto, você o **firma** — e aí o sistema dá baixa de verdade no
material e organiza as sobras:

- **Baixa real do estoque:** o material cortado sai do estoque, com custo, ligado à
  **ordem de produção** quando houver. Tudo aparece nos relatórios de estoque.
- **De qual lote dar baixa — você escolhe:**
  - **Automático:** o sistema usa os lotes mais antigos primeiro (FIFO).
  - **Manual:** o operador escolhe o lote/corrida de cada peça.
  - **A empresa decide o padrão** numa configuração única; cada plano pode mudar se
    precisar. (Você tem os dois e ainda um padrão da casa.)
- **Sobras viram retalho no estoque:** cada sobra aproveitável volta como um retalho
  guardado, **já com a corrida e o certificado** do material de origem — a
  rastreabilidade não se perde no recorte. No próximo plano, esses retalhos são
  **usados primeiro**, antes de abrir barra nova.
- **Cada material na sua unidade:** o sistema entende como o material é estocado —
  por **peça, metro, m², m³, quilo ou tonelada** — e converte o corte para a unidade
  certa na hora da baixa. Para peso/área/volume, basta informar um **fator** (ex.:
  quilos por metro da barra) e o sistema calcula a baixa e o custo corretamente. A
  unidade é puxada automaticamente do cadastro do item.

## Corte de chapas (2D)

Além de barras, o sistema corta **chapas e painéis** (MDF, aglomerado, chapa de aço) —
o coração da linha moveleira e de funilaria. Você informa as **peças retangulares**
(largura × altura × quantidade) e as **chapas disponíveis**, e o sistema monta o
**mapa de corte da chapa**: onde fica cada peça, com cortes retos de ponta a ponta
(compatíveis com a seccionadora).

Ele respeita o que importa no chão-de-fábrica:

- **Veio (grão):** peças com veio visível não são giradas, para o desenho da madeira
  ficar alinhado.
- **Rotação:** quando a peça pode girar 90°, o sistema aproveita isso para encaixar mais.
- **Espessura da serra e refile** da borda da chapa, como no corte de barras.
- **Sobras viram retalho de chapa:** o maior pedaço que sobra (largura × altura) volta
  ao estoque, com rastreabilidade, para o próximo corte.

Na hora de firmar, a baixa sai na unidade certa da chapa — **por peça, por m² ou por
quilo** — automaticamente.

## Corte de formas livres (laser / plasma)

O sistema também corta **peças de formato qualquer** — não só retângulos. Você informa
o **contorno da peça** (o desenho), e ele organiza as peças na chapa aproveitando o
espaço, inclusive **girando** quando possível.

- **Já funciona "de fábrica":** mesmo sem nenhum software extra, o sistema encaixa as
  peças pela sua área e entrega um plano utilizável.
- **Pode ficar ainda melhor:** dá para conectar um **motor de nesting especializado**
  (externo) que encaixa formatos irregulares com altíssimo aproveitamento — é só ligar,
  sem mudar nada no seu dia a dia.

## Motor de otimização de máxima economia

O cálculo do plano usa técnicas avançadas de otimização industrial, embutidas no
próprio VentureERP:

- **Barras (1D) e chapas (2D):** em vez de só "encaixar como der", o sistema **testa
  combinações de padrões de corte** e escolhe a que gasta menos material — o método de
  *geração de colunas*, padrão da indústria. Na prática, o que antes usava **2 chapas
  pode passar a usar 1**, e barras com estoques de tamanhos variados ganham vários pontos
  de aproveitamento (ex.: de **83% para 91%** num caso real).
- **Formas livres (laser/plasma):** o sistema **procura a melhor ordem de encaixe** das
  peças (busca inteligente), em vez de uma tentativa só — peças em "L" que desperdiçavam
  uma chapa passam a **caber em menos chapas** (ex.: de **4 para 3 chapas, 65% → 87%**).
  E agora **gira as peças em qualquer ângulo** (não só de 90 em 90 graus): uma peça
  comprida que não cabe "de pé" na chapa pode entrar **na diagonal**, e contornos
  irregulares se encaixam melhor uns nos outros.
- **Nunca piora:** o motor avançado sempre compara com o método simples e **fica com o
  melhor dos dois** — então a otimização só tem a ganhar, nunca a perder.
- **Resultado sempre igual para a mesma entrada:** o mesmo pedido gera sempre o mesmo
  plano, o que torna os planos **auditáveis**.
- **Rápido:** segundos por plano. Cada ponto de aproveitamento a mais é aço/MDF que deixa
  de virar sucata.

## O plano nasce sozinho das ordens

Você não precisa digitar as peças uma a uma. A partir das **ordens de produção** (e das
ordens sugeridas pelo MRP), o sistema **monta o plano de corte automaticamente**:

- Lê a **estrutura do produto** (o que ele leva) e identifica as peças que são
  **cortadas de matéria-prima** — com o tamanho de cada uma.
- Descobre **de qual material** cada peça é cortada e **junta várias ordens do mesmo
  material num único plano** — assim o aproveitamento da chapa/barra é o melhor possível.
- Cada peça fica marcada com a **ordem de origem**, e você só precisa informar o estoque
  disponível e mandar otimizar/firmar.

Se alguma peça não puder ser mapeada (faltou cadastrar o material, por exemplo), o
sistema **avisa** em vez de adivinhar.

## Recursos de chão-de-fábrica

- **Mapa de corte para baixar e imprimir:** o desenho de cada chapa/barra, em **SVG,
  DXF (para a máquina/CAM) e PDF**, mostrando onde fica cada peça — inclusive o
  **formato real** das peças irregulares (laser/plasma) e como elas ficam **giradas**,
  não apenas retângulos.
- **Programa de corte e agenda:** a sequência de cortes e o **agendamento na máquina**,
  já no calendário de produção. Para chapas, o sistema entrega a **sequência de cortes
  retos da seccionadora** (qual corte fazer primeiro, depois os sub-cortes) — peças que
  dividem a mesma linha de corte saem em **um corte só**, economizando serra.
- **Encaixe inteligente de formas:** peças de formato irregular se **intertravam** umas
  nas outras, aproveitando o espaço melhor do que só pelo retângulo — sem depender de
  software externo.
- **Fita de borda:** marque os lados que levam fita e o sistema soma o **comprimento e o
  custo** de fita do plano.
- **Custo por ordem:** quando um plano junta várias ordens, o custo do material é
  **rateado entre elas** automaticamente — e o **custo da fita de borda** entra junto, na
  ordem da peça que leva fita.
