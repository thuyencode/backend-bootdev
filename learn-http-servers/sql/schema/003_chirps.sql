-- +goose Up
ALTER TABLE
    chirps DROP COLUMN email;

-- +goose Down
ALTER TABLE
    chirps
ADD
    email TEXT NOT NULL UNIQUE;
