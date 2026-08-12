-- +goose Up

-- Reine Text-Mailings: eine Vorlage ist entweder ein HTML-Newsletter (Default,
-- TinyMCE-Editor, wird beim Versand in das newsletter_html-Gerüst gehüllt) oder
-- eine reine Text-Mail (Plain-Textarea, geht 1:1 als TextBody an Postmark).
--
-- body_html und body_text liegen bewusst in getrennten Spalten: das Umschalten
-- des Formats darf den Inhalt der jeweils anderen Variante nicht zerstören.
alter table mail_template add column format    varchar(16) not null default 'html';
alter table mail_template add column body_text text        not null default '';

-- Das Mailing friert beim Anlegen Betreff + Body der Vorlage ein. Damit der
-- Versand später weiß, wie body_snapshot zu interpretieren ist (HTML-Wrapper
-- oder Plain-Text), wandert das Format mit in den Snapshot. body_snapshot hält
-- je nach format den HTML- oder den Text-Body.
alter table mailing add column format varchar(16) not null default 'html';

-- +goose Down
alter table mailing       drop column if exists format;
alter table mail_template drop column if exists body_text;
alter table mail_template drop column if exists format;
