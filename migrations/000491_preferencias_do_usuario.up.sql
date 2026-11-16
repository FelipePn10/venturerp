-- Preferências de exibição de cada usuário (valem em qualquer computador e
-- em todas as empresas dele). A primeira: nas buscas e campos de cliente,
-- mostrar a razão social ou o nome fantasia.
-- Sem FK para users: no ambiente de treinamento os usuários ficam no banco de
-- identidade, separado deste.
CREATE TABLE user_preferences (
    user_id               UUID PRIMARY KEY,
    customer_name_display VARCHAR(20) NOT NULL DEFAULT 'RAZAO_SOCIAL',
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_user_preferences_customer_name
        CHECK (customer_name_display IN ('RAZAO_SOCIAL', 'FANTASIA'))
);
