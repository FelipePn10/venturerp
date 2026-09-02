# Ambiente de treinamento isolado

## Objetivo

Disponibilizar dois modos de acesso ao VentureERP:

- produção, com o comportamento e a política de backup atuais;
- treinamento, para cadastros, execuções, testes e capacitação sem alterar os
  dados operacionais de produção.

## Decisões aprovadas

1. Executar uma segunda instância leve da API, usando a mesma imagem imutável.
2. Usar uma segunda base lógica no mesmo PostgreSQL da VPS, com credenciais e
   permissões próprias.
3. Manter usuários, credenciais, empresas, papéis, vínculos e versões de
   autenticação sincronizados a partir de produção; os demais dados permanecem
   isolados.
4. Não incluir a base de treinamento no backup de produção.
5. Permitir integrações externas somente em homologação/teste. Documentos,
   e-mails e demais saídas devem identificar inequivocamente que são testes e
   não possuem valor fiscal quando aplicável.
6. Falhar de forma segura quando um provedor não oferecer ou não estiver
   configurado em modo de homologação; nunca usar produção como fallback.
7. Preservar isolamento por empresa também dentro do ambiente de treinamento.

## Escopo

- configuração explícita do ambiente de dados no backend e no JWT;
- autenticação de treinamento contra a autoridade de usuários de produção;
- validação de autorização e versão de autenticação sem depender de cópias
  eventualmente desatualizadas;
- bootstrap/deploy da segunda base e da segunda API com limites conservadores;
- migrations aplicadas também à base de treinamento;
- sincronização idempotente dos registros de identidade necessários às chaves
  estrangeiras da base de treinamento;
- separação explícita dos backups;
- proteção e identificação do modo de homologação nas integrações existentes;
- contrato HTTP que permita ao cliente distinguir produção de treinamento;
- testes de autenticação, isolamento, configuração fail-closed e regressão.

## Fora do escopo

- copiar dados operacionais de produção para treinamento;
- incluir a base de treinamento nos backups;
- refatorações não relacionadas;
- alterar migrations já publicadas;
- modificar ou apagar alterações preexistentes no worktree.

## Critérios de aceite

1. O mesmo usuário ativo e autorizado em produção consegue autenticar no modo
   de treinamento sem manter uma segunda senha manualmente.
2. Tokens identificam o ambiente e não são aceitos pela instância oposta.
3. Toda operação autenticada de treinamento usa exclusivamente a base de
   treinamento e mantém o tenant selecionado.
4. Cadastros e execuções de treinamento não aparecem em produção, e o inverso
   também não ocorre.
5. Alterações de senha, papel, vínculo, bloqueio e versão de autenticação em
   produção passam a valer no treinamento pelo mecanismo definido.
6. A base de treinamento não participa do backup nem do restore produtivo.
7. Integrações no treinamento usam homologação/teste, identificam as saídas e
   recusam execução quando essa garantia não puder ser comprovada.
8. Deploy e atualização aplicam migrations nas duas bases, preservando backup,
   readiness e rollback da produção.
9. Testes cobrem tentativa de cruzar ambientes, dois tenants, token no ambiente
   incorreto, sincronização/revalidação e configuração externa insegura.

## Restrições operacionais

- Trabalhar somente no worktree `panossoerp-ajustes`, branch
  `fix/ajustes-operacionais`.
- Não tocar na branch `develop`.
- Não editar código gerado pelo sqlc manualmente.
- Preservar todas as alterações preexistentes no worktree.

## Validação executada

- Duas bases PostgreSQL descartáveis receberam a cadeia completa de migrations
  até `319` e foram removidas após os testes.
- Teste de integração real cobriu espelhamento idempotente, senha inutilizável
  no treinamento, atualização de papel/versão e bloqueio de usuário inativo.
- Duas APIs reais temporárias provaram login nos dois ambientes, rejeição
  cruzada dos JWTs, isolamento de cadastro operacional e ausência das rotas de
  atualização do host no treinamento.
- `go vet ./...`, regressão `go test ./...`, testes focados com `-race`, sintaxe
  shell, Compose com profile `training`, `gofmt -d` e `git diff --check`
  passaram.
