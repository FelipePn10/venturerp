# Troca de senha com aprovação administrativa

Disponível inicialmente somente no ambiente de desenvolvimento. A promoção para produção exige aprovação explícita.

## Fluxo

1. O usuário autenticado cria uma solicitação, com motivo opcional (até 500 caracteres). Só uma solicitação em andamento (aguardando ou aprovada e válida) por usuário: a segunda responde 409 com mensagem explicativa.
2. Um `ADMIN` da mesma empresa lista e aprova ou rejeita. Quem pede não aprova a própria solicitação (403); a rejeição exige motivo de 3 a 500 caracteres, que o solicitante vê.
3. A aprovação expira em 15 minutos e pode ser consumida uma única vez. Aprovação vencida aparece como `EXPIRED` na leitura e é encerrada ao abrir um pedido novo — antes ficava `APPROVED` para sempre e impedia o usuário de pedir outra troca (a migration `000432` encerrou as que já estavam nessa situação).
4. O usuário conclui informando a senha atual, a nova senha e sua confirmação. A tela descobre a solicitação aprovada por `GET /mine`; o identificador nunca é digitado nem exibido.
5. A conclusão incrementa `users.auth_version`; todos os JWTs emitidos anteriormente são recusados imediatamente.

Todas as operações mutáveis passam pelo middleware de auditoria e todas as queries incluem `enterprise_id`. A resposta não traz `user_id` nem `enterprise_id`; traz nome e e-mail de quem pediu, nome de quem aprovou/rejeitou, datas e `own` (a solicitação é de quem consulta).

## Endpoints

Todos exigem `Authorization: Bearer <token>`.

- `POST /api/password-change-requests/` — cria uma solicitação para o usuário autenticado. Corpo opcional `{"reason":"..."}`.
- `GET /api/password-change-requests/mine` — solicitações do usuário autenticado, a mais recente primeiro.
- `GET /api/password-change-requests/?status=PENDING` — lista solicitações da empresa; somente `ADMIN`.
- `POST /api/password-change-requests/{requestID}/approve` — aprova; somente `ADMIN`.
- `POST /api/password-change-requests/{requestID}/reject` — rejeita; somente `ADMIN`.
- `POST /api/password-change-requests/{requestID}/complete` — conclui a troca pelo titular.

Respostas de recusa: 400 (senha fraca, igual à atual, motivo inválido, identificador fora do formato), 403 (sem permissão, autoaprovação), 404 (solicitação de outra empresa ou de outro usuário), 409 (já existe solicitação em andamento, já decidida, aprovação expirada, ainda não aprovada), 422 (senha atual incorreta — não é 401, para o app não encerrar a sessão).

Corpo da rejeição:

```json
{"reason":"Motivo obrigatório, de 3 a 500 caracteres"}
```

Corpo da conclusão:

```json
{
  "current_password":"senha atual",
  "new_password":"NovaSenha#2026",
  "confirm_password":"NovaSenha#2026"
}
```

## Telas

VSEC0100 (o código antigo VUSR0100 abre a mesma tela): seção **Minha senha** para todo usuário e **Aprovação das solicitações da empresa** para administradores. Na tela de login, **Trocar senha** confirma e-mail e senha atual e mostra o passo em que a troca está.

## Política da nova senha

- entre 12 e 128 caracteres;
- pelo menos uma letra maiúscula e uma minúscula;
- pelo menos um número e um caractere especial;
- diferente da senha atual;
- hash bcrypt com custo 12.

## Banco de dados

A migration `000238_password_change_approval` adiciona `users.auth_version` e a tabela `password_change_requests`. A conclusão usa transação e bloqueio da linha do usuário para impedir corrida entre duas tentativas.

O executor de migrations deve conceder ao papel da API `SELECT, INSERT, UPDATE, DELETE` na nova tabela e `SELECT` ao papel read-only, mantendo o padrão de privilégios do ambiente.
