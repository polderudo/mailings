-- +goose Up
ALTER TABLE mail_template ADD COLUMN archived boolean NOT NULL DEFAULT false;
ALTER TABLE mail_list     ADD COLUMN archived boolean NOT NULL DEFAULT false;
ALTER TABLE mailing       ADD COLUMN archived boolean NOT NULL DEFAULT false;

-- +goose Down
ALTER TABLE mailing       DROP COLUMN archived;
ALTER TABLE mail_list     DROP COLUMN archived;
ALTER TABLE mail_template DROP COLUMN archived;
