-- Keeps databases that ran an earlier development version of migration 005 in
-- sync with the final balanced 3/3/3 question bank. The guards make this a
-- no-op for fresh installations.
UPDATE grammar_topics
SET practice = practice || $json$[
  {"id":"nr-p5","kind":"choice","difficulty":1,"prompt":"После числа","phrase":"На столе восемь ___. (чашка)","options":["чашка","чашки","чашек"],"answer":"чашек","explanation":"После пяти и больше нужен родительный множественного: восемь чашек."}
]$json$::jsonb
WHERE id = 'grammar_numbers_reflexive'
  AND NOT EXISTS (SELECT 1 FROM jsonb_array_elements(practice) AS question WHERE question->>'id' = 'nr-p5');

UPDATE grammar_topics
SET game = game || $json$[
  {"id":"nr-a6","kind":"choice","difficulty":2,"prompt":"Составное число","phrase":"В книге двести тридцать четыре ___. (страница)","options":["страница","страницы","страниц"],"answer":"страницы","explanation":"234 оканчивается на 4, но не на 14: четыре страницы."},
  {"id":"nr-a7","kind":"choice","difficulty":3,"prompt":"Согласуйте сказуемое","phrase":"Двадцать один студент ___ тест.","options":["написал","написали","написало"],"answer":"написал","explanation":"Сочетание с один требует сказуемого в единственном числе."}
]$json$::jsonb
WHERE id = 'grammar_numbers_reflexive'
  AND NOT EXISTS (SELECT 1 FROM jsonb_array_elements(game) AS question WHERE question->>'id' = 'nr-a6');

UPDATE grammar_topics
SET game = game || $json$[
  {"id":"rv-g9","kind":"choice","difficulty":1,"prompt":"Выберите форму","phrase":"Ты часто ошиба__?","options":["ешь","ешься","ется"],"answer":"ешься","explanation":"Ошибаться → ты ошибаешься."}
]$json$::jsonb
WHERE id = 'grammar_reflexive'
  AND NOT EXISTS (SELECT 1 FROM jsonb_array_elements(game) AS question WHERE question->>'id' = 'rv-g9');
