-- Convivência dos dois modelos de tributação na NF-e de saída, por empresa.
--
-- AUTOMATICO (padrão, sem linha = AUTOMATICO): o modelo atual (ICMS, IPI,
-- PIS/COFINS) mais o grupo IBS/CBS do regime normal, como determina a NT
-- 2025.002. SOMENTE_ATUAL: a nota sai só com o modelo atual, sem o grupo
-- IBS/CBS — válvula para a empresa continuar emitindo se a SEFAZ ou a Focus
-- recusarem algo do grupo novo, sem precisar de uma versão nova do sistema.
CREATE TABLE ibscbs_parametros (
    enterprise_id BIGINT      PRIMARY KEY REFERENCES enterprise(id) ON DELETE CASCADE,
    modo          VARCHAR(20) NOT NULL CHECK (modo IN ('AUTOMATICO', 'SOMENTE_ATUAL')),
    motivo        TEXT,
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by    UUID        NOT NULL REFERENCES users(id)
);
