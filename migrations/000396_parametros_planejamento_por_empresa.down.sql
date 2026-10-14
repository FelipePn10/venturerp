DROP TRIGGER IF EXISTS trg_enterprise_planning_params ON enterprise;
DROP FUNCTION IF EXISTS semear_parametros_planejamento_nova_empresa();
DROP FUNCTION IF EXISTS semear_parametros_planejamento(BIGINT);

-- Só sai o que a 000396 criou e continua como foi semeado.
DELETE FROM planning_params p
 USING planning_params_semeados s
 WHERE p.id = s.planning_param_id AND p.updated_at = p.created_at;
DROP TABLE planning_params_semeados;

ALTER TABLE planning_params DROP CONSTRAINT planning_params_enterprise_id_fkey;
ALTER TABLE planning_params ADD CONSTRAINT planning_params_enterprise_id_fkey
    FOREIGN KEY (enterprise_id) REFERENCES enterprise(id);
