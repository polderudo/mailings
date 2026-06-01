-- +goose Up
-- postmark_status_error hält den Fehlertext aus einem fehlgeschlagenen Versand
-- (POST /email/bulk bzw. Validierung). Wird im Frontend unter der Bulk-Request-ID
-- angezeigt, um den Fehlerfall nachvollziehen zu können. text, da Postmark-Fehler
-- (inkl. der feldbezogenen Errors) länger werden können.
ALTER TABLE mailing ADD COLUMN postmark_status_error text NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE mailing DROP COLUMN postmark_status_error;
