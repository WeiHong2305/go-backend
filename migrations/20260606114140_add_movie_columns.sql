-- +goose Up
ALTER TABLE movies
ADD COLUMN runtime_minutes INT,
ADD COLUMN genre VARCHAR(100),
ADD COLUMN rating NUMERIC(3, 1) CHECK (rating >= 0 AND rating <= 10);

-- +goose Down
ALTER TABLE movies
DROP COLUMN runtime_minutes,
DROP COLUMN genre,
DROP COLUMN rating;
