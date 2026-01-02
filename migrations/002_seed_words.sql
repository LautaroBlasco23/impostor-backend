ALTER TABLE words ADD CONSTRAINT unique_word UNIQUE (text, category);

INSERT INTO words (text, category) VALUES
-- Animals
('lion', 'animals'),
('elephant', 'animals'),
('giraffe', 'animals'),
('penguin', 'animals'),
('dolphin', 'animals'),
('tiger', 'animals'),
('kangaroo', 'animals'),
('panda', 'animals'),

-- Food
('pizza', 'food'),
('sushi', 'food'),
('burger', 'food'),
('pasta', 'food'),
('tacos', 'food'),
('ice cream', 'food'),
('chocolate', 'food'),
('pancakes', 'food'),

-- Sports
('soccer', 'sports'),
('basketball', 'sports'),
('tennis', 'sports'),
('swimming', 'sports'),
('baseball', 'sports'),
('volleyball', 'sports'),
('boxing', 'sports'),
('golf', 'sports'),

-- Countries
('france', 'countries'),
('japan', 'countries'),
('brazil', 'countries'),
('canada', 'countries'),
('australia', 'countries'),
('germany', 'countries'),
('italy', 'countries'),
('mexico', 'countries'),

-- Movies
('avatar', 'movies'),
('titanic', 'movies'),
('inception', 'movies'),
('matrix', 'movies'),
('star wars', 'movies'),
('jurassic park', 'movies'),
('avengers', 'movies'),
('gladiator', 'movies'),

-- Professions
('doctor', 'professions'),
('teacher', 'professions'),
('engineer', 'professions'),
('artist', 'professions'),
('chef', 'professions'),
('lawyer', 'professions'),
('pilot', 'professions'),
('firefighter', 'professions');
