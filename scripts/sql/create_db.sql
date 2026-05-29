-- run as superuser
CREATE USER mailings WITH PASSWORD 'mailings123';
CREATE DATABASE mailings
    WITH
    OWNER = mailings
    ENCODING = 'UTF8'
    LC_COLLATE = 'en_US.utf8'
    LC_CTYPE = 'en_US.utf8'
    TABLESPACE = pg_default
    CONNECTION LIMIT = -1;

ALTER USER mailings WITH SUPERUSER;
GRANT ALL PRIVILEGES ON DATABASE mailings TO mailings;
GRANT SELECT, UPDATE ON pg_catalog.pg_attribute TO mailings;
GRANT USAGE ON SCHEMA pg_catalog TO mailings;

CREATE EXTENSION IF NOT EXISTS pg_trgm;
CREATE EXTENSION IF NOT EXISTS unaccent;
CREATE EXTENSION IF NOT EXISTS citext;
