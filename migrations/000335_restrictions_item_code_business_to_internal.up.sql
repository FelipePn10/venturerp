-- Mesma correção da 334, agora para as restrições: elas eram gravadas com o
-- código de negócio do item (900700), enquanto o configurador as consulta pela
-- chave interna de `items.code`. A regra existia no cadastro e não bloqueava
-- nada na hora de configurar o produto.
--
-- Idempotente: só reaponta a linha cujo valor atual não é uma chave interna
-- válida e é um código de negócio conhecido.
UPDATE restrictions AS alvo
   SET item_code = itens.code
  FROM items AS itens
 WHERE itens.business_code = alvo.item_code::text
   AND alvo.item_code IS NOT NULL
   AND NOT EXISTS (SELECT 1 FROM items AS ok WHERE ok.code = alvo.item_code);
