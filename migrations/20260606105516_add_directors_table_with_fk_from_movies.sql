-- +goose Up
CREATE TABLE directors (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

ALTER TABLE movies
DROP COLUMN director;

ALTER TABLE movies
ADD COLUMN director_id BIGINT NOT NULL;

ALTER TABLE movies
ADD CONSTRAINT fk_director
    FOREIGN KEY (director_id)
    REFERENCES directors(id)
    ON DELETE RESTRICT;

CREATE INDEX idx_movies_director_id ON movies(director_id);

-- +goose Down
ALTER TABLE movies
DROP CONSTRAINT fk_director;

ALTER TABLE movies DROP COLUMN director_id;
ALTER TABLE movies ADD COLUMN director VARCHAR(255) NOT NULL;

DROP TABLE directors;