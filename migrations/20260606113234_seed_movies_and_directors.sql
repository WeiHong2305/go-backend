-- +goose Up
INSERT INTO directors (name) VALUES
    ('Christopher Nolan'),
    ('Steven Spielberg'),
    ('Quentin Tarantino'),
    ('Martin Scorsese'),
    ('Alfred Hitchcock');

INSERT INTO movies (title, director_id, release_year) VALUES
('Inception', (SELECT id FROM directors WHERE name = 'Christopher Nolan'), 2010),
('Jurassic Park', (SELECT id FROM directors WHERE name = 'Steven Spielberg'), 1993),
('Pulp Fiction', (SELECT id FROM directors WHERE name = 'Quentin Tarantino'), 1994),
('The Departed', (SELECT id FROM directors WHERE name = 'Martin Scorsese'), 2006),
('Schindler''s List', (SELECT id FROM directors WHERE name = 'Steven Spielberg'), 1993),
('Psycho', (SELECT id FROM directors WHERE name = 'Alfred Hitchcock'), 1960);

-- +goose Down
DELETE FROM movies WHERE title IN (
    'Inception', 'Jurassic Park', 'Pulp Fiction', 'The Departed', 'Schindler''s List', 'Psycho'
);
DELETE FROM directors WHERE name IN (
    'Christopher Nolan', 'Steven Spielberg', 'Quentin Tarantino', 'Martin Scorsese', 'Alfred Hitchcock'
);
