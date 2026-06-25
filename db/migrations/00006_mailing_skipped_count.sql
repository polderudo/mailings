-- +goose Up

-- skipped_count hält die Anzahl der beim Versand NICHT gesendeten Empfänger-
-- Adressen eines Mailings fest: Adressen, die vor dem Postmark-Bulk-Request
-- herausgefiltert wurden, weil sie ungültig sind oder auf der Blacklist stehen
-- (gespiegelte Postmark-Suppressions + manuelle Sperren). Wird in SendBulk
-- gesetzt.
alter table mailing add column skipped_count integer not null default 0;

-- +goose Down
alter table mailing drop column if exists skipped_count;
