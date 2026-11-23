-- Saneamento da cópia de produção no banco de treino (modo simulação, melhoria 2).
-- Roda DEPOIS do restore, no banco de treino. A instância de treino já recusa a
-- SEFAZ de produção (DATA_ENVIRONMENT=training) e marca os e-mails; isto é a
-- segunda trava, no dado: nada desta cópia fala com o mundo como se fosse real.
BEGIN;
-- A DANFE e o XML das notas copiadas ficam gravados como caminho relativo da
-- emissora (/arquivos/...), completado pelo ambiente ATUAL da empresa. Antes de
-- trocar para homologação, o caminho vira endereço completo do ambiente em que a
-- nota foi emitida — senão a nota de produção seria procurada na homologação.
UPDATE fiscal_exits x
   SET danfe_path = CASE WHEN x.danfe_path LIKE '/%' THEN b.base || x.danfe_path ELSE x.danfe_path END,
       xml_path   = CASE WHEN x.xml_path   LIKE '/%' THEN b.base || x.xml_path   ELSE x.xml_path   END
  FROM (SELECT enterprise_id,
               CASE WHEN focus_nfe_ambiente = 'producao' THEN 'https://api.focusnfe.com.br'
                    ELSE 'https://homologacao.focusnfe.com.br' END AS base
          FROM fiscal_configs) b
 WHERE b.enterprise_id = x.enterprise_id
   AND (x.danfe_path LIKE '/%' OR x.xml_path LIKE '/%');
-- NF-e só em homologação; a caixa de entrada DF-e não sincroniza sozinha. O
-- token de PRODUÇÃO da emissora não fica na cópia (o agente grava o de
-- homologação, se configurado em TRAINING_FOCUS_TOKEN).
UPDATE fiscal_configs SET focus_nfe_ambiente = 'homologacao', dfe_sync_automatico = FALSE, focus_nfe_token = NULL;
-- Senhas: o treino autentica na produção (IDENTITY_DATABASE_URL); o hash não fica
-- na cópia (mesmo valor do espelhamento do login).
UPDATE users SET password = '!managed-by-production-identity-authority';
-- Nada que estava na fila de envio sai pela cópia.
UPDATE notification_outbox SET state = 'CANCELADO' WHERE state IN ('PENDENTE', 'PROCESSANDO', 'FALHOU');
-- O nome da empresa diz que é treino em toda tela e documento.
UPDATE enterprise SET name = '[TREINO] ' || name WHERE name NOT LIKE '[TREINO]%';
COMMIT;
