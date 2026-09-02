# Ambiente de treinamento

O treinamento roda com a mesma imagem da produção, em uma segunda instância da
API (`127.0.0.1:5071`) e uma segunda base lógica no mesmo PostgreSQL. A API de
produção continua em `127.0.0.1:5070`.

## Isolamento e identidade

- `DATABASE_URL` da instância de treinamento aponta exclusivamente para
  `venturerp_training`.
- `IDENTITY_DATABASE_URL` aponta para produção e é usada somente por login,
  cadastro/troca de senha e revalidação de autorização.
- No login, usuário, empresa e vínculo selecionado são espelhados de forma
  idempotente na base de treinamento para satisfazer autoria e chaves
  estrangeiras. O hash da senha não é copiado: a linha local recebe um valor
  inutilizável, pois toda autenticação ocorre na autoridade de produção. Nenhum
  dado operacional é copiado.
- O JWT contém `environment`. Um token de treinamento recebe `401` na produção
  e vice-versa, mesmo se as instâncias compartilharem o segredo.
- A resposta do login contém `environment: "production"` ou `"training"`; o
  cliente deve manter essa indicação visível durante toda a sessão.

## Provisionamento

1. Crie a role e a base vazia `venturerp_training` no mesmo cluster PostgreSQL,
   com uma senha própria e sem privilégios sobre outras bases.
2. Copie `deploy/production/training.env.example` para o caminho protegido
   `/opt/venturerp/panossoerp/.env.training`, preencha os segredos e use modo
   `0600`.
3. Configure `TRAINING_DATABASE_URL` e `TRAINING_API_ENV_FILE` em
   `/etc/venturerp/update.env`.
4. Execute o updater oficial. Ele aplica as mesmas migrations nas duas bases e
   ativa o profile Compose `training`.
5. Valide `/health/ready` nas portas 5070 e 5071 e faça login em ambas.

A base de treinamento não entra no `pg_dump`, restore nem retenção de backups.
O updater cria e verifica backup somente de `DATABASE_NAME`, que é produção.
As rotas `/api/system/update` não são registradas na API de treinamento; testes
no painel não conseguem enfileirar uma atualização real do host.

## Integrações

O processo define `DATA_ENVIRONMENT=training`. O cliente Focus NF-e recusa
qualquer configuração `producao` antes de abrir conexão; as configurações
fiscais do treinamento devem usar `homologacao`, cujo documento é emitido sem
valor fiscal. E-mails continuam podendo ser enviados, mas recebem a marca
`[TREINAMENTO - SEM VALOR]` no assunto e aviso no corpo texto/HTML.

Nunca use produção como fallback quando uma integração não oferecer sandbox ou
homologação. Nesse caso, a operação deve falhar e ser configurada antes do uso.

## Recursos

A segunda API está limitada a 0,5 CPU, 256 MiB de memória e tmpfs de 32 MiB.
Não existe um segundo servidor PostgreSQL: apenas outra base no cluster já
existente.
