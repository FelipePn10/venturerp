# VentureERP — Romaneio / Expedição

> Documento de apresentação

---

## O que o sistema faz nessa área?

O **Romaneio** é o documento que organiza e acompanha a **saída da mercadoria** —
da hora em que o pedido é separado no estoque até o caminhão deixar a fábrica.
Ele responde, de forma rastreável, três perguntas que toda expedição precisa
responder: **o que sai, em quantos volumes, e com quem vai**.

É o documento operacional de expedição do VentureERP: simples de operar e já
integrado ao estoque, às cargas e às notas fiscais.

> Importante: o romaneio **não substitui a nota fiscal**. A NF-e é o documento
> legal (e é ela que dá baixa no estoque). O romaneio é o documento **logístico**
> que controla a separação, a conferência, a embalagem e o transporte —
> complementando a nota.

---

## Por que isso importa

- **Acaba o "sumiço" na expedição:** cada saída tem número, status e histórico de
  quem fez o quê e quando.
- **Conferência confiável:** o que foi separado é conferido contra o que foi
  pedido — e qualquer **divergência (sobra ou falta)** aparece antes de o caminhão
  sair.
- **Não vende o que não tem:** ao separar, o sistema **reserva** o material; o
  estoque disponível para novos pedidos já desconta a carga em preparação.
- **Documento profissional na mão do motorista:** um romaneio impresso com a sua
  marca, dados da carga, volumes e transporte — no padrão de uma indústria séria.

---

## Como funciona, na prática

O romaneio segue um **fluxo com etapas claras** (uma "esteira"), e o sistema só
deixa avançar quando a etapa anterior está em ordem. A fase de expedição também
tem o conceito de **carga**, que agrupa um ou mais romaneios no mesmo caminhão,
rota, box de expedição e previsão de entrega:

1. **Abrir** — o romaneio nasce a partir de um **pedido de venda** (ou de compra,
   ou de uma ordem de produção). Você não digita item por item: o sistema
   **puxa os itens do pedido** automaticamente.
2. **Separar** — quando o estoque vai buscar o material, o romaneio entra em
   *separação* e o sistema **reserva o estoque** dos itens.
3. **Conferir** — item a item, confere-se a quantidade real contra a planejada. Se
   bateu, segue; se houve **diferença**, ela fica registrada.
4. **Embalar (volumes)** — informa-se como a carga foi embalada: caixas, pallets,
   fardos… cada **volume** com seu peso, dimensões e identificação.
5. **Planejar a carga** — agrupa-se os romaneios em uma **carga**, define-se
   transportadora, motorista, veículo, rota, box/doca e orientações de entrega.
6. **Carregar e liberar** — a carga passa por liberação, início e fim de
   carregamento, com monitor de expedição e separação.
7. **Amarrar a nota e despachar** — liga-se a **NF-e** da carga ao romaneio/carga
   e confirma-se o despacho. A reserva é baixada e o romaneio é impresso.

> O sistema **impede atalhos perigosos**: não dá para despachar sem conferir, nem
> cancelar uma carga já despachada, nem mexer nos itens depois de conferida.

---

## Conferência e divergências

Na conferência, o operador registra **quanto realmente separou** de cada item.

- Bateu com o pedido → item conferido, segue o fluxo.
- Saiu **a mais ou a menos** → o sistema marca a **divergência** e **trava o
  despacho**.
- Para despachar mesmo assim (ex.: faltou material e o cliente aceitou parcial), é
  preciso uma **liberação consciente** — ninguém despacha divergência "sem querer".

Isso é o que diferencia um controle de expedição de verdade de uma simples lista
impressa.

---

## Volumes (a embalagem da carga)

Cada carga é detalhada por **volume** — exatamente o que uma transportadora
espera receber:

- **Espécie da embalagem:** caixa, pallet, fardo, engradado, bobina, tambor…
- **Peso líquido e peso bruto** (separados — o sistema não confunde o peso do
  produto com o peso com embalagem).
- **Dimensões (altura × largura × comprimento)** e **cubagem (m³)** calculada
  automaticamente — a informação que define o frete.
- **Marca / identificação** do volume.

O romaneio impresso traz a **tabela de volumes** completa, e os totais de **peso
líquido, peso bruto e cubagem** somados.

