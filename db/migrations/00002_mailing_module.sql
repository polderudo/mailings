-- +goose Up

create table mail_template
(
    id            SERIAL PRIMARY KEY NOT NULL,
    name          varchar(256) NOT NULL,
    subject       varchar(500) NOT NULL DEFAULT '',
    body_html     text NOT NULL DEFAULT '',

    created_by_user_id  integer NULL REFERENCES user_profile(id),
    updated_by_user_id  integer NULL REFERENCES user_profile(id),

    created_at    timestamp with time zone NOT NULL DEFAULT current_timestamp,
    updated_at    timestamp with time zone NULL
);
create unique index mail_template_ix_name on mail_template (lower(name));

create table mail_list
(
    id            SERIAL PRIMARY KEY NOT NULL,
    name          varchar(256) NOT NULL,
    description   varchar(500) NOT NULL DEFAULT '',

    created_by_user_id  integer NULL REFERENCES user_profile(id),
    updated_by_user_id  integer NULL REFERENCES user_profile(id),

    created_at    timestamp with time zone NOT NULL DEFAULT current_timestamp,
    updated_at    timestamp with time zone NULL
);
create unique index mail_list_ix_name on mail_list (lower(name));

create table mail_domain
(
    id                  SERIAL PRIMARY KEY NOT NULL,
    domain              varchar(256) NOT NULL,
    from_email          varchar(256) NOT NULL,
    from_name           varchar(256) NOT NULL DEFAULT '',
    postmark_stream_id  varchar(256) NOT NULL DEFAULT '',
    is_active           boolean NOT NULL DEFAULT true,

    created_at          timestamp with time zone NOT NULL DEFAULT current_timestamp,
    updated_at          timestamp with time zone NULL
);
create unique index mail_domain_ix_domain on mail_domain (lower(domain));

insert into mail_domain (domain, from_email, from_name, postmark_stream_id, is_active) values
    ('nakami.de', 'noreply@nakami.de', 'Nakami Lounge', 'broadcast', true);

create table mailing
(
    id                  SERIAL PRIMARY KEY NOT NULL,
    name                varchar(256) NOT NULL,
    template_id         integer NOT NULL REFERENCES mail_template(id),
    list_id             integer NOT NULL REFERENCES mail_list(id),
    domain_id           integer NOT NULL REFERENCES mail_domain(id),

    status              varchar(32) NOT NULL DEFAULT 'draft',
    subject_snapshot    varchar(500) NOT NULL DEFAULT '',
    body_snapshot       text NOT NULL DEFAULT '',

    -- total_recipients gibt es bewusst nicht: die verknüpfte mail_list darf
    -- sich nach Anlegen des Mailings noch ändern, daher wird die Anzahl der
    -- Empfänger immer live aus mail_list_recipient ermittelt.
    sent_count          integer NOT NULL DEFAULT 0,
    failed_count        integer NOT NULL DEFAULT 0,

    started_at          timestamp with time zone NULL,
    finished_at         timestamp with time zone NULL,

    created_by_user_id  integer NULL REFERENCES user_profile(id),
    updated_by_user_id  integer NULL REFERENCES user_profile(id),

    created_at          timestamp with time zone NOT NULL DEFAULT current_timestamp,
    updated_at          timestamp with time zone NULL
);
create index mailing_ix_status on mailing (status);
create index mailing_ix_template on mailing (template_id);
create index mailing_ix_list on mailing (list_id);
create index mailing_ix_domain on mailing (domain_id);

create table mail_list_recipient
(
    id            SERIAL PRIMARY KEY NOT NULL,
    list_id       integer NOT NULL REFERENCES mail_list(id) ON DELETE CASCADE,
    email         varchar(256) NOT NULL,
    forename      varchar(256) NOT NULL DEFAULT '',
    lastname      varchar(256) NOT NULL DEFAULT '',

    -- Letzter Versand-Status pro Empfänger; wird vom jüngsten Mailing überschrieben.
    last_status              varchar(32)  NOT NULL DEFAULT '',
    last_postmark_message_id varchar(256) NOT NULL DEFAULT '',
    last_error               text         NOT NULL DEFAULT '',
    last_sent_at             timestamp with time zone NULL,
    last_mailing_id          integer NULL REFERENCES mailing(id) ON DELETE SET NULL,

    created_at    timestamp with time zone NOT NULL DEFAULT current_timestamp
);
create unique index mail_list_recipient_ix_unique       on mail_list_recipient (list_id, lower(email));
create index        mail_list_recipient_ix_list         on mail_list_recipient (list_id);
create index        mail_list_recipient_ix_last_mailing on mail_list_recipient (last_mailing_id);
create index        mail_list_recipient_ix_last_status  on mail_list_recipient (last_status);

-- +goose Down
drop table if exists mail_list_recipient;
drop table if exists mailing;
drop table if exists mail_domain;
drop table if exists mail_list;
drop table if exists mail_template;
