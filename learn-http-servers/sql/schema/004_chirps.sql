-- +goose Up
ALTER TABLE chirps
ADD body TEXT NOT NULL;

-- +goose Down
ALTER TABLE chirps
DROP COLUMN body;
