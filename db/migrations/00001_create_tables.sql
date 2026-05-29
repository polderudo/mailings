-- +goose Up
CREATE EXTENSION IF NOT EXISTS pg_trgm;
CREATE EXTENSION IF NOT EXISTS unaccent;
CREATE EXTENSION IF NOT EXISTS citext;


CREATE TABLE user_profile
(
    id SERIAL PRIMARY KEY NOT null,
    email character varying(256) NOT NULL,
    salutation character varying(256) not null,
    forename character varying(256) not NULL,
    lastname character varying(256) NOT NULL,

    password character varying not null,
    must_reset_password boolean NOT null default false,

    last_login timestamp with time zone NULL,

    created_by_user_profile_id integer null,
    last_updated_by_user_id integer null,

    disabled boolean not null default false,
    email_validated boolean not null default false,

    created_at timestamp with time zone NOT NULL default current_timestamp,
    updated_at timestamp with time zone NULL
);
CREATE unique INDEX user_ix_profile_email ON user_profile (lower(email) );
CREATE INDEX user_ix_profile_forename ON user_profile (lower(forename));
CREATE INDEX user_ix_profile_lastname ON user_profile (lower(lastname));

CREATE TABLE user_group
(
    group_name character varying(256) PRIMARY KEY NOT NULL,
    description character varying(500) NULL,
    can_be_delete boolean not null default false,
    is_admin boolean not null default false,

    changed_by_user_id integer null,

    created_at timestamp with time zone NOT NULL default current_timestamp,
    updated_at timestamp with time zone NULL
);

insert into user_group (group_name, description, can_be_delete, is_admin) values
    ('admin', 'Administrator', false, true),
    ('developer', 'Developer', true, true),
    ('operator', 'Operator Group', true, false),
    ('office', 'Office Group', true, false);

create table token(
    id SERIAL PRIMARY KEY not null,
    user_id int null REFERENCES user_profile(id),
    object_id int null,
    token_name varchar(50) not null,
    token_value varchar(256) not null,
    token_uri varchar(500) not null,
    valid_to timestamp with time zone NOT NULL,
    validated_at timestamp with time zone null,

    created_at timestamp with time zone not null DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp with time zone NULL
);
create unique index token_ix1 on token(token_name, token_value);
CREATE index token_ix2 on token(user_id);
CREATE index token_ix3 on token(object_id);

create table mail
(
    id SERIAL PRIMARY KEY not null,
    mail_from character varying not null,
    mail_to character varying not null,
    mail_subject character varying not null,
    mail_body  character varying not null,
    external_status character varying null,
    external_status_id character varying null,
    internal_status bigint not null,

    created_at timestamp with time zone NOT NULL default current_timestamp,
    updated_at timestamp with time zone NULL default current_timestamp
);

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION public.slugify(
    v TEXT
) RETURNS TEXT
    STRICT IMMUTABLE AS
$$
BEGIN
    -- 1. trim trailing and leading whitespaces from text
    -- 2. remove accents (diacritic signs) from a given text
    -- 3. lowercase unaccented text
    -- 4. replace non-alphanumeric (excluding hyphen, underscore) with a hyphen
    -- 5. trim leading and trailing hyphens
    RETURN trim(BOTH '-' FROM regexp_replace(public.unaccent(trim(v)), '[^.a-z0-9\-_]+', '-', 'gi'));
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

create table client
(
    client_seq BIGSERIAL NOT null,
    ip character varying(256) not null,
    equip_id character varying(256) NOT null,
    browser text,
    order_list_equips text[],

    PRIMARY KEY (ip, browser)
);
create index client_ix1 on client(equip_id);

-- +goose Down
DROP TABLE IF EXISTS casbin_rules_v1;
drop table mail;
drop table token;
drop table user_group;
drop table user_profile;
drop table client;
