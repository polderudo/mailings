-- +goose Up

-- mail_blacklist spiegelt die Postmark-Suppression-Liste (pro Message-Stream)
-- in eine globale, list-übergreifende Sperrliste. Adressen landen hier nach
-- Hard-Bounce, Spam-Beschwerde, Abmeldung (pm:unsubscribe) oder manueller
-- Sperrung. Der Cron-Job mail_blacklist-sync gleicht die Tabelle gegen den
-- Postmark Suppressions-Dump ab; SendBulk filtert geblockte Adressen vor dem
-- Versand heraus.
create table mail_blacklist
(
    id          SERIAL PRIMARY KEY NOT NULL,
    email       varchar(256) NOT NULL,

    -- HardBounce | SpamComplaint | ManualSuppression (Postmark) bzw. leer/manual
    reason      varchar(32)  NOT NULL DEFAULT '',
    -- Recipient | Customer | Admin (Postmark-Origin)
    origin      varchar(32)  NOT NULL DEFAULT '',
    -- Postmark-Message-Stream, aus dem die Suppression stammt (z. B. broadcast)
    stream      varchar(256) NOT NULL DEFAULT '',
    -- Herkunft des Eintrags: 'postmark' (gespiegelt, wird gegen den Dump
    -- abgeglichen/entfernt) oder 'manual' (lokal gesetzt, bleibt unangetastet).
    source      varchar(16)  NOT NULL DEFAULT 'postmark',

    created_at  timestamp with time zone NOT NULL DEFAULT current_timestamp,
    updated_at  timestamp with time zone NULL
);
create unique index mail_blacklist_ix_email   on mail_blacklist (lower(email));
create index        mail_blacklist_ix_source  on mail_blacklist (source);
create index        mail_blacklist_ix_reason  on mail_blacklist (reason);

-- +goose Down
drop table if exists mail_blacklist;
