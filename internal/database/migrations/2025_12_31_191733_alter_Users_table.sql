-- +migrate Up
ALTER TABLE Users DROP COLUMN apellido;

-- +migrate Down
ALTER TABLE Users ADD COLUMN apellido varchar(100) unique;
