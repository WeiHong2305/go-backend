-- +goose Up
UPDATE movies SET runtime_minutes = 148, genre = 'Sci-Fi', rating = 8.8 WHERE title = 'Inception';
UPDATE movies SET runtime_minutes = 127, genre = 'Sci-Fi', rating = 8.2 WHERE title = 'Jurassic Park';
UPDATE movies SET runtime_minutes = 154, genre = 'Crime', rating = 8.9 WHERE title = 'Pulp Fiction';
UPDATE movies SET runtime_minutes = 151, genre = 'Crime', rating = 8.5 WHERE title = 'The Departed';
UPDATE movies SET runtime_minutes = 195, genre = 'Drama', rating = 9.0 WHERE title = 'Schindler''s List';
UPDATE movies SET runtime_minutes = 109, genre = 'Thriller', rating = 8.5 WHERE title = 'Psycho';

-- +goose Down
UPDATE movies SET runtime_minutes = NULL, genre = NULL, rating = NULL 
WHERE title IN (
    'Inception', 'Jurassic Park', 'Pulp Fiction', 'The Departed', 'Schindler''s List', 'Psycho'
);
