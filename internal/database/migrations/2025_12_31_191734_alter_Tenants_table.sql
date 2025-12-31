-- +migrate Up
ALTER TABLE Tenants ADD COLUMN padron varchar(100) unique;
ALTER TABLE Tenants DROP COLUMN crespo;

-- +migrate Down
ALTER TABLE Tenants ADD COLUMN crespo varchar(100) unique;
ALTER TABLE Tenants DROP COLUMN padron;
