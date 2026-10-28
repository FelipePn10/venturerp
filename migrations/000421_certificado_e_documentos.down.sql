DROP TRIGGER IF EXISTS trg_quality_document_revisions_congeladas ON quality_document_revisions;
DROP FUNCTION IF EXISTS quality_document_revisions_congeladas();
DROP TABLE IF EXISTS quality_document_revisions;
DROP TABLE IF EXISTS quality_documents;
DROP TRIGGER IF EXISTS trg_quality_certificates_congelado ON quality_certificates;
DROP FUNCTION IF EXISTS quality_certificates_congelado();
DROP TABLE IF EXISTS quality_certificates;
SELECT instalar_conferencia_de_empresa();