---

## Transporte (os dados da viagem)

No mesmo romaneio ficam registrados os dados de quem leva a carga:

- **Transportadora**, **modalidade de frete** (CIF / FOB / por terceiros),
  **valor do frete** e **seguro**.
- **Placa do veículo**, **motorista** e código **ANTT**.
- **Lacres** da carga (segurança/auditoria).
- **Previsão de entrega**.

---

## Cargas, boxes e painel de expedição

A carga é o **cockpit do caminhão**. Ela mostra tudo que vai sair junto e em que
etapa está:

- Romaneios incluídos na carga.
- NF-es/documentos fiscais vinculados.
- Box ou doca onde a carga está sendo preparada.
- Rota, origem, destino, motorista e veículo.
- Totais de volumes, peso líquido, peso bruto e cubagem.
- Status: planejada, liberada, em carregamento, carregada, despachada ou
  cancelada.

O monitor de expedição mostra as cargas por status e o monitor de separação
mostra cada romaneio com itens conferidos, divergências e vínculo com a carga.
O painel logístico consolida quantas cargas estão planejadas, liberadas, em
carregamento, carregadas, despachadas, quantos boxes estão ocupados e o volume
total previsto para sair.

---

## Ligação com a nota fiscal e o estoque

O romaneio conversa com o resto do ERP — não é uma ilha:

- **Estoque:** ao **separar**, a carga é **reservada** (o disponível já cai); ao
  **despachar**, a reserva é encerrada; se a carga é **cancelada**, a reserva é
  **devolvida**. A **baixa física** definitiva acontece na **emissão da NF-e** —
  sem risco de baixar o estoque duas vezes.
- **Nota fiscal:** o romaneio guarda o **número e a chave da NF-e** da carga, e o
  documento impresso mostra essa referência. Físico e fiscal andam juntos.

---

## O romaneio impresso

Um clique gera o romaneio em **PDF profissional** (e também em **Excel**), com:

- **Cabeçalho com o logo e a cor da sua empresa**, CNPJ, IE e endereço.
- **Dados da carga** (número, data, pedido de origem, NF-e e lacres).
- **Destinatário** e **transportadora** (com placa, motorista, ANTT, frete).
- **Tabela de itens** e **tabela de volumes**.
- **Totais** (peso líquido, peso bruto, cubagem, frete, seguro, previsão).
- **Campos de assinatura** (Emitente, Transportadora, Destinatário) e numeração de
  página.

> É o mesmo motor de documentos usado nos relatórios do sistema — visual de
> indústria de alto nível, pronto para entregar ao cliente e ao motorista.

---

## Exemplo simples

> Um pedido de venda de 8 itens é confirmado. Em vez de redigitar tudo, a
> expedição **gera o romaneio a partir do pedido**. O estoque separa o material
> (o sistema **reserva** automaticamente), confere item a item — e percebe que de
> um item separou **8 em vez de 10**. O sistema **avisa a divergência** e **não
> deixa despachar**. O responsável fala com o vendedor, o cliente aceita os 8, a
> diferença é **liberada conscientemente**, a carga é embalada em **3 volumes**
> (1 pallet + 2 caixas), o motorista e a placa são informados, a **NF-e é
> amarrada** — e o romaneio sai **impresso, com a marca da empresa**, para seguir
> viagem.

---

## Em resumo

| O que toda expedição precisa | Como o VentureERP entrega |
|------------------------------|---------------------------|
| Documento de saída rastreável | Romaneio com número, status e **histórico** de cada etapa |
| Separação | Etapa de separação com **reserva de estoque** |
| Conferência | Conferência por item com **detecção de divergência** |
| Embalagem | **Volumes** com espécie, peso líq/bruto, dimensão e cubagem |
| Transporte | Frete, **placa/motorista/ANTT, lacres** e previsão |
| Planejamento de carga | Agrupamento por caminhão/rota, **box de expedição** e monitor |
| Integração fiscal | **Vínculo com a NF-e**; baixa de estoque na nota |
| Documento para o cliente | **PDF/Excel profissional** com a marca da empresa |

É o controle de expedição que os grandes ERPs têm — no seu chão-de-fábrica, sem a
complexidade deles.
