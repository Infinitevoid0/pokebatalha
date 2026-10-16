-- roda só na criação do volume: um banco e um usuário por serviço,
-- sem acesso ao banco dos outros.

CREATE USER catalogo WITH PASSWORD 'catalogo';
CREATE DATABASE catalogo_db OWNER catalogo;
REVOKE CONNECT ON DATABASE catalogo_db FROM PUBLIC;

CREATE USER jogadores WITH PASSWORD 'jogadores';
CREATE DATABASE jogadores_db OWNER jogadores;
REVOKE CONNECT ON DATABASE jogadores_db FROM PUBLIC;

CREATE USER batalha WITH PASSWORD 'batalha';
CREATE DATABASE batalha_db OWNER batalha;
REVOKE CONNECT ON DATABASE batalha_db FROM PUBLIC;
