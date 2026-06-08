# VentureERP — Cadastros

### Apresentação para a empresa

---

Os cadastros são a **base de tudo** no VentureERP. Cada informação cadastrada uma única vez (uma empresa, um cliente, um fornecedor, um produto, uma condição de pagamento, uma alíquota de imposto) volta a ser usada automaticamente em pedidos, notas fiscais, planejamento, produção e financeiro. **Quanto mais completo o cadastro, mais o sistema trabalha sozinho** — e menos erros aparecem lá na frente.

Este documento cobre **todos** os cadastros do sistema, agrupados por assunto.

---

## Sumário

1. [Empresa](#1-empresa)
2. [Localização (países e estados)](#2-localização-países-e-estados)
3. [Cliente](#3-cliente)
4. [Fornecedor e Transportadora](#4-fornecedor-e-transportadora)
5. [Item / Produto](#5-item--produto)
6. [PDM — descrição técnica padronizada](#6-pdm--descrição-técnica-padronizada)
7. [Classificação de itens (máscaras de classificação)](#7-classificação-de-itens-máscaras-de-classificação)
8. [Estrutura do Produto (BOM — a "receita")](#8-estrutura-do-produto-bom--a-receita)
9. [Configurador de produto (perguntas e restrições)](#9-configurador-de-produto-perguntas-e-restrições)
10. [Conversão de unidade de medida](#10-conversão-de-unidade-de-medida)
11. [Classificação fiscal do item (NCM)](#11-classificação-fiscal-do-item-ncm)
12. [Fornecedor preferencial por item](#12-fornecedor-preferencial-por-item)
13. [Funcionário](#13-funcionário)
14. [Armazém e Localização de estoque](#14-armazém-e-localização-de-estoque)
15. [Ordem recomendada de cadastro](#15-ordem-recomendada-de-cadastro)

---

## 1. Empresa

A **empresa** é o seu próprio negócio dentro do sistema. É cadastrada uma vez e alimenta automaticamente **todas** as notas fiscais, cálculos de imposto e obrigações:

- razão social, nome fantasia, CNPJ e inscrições (estadual, municipal);
- endereço completo;
- **regime tributário** (Lucro Real, Lucro Presumido ou Simples Nacional) — define como os impostos são calculados;
- dados de integração com a SEFAZ (certificado/credenciais para emissão de NF-e).

O sistema suporta **matriz e filiais**: cada estabelecimento tem seu próprio CNPJ e parâmetros, e os fornecedores podem ser vinculados a uma ou mais empresas.

---

## 2. Localização (países e estados)

Cadastros de base geográfica, usados em endereços e nas regras fiscais (a UF de origem e destino define a alíquota de ICMS entre estados):

| Cadastro | Conteúdo |
|---|---|
| **Países** | Nome e sigla; usados em endereços e em operações de exportação |
| **UFs (estados)** | Cada estado vinculado ao seu país; base do cálculo de ICMS interestadual |

Normalmente são cadastrados uma vez (o Brasil e suas 27 UFs) e raramente mudam.

---

## 3. Cliente

O cadastro de cliente guarda **quem compra** da empresa e **as condições comerciais** que valem para ele. O cadastro principal é acompanhado de vários **cadastros de apoio**, que são preenchidos uma vez e reaproveitados em todos os clientes.

### Dados do cliente

| Bloco | Para que serve |
|---|---|
| Dados básicos | Razão social/nome, CNPJ ou CPF, inscrição estadual |
| **Endereços** | Cobrança e entrega — usados na nota fiscal e no frete |
| **Contatos** | Pessoas, telefones e e-mails, organizados por tipo |
| **Estabelecimentos** | Filiais do cliente ligadas à matriz |
| Classificação | Região, segmento de mercado e tipo de cliente (para análises) |

### Cadastros de apoio do cliente

| Apoio | O que define |
|---|---|
| **Regiões** | Agrupamento geográfico/comercial dos clientes |
| **Segmentos de mercado** | Ramo de atividade (para relatórios e curva ABC) |
| **Tipos de contato** | Financeiro, compras, técnico… |
| **Tipos de cliente** | Categoria comercial |
| **Portadores e grupos de portadores** | Bancos/agentes de cobrança (financeiro) |
| **Condições de pagamento** | Prazos e **parcelas** (à vista, 30/60/90…) que alimentam o contas a receber |
| **Tabelas de venda e preços** | Lista de preços por item aplicada aos pedidos do cliente |
| **Tipos de nota fiscal de saída** | Como a NF-e do cliente deve ser emitida |
| **Tipos de imposto** | Tratamento tributário aplicável |

### Crédito e bloqueio

O cliente pode ser **bloqueado** e **desbloqueado** (manualmente ou por limite de crédito). Enquanto bloqueado, **não é possível avançar pedidos** desse cliente — uma proteção contra vender para quem está inadimplente ou acima do limite.

---

## 4. Fornecedor e Transportadora

O fornecedor é **de quem a empresa compra**; a transportadora é um tipo de fornecedor (de frete). O cadastro guarda dados fiscais, contatos e — o mais importante para a automação — os **padrões de compra**.

### Dados do fornecedor

- razão social, CNPJ, inscrição estadual, endereço, **telefones, e-mails e contatos**;
- **vencimentos** (regras para calcular as datas das contas a pagar);
- vínculo com uma ou mais **empresas** (matriz/filiais);
- **padrões de compra** (condição de pagamento, tabela de preço, tipo de nota e frete) — usados para preencher pedidos automaticamente.

### Cadastros de apoio do fornecedor

| Apoio | O que define |
|---|---|
| **Tipos de fornecedor** | Categoria (matéria-prima, serviço, transportadora…) |
| **Tipos de contato** | Financeiro, comercial, etc. |
| **Parâmetros de fornecedores** | Regras padrão por empresa (1 conjunto por empresa) |

### Recursos importantes

- **Consulta à SEFAZ:** o sistema consulta a situação cadastral do CNPJ direto na Receita/SEFAZ, evitando cadastrar fornecedor irregular.
- **Padrões de compra (purchasing defaults):** quando o planejamento sugere uma compra, ela já vem com o fornecedor e suas condições preenchidas.
- **Bloqueio/desbloqueio** e **exclusão** (esta última restrita ao administrador).

---

## 5. Item / Produto

O item é o **coração do cadastro**: tudo que a empresa compra, fabrica ou vende é um item. Cada item tem uma **natureza** que determina como o sistema o trata:

| Natureza | Exemplo | Como o sistema trata |
|---|---|---|
| **Comprado / matéria-prima** | Chapa de aço | Entra por compra; o MRP gera ordens de **compra** |
| **Fabricado / intermediário** | Suporte cortado | É produzido e consumido em outra etapa |
| **Fabricado / produto final** | Suporte Soldado SS-100 | É o que o cliente compra; o MRP gera ordens de **produção** |
| **De terceiro** | Item em poder de terceiros/beneficiamento | Tratamento especial |

O item também tem **situação** (em linha, promoção…) e um **status de fabricação/compra**, que ajudam a organizar o portfólio.

### O que define um item bem cadastrado

- **Descrição técnica padronizada (PDM)** — ver seção 6.
- **Unidade de medida** e suas **conversões** (ver seção 10).
- **Nível de planejamento (LLC)** — organiza a ordem em que o MRP "abre" o produto, do final até a matéria-prima. É a espinha dorsal do cálculo.
- **Classificação fiscal (NCM)** — ver seção 11.
- **Tempos de máquina e roteiro** — ver `maquinas.md` e `producao.md`.
- **Estrutura (BOM)** — ver seção 8.

### Variante (máscara)

Um mesmo item pode ter **variações** (dimensões, acabamentos). O sistema chama isso de **máscara/variante**, e pode haver tempos de produção e preços diferentes por variante. O sistema gera a máscara do item automaticamente a partir das características.

### Prontidão para ativação

O sistema verifica se um item está **pronto para ser usado no MRP** (validação de prontidão), apontando o que ainda falta — por exemplo, classificação fiscal, conversão de unidade, estrutura ou roteiro ausentes. Isso evita "rodar o planejamento" com cadastro incompleto.

---

## 6. PDM — descrição técnica padronizada

O **PDM** (Product Data Management) monta a **descrição do item por características**, em vez de texto livre. Isso elimina o "item gêmeo" (o mesmo produto cadastrado duas vezes com descrições diferentes). Dois cadastros sustentam o PDM:

| Cadastro | O que é |
|---|---|
| **Grupos** | Famílias de itens (ex.: "Chapas", "Parafusos") com suas características |
| **Modificadores** | As características que compõem a descrição (material, dimensão, acabamento…) |

A partir do grupo e dos modificadores escolhidos, o sistema **gera a descrição e a máscara** do item de forma consistente.

---

## 7. Classificação de itens (máscaras de classificação)

Além do PDM, o sistema permite **classificar os itens em árvore** por meio de **máscaras de classificação** e suas **classificações** (categorias e subcategorias). Serve para organizar o catálogo, gerar relatórios e aplicar regras por categoria. Cada classificação pode ter **filhos** (subníveis), formando uma hierarquia.

---

## 8. Estrutura do Produto (BOM — a "receita")

A **estrutura** (ou BOM — lista de materiais) diz **do que cada produto é feito**: quais componentes, em que quantidade e com qual percentual de perda esperado.

**Exemplo simplificado:**
```
Suporte Soldado SS-100  (1 unidade)
 ├── Suporte Cortado     2 peças
 │     └── Chapa de Aço   0,8 kg   (+5% de perda)
 └── Parafuso M8          4 unidades
```

É a partir da estrutura que o MRP descobre que, para fabricar 100 suportes, precisa de 200 peças cortadas, 160 kg de chapa (já com a perda) e 400 parafusos.

O sistema oferece ferramentas para trabalhar a estrutura:

| Recurso | Para que serve |
|---|---|
| **Criar / atualizar estrutura** | Montar e manter a receita de cada produto |
| **Resolver estrutura** | "Explodir" um produto e ver todos os componentes, nível a nível |
| **Consultar estrutura** | Visualizar a árvore de um item |
| **Onde é usado (where-used)** | Mostrar em **quais produtos** um componente aparece — essencial para avaliar o impacto de trocar uma matéria-prima |

---

## 9. Configurador de produto (perguntas e restrições)

Para produtos que variam conforme escolhas do cliente, o sistema tem um **configurador**:

| Recurso | O que faz |
|---|---|
| **Perguntas e opções** | Define perguntas (ex.: "Cor?", "Espessura?") e suas respostas possíveis |
| **Associação de perguntas a itens** | Liga as perguntas aos itens que as utilizam |
| **Restrições e motivos de restrição** | Regras que **bloqueiam combinações inválidas** (ex.: "espessura X não pode ter acabamento Y") com um motivo cadastrado |

Na prática, o configurador garante que só sejam montados produtos **tecnicamente possíveis**, e as restrições são avaliadas automaticamente durante a configuração.

---

## 10. Conversão de unidade de medida

Um item pode ser **comprado em uma unidade e consumido em outra** — por exemplo, a chapa é comprada em **quilos** mas consumida por **peça**. O cadastro de **conversão de UM por item** ensina o sistema a converter sozinho (ex.: 1 kg = 1/179,5 chapa), mantendo estoque, custo e planejamento coerentes sem conta manual.

---

## 11. Classificação fiscal do item (NCM)

Cada item recebe sua **classificação fiscal** (NCM e atributos relacionados), que define a tributação nas notas. O cadastro suporta ainda **idiomas** (descrições em outros idiomas) e **atributos de exportação**, úteis para operações internacionais. O **%IPI** definido aqui é puxado automaticamente para o item.

---

## 12. Fornecedor preferencial por item

Cada matéria-prima pode apontar seu **fornecedor preferido** (e fornecedores alternativos). Quando o planejamento sugere uma compra, ela já sai com esse fornecedor e suas condições — acelerando a aprovação e padronizando as compras.

---

## 13. Funcionário

O cadastro de **funcionários** identifica as pessoas que operam o sistema e a fábrica: responsáveis por apontamentos de produção, vendedores, aprovadores. Permite criar, listar, atualizar e **desativar** funcionários (sem apagar o histórico).

---

## 14. Armazém e Localização de estoque

O estoque físico é estruturado em dois níveis:

| Cadastro | O que é |
|---|---|
| **Armazém / Depósito** | O local de estoque, com seu **tipo** (ex.: linha de produção, normal) |
| **Localização** | A posição dentro do armazém, com tipo (interno, externo, inspeção, rejeição, reserva, trânsito, especial…) |

Essa estrutura permite saber não só **quanto** existe, mas **onde** está cada item.

---

## 15. Ordem recomendada de cadastro

Para começar do zero sem retrabalho, esta é a sequência ideal:

```
1.  Empresa (dados e regime tributário)
2.  Localização (países e UFs)
3.  Cadastros de apoio do cliente e do fornecedor
    (condições de pagamento, tabelas, tipos de nota, tipos de imposto)
4.  Armazéns e localizações
5.  Fornecedores  +  Clientes
6.  PDM: grupos e modificadores
7.  Itens (matéria-prima → intermediários → produto final)
8.  Classificação fiscal (NCM) e conversões de unidade dos itens
9.  Fornecedor preferencial dos itens comprados
10. Estrutura (BOM) de cada produto
11. Máquinas e tempos  +  Roteiro  (ver maquinas.md e producao.md)
12. (Opcional) Configurador: perguntas, opções e restrições
13. Verificar a "prontidão para ativação" de cada item
```

Com esses cadastros prontos, o sistema já consegue receber um pedido de venda e planejar tudo automaticamente.

> A versão técnica (endpoints, campos e regras internas) está em
> `../dev/cadastros-cliente.md`, `../dev/cadastros-fornecedor.md` e `../dev/cadastros-item.md`.
