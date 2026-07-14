# Troca de senha com aprovação administrativa

Disponível inicialmente somente no ambiente de desenvolvimento. A promoção para produção exige aprovação explícita.

## Fluxo

1. O usuário autenticado cria uma solicitação.
2. Um `ADMIN` da mesma empresa lista e aprova ou rejeita a solicitação. O próprio usuário não pode aprovar sua solicitação.
3. A aprovação expira em 15 minutos e pode ser consumida uma única vez.
4. O usuário conclui informando a senha atual, a nova senha e sua confirmação.
5. A conclusão incrementa `users.auth_version`; todos os JWTs emitidos anteriormente são recusados imediatamente.

Todas as operações mutáveis passam pelo middleware de auditoria e todas as queries incluem `enterprise_id`.

## Endpoints

Todos exigem `Authorization: Bearer <token>`.

- `POST /api/password-change-requests/` — cria uma solicitação para o usuário autenticado.
- `GET /api/password-change-requests/?status=PENDING` — lista solicitações da empresa; somente `ADMIN`.
- `POST /api/password-change-requests/{requestID}/approve` — aprova; somente `ADMIN`.
- `POST /api/password-change-requests/{requestID}/reject` — rejeita; somente `ADMIN`.
- `POST /api/password-change-requests/{requestID}/complete` — conclui a troca pelo titular.

Corpo da rejeição:

```json
{"reason":"Motivo opcional, limitado a 500 caracteres"}
```

Corpo da conclusão:

```json
{
  "current_password":"senha atual",
  "new_password":"NovaSenha#2026",
  "confirm_password":"NovaSenha#2026"
}
```

## Política da nova senha

- entre 12 e 128 caracteres;
- pelo menos uma letra maiúscula e uma minúscula;
- pelo menos um número e um caractere especial;
- diferente da senha atual;
- hash bcrypt com custo 12.

## Banco de dados

A migration `000238_password_change_approval` adiciona `users.auth_version` e a tabela `password_change_requests`. A conclusão usa transação e bloqueio da linha do usuário para impedir corrida entre duas tentativas.

O executor de migrations deve conceder ao papel da API `SELECT, INSERT, UPDATE, DELETE` na nova tabela e `SELECT` ao papel read-only, mantendo o padrão de privilégios do ambiente.
