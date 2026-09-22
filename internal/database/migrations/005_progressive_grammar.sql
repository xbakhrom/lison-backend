ALTER TABLE user_grammar_progress
    ADD COLUMN mastery_level smallint NOT NULL DEFAULT 1 CHECK (mastery_level BETWEEN 1 AND 3);

ALTER TABLE grammar_review_logs
    ADD COLUMN mastery_level smallint NOT NULL DEFAULT 1 CHECK (mastery_level BETWEEN 1 AND 3);

-- The route follows prerequisites: word agreement, core verb forms, the two
-- first cases, time, then the A2 case system and fluency topics.
UPDATE grammar_topics SET position = 1, level = 'A1' WHERE id = 'grammar_gender_number';
UPDATE grammar_topics SET position = 2, level = 'A1' WHERE id = 'grammar_present';
UPDATE grammar_topics SET position = 3, level = 'A1' WHERE id = 'grammar_pronouns_modals';
UPDATE grammar_topics SET position = 4, level = 'A1' WHERE id = 'grammar_accusative';
UPDATE grammar_topics SET position = 5, level = 'A1' WHERE id = 'grammar_prepositional';
UPDATE grammar_topics SET position = 6, level = 'A1–A2' WHERE id = 'grammar_past_future';
UPDATE grammar_topics SET position = 7, level = 'A2' WHERE id = 'grammar_adjectives';
UPDATE grammar_topics SET position = 8, level = 'A2' WHERE id = 'grammar_genitive';
UPDATE grammar_topics SET position = 9, level = 'A2' WHERE id = 'grammar_dative';
UPDATE grammar_topics SET position = 10, level = 'A2' WHERE id = 'grammar_instrumental';
UPDATE grammar_topics SET position = 11, level = 'A2' WHERE id = 'grammar_aspect';
UPDATE grammar_topics SET position = 12, level = 'A2' WHERE id = 'grammar_numbers_reflexive';

-- Existing guided tasks are introductory. Existing game tasks are distributed
-- across the three steps so old content also participates in adaptation.
UPDATE grammar_topics AS topic
SET practice = (
        SELECT COALESCE(jsonb_agg(question || jsonb_build_object('difficulty', 1, 'kind', 'choice') ORDER BY ordinal), '[]'::jsonb)
        FROM jsonb_array_elements(topic.practice) WITH ORDINALITY AS source(question, ordinal)
    ),
    game = (
        SELECT COALESCE(jsonb_agg(
            question || jsonb_build_object(
                'difficulty', CASE WHEN ordinal <= 2 THEN 1 WHEN ordinal = 3 THEN 2 ELSE 3 END,
                'kind', 'choice'
            ) ORDER BY ordinal
        ), '[]'::jsonb)
        FROM jsonb_array_elements(topic.game) WITH ORDINALITY AS source(question, ordinal)
    )
WHERE status = 'published';

UPDATE grammar_topics SET
practice = practice || $json$[
  {"id":"gn-p4","kind":"choice","difficulty":1,"prompt":"Согласуйте","phrase":"___ море было спокойным.","options":["Синий","Синяя","Синее"],"answer":"Синее","explanation":"Море — средний род: синее море."}
]$json$::jsonb,
game = game || $json$[
  {"id":"gn-a1","kind":"true_false","difficulty":1,"prompt":"Верно или нет?","phrase":"«Новая словарь» — правильное сочетание.","options":["Верно","Неверно"],"answer":"Неверно","explanation":"Словарь — мужского рода: новый словарь."},
  {"id":"gn-a2","kind":"order","difficulty":2,"prompt":"Соберите предложение","phrase":"Слова уже даны — расставьте их по порядку.","tokens":["Новые","книги","лежали","на","столе."],"answer":"Новые книги лежали на столе.","explanation":"Определение и сказуемое согласуются с множественным числом."},
  {"id":"gn-a3","kind":"choice","difficulty":2,"prompt":"Найдите форму","phrase":"В саду растут молодые ___. (дерево)","options":["деревы","деревья","дерева"],"answer":"деревья","explanation":"Дерево имеет особую форму множественного числа: деревья."},
  {"id":"gn-a4","kind":"order","difficulty":3,"prompt":"Соберите без ошибки","phrase":"Учитывайте род и прошедшее время.","tokens":["Маленькая","девочка","быстро","уснула."],"answer":"Маленькая девочка быстро уснула.","explanation":"Девочка, маленькая и уснула стоят в женском роде."},
  {"id":"gn-a5","kind":"true_false","difficulty":3,"prompt":"Проверьте исключение","phrase":"Множественное число слова «ухо» — «ухи».","options":["Верно","Неверно"],"answer":"Неверно","explanation":"Правильная форма — уши."}
]$json$::jsonb
WHERE id = 'grammar_gender_number';

UPDATE grammar_topics SET
practice = practice || $json$[
  {"id":"pr-p4","kind":"choice","difficulty":1,"prompt":"Выберите окончание","phrase":"Вы работа__ сегодня?","options":["ете","ите","ют"],"answer":"ете","explanation":"Работать — I спряжение: вы работаете."}
]$json$::jsonb,
game = game || $json$[
  {"id":"pr-a1","kind":"true_false","difficulty":1,"prompt":"Верно или нет?","phrase":"«Она читают книгу» — правильная форма.","options":["Верно","Неверно"],"answer":"Неверно","explanation":"С местоимением она нужна форма читает."},
  {"id":"pr-a2","kind":"order","difficulty":2,"prompt":"Соберите предложение","phrase":"Поставьте слова в естественном порядке.","tokens":["Мы","часто","говорим","по-русски."],"answer":"Мы часто говорим по-русски.","explanation":"Мы говорим — форма II спряжения."},
  {"id":"pr-a3","kind":"choice","difficulty":2,"prompt":"Учтите чередование","phrase":"Я искать → я ___.","options":["искаю","ищу","искую"],"answer":"ищу","explanation":"В форме я: искать → ищу."},
  {"id":"pr-a4","kind":"order","difficulty":3,"prompt":"Соберите вопрос","phrase":"Не забудьте особую форму глагола.","tokens":["Почему","ты","не","хочешь","есть?"],"answer":"Почему ты не хочешь есть?","explanation":"Хотеть: ты хочешь; после него используется инфинитив."},
  {"id":"pr-a5","kind":"true_false","difficulty":3,"prompt":"Проверьте исключение","phrase":"Глагол «держать» относится ко II спряжению.","options":["Верно","Неверно"],"answer":"Верно","explanation":"Держать — одно из исключений II спряжения."}
]$json$::jsonb
WHERE id = 'grammar_present';

UPDATE grammar_topics SET
practice = practice || $json$[
  {"id":"pm-p4","kind":"choice","difficulty":1,"prompt":"Выберите местоимение","phrase":"___ живём в Ташкенте.","options":["Мы","Нас","Нам"],"answer":"Мы","explanation":"Для подлежащего нужна форма мы."}
]$json$::jsonb,
game = game || $json$[
  {"id":"pm-a1","kind":"true_false","difficulty":1,"prompt":"Верно или нет?","phrase":"«Я хочу учиться» — правильная конструкция.","options":["Верно","Неверно"],"answer":"Верно","explanation":"После хотеть ставится инфинитив."},
  {"id":"pm-a2","kind":"order","difficulty":2,"prompt":"Соберите просьбу","phrase":"Поставьте слова по порядку.","tokens":["Вы","можете","мне","помочь?"],"answer":"Вы можете мне помочь?","explanation":"Мочь согласуется с вы, а помогать требует дательного: мне."},
  {"id":"pm-a3","kind":"choice","difficulty":2,"prompt":"После предлога","phrase":"Мы думаем о ___. (она)","options":["ей","неё","ней"],"answer":"ней","explanation":"После о употребляется предложный: о ней."},
  {"id":"pm-a4","kind":"order","difficulty":3,"prompt":"Соберите фразу","phrase":"Различите личную и общую возможность.","tokens":["Здесь","нельзя","оставлять","машину."],"answer":"Здесь нельзя оставлять машину.","explanation":"Нельзя выражает общее запрещение."},
  {"id":"pm-a5","kind":"true_false","difficulty":3,"prompt":"Проверьте форму","phrase":"После «благодаря» правильно говорить «благодаря нему».","options":["Верно","Неверно"],"answer":"Неверно","explanation":"После благодаря начальное н- не добавляется: благодаря ему."}
]$json$::jsonb
WHERE id = 'grammar_pronouns_modals';

UPDATE grammar_topics SET
practice = practice || $json$[
  {"id":"vn-p4","kind":"choice","difficulty":1,"prompt":"Кого?","phrase":"Мы встретили ___. (учитель)","options":["учитель","учителя","учителю"],"answer":"учителя","explanation":"Одушевлённый объект мужского рода: учителя."}
]$json$::jsonb,
game = game || $json$[
  {"id":"vn-a1","kind":"true_false","difficulty":1,"prompt":"Верно или нет?","phrase":"«Я люблю музыка» — правильная форма.","options":["Верно","Неверно"],"answer":"Неверно","explanation":"Любить что? Музыку."},
  {"id":"vn-a2","kind":"order","difficulty":2,"prompt":"Соберите предложение","phrase":"Покажите направление.","tokens":["После","работы","мы","идём","в","кафе."],"answer":"После работы мы идём в кафе.","explanation":"Куда? В кафе; несклоняемое слово сохраняет форму."},
  {"id":"vn-a3","kind":"choice","difficulty":2,"prompt":"Одушевлённость","phrase":"Я вижу новых ___. (студенты)","options":["студенты","студентов","студентами"],"answer":"студентов","explanation":"Во множественном числе одушевлённый винительный совпадает с родительным."},
  {"id":"vn-a4","kind":"order","difficulty":3,"prompt":"Соберите без ошибки","phrase":"Согласуйте определение с объектом.","tokens":["Она","купила","красивую","зимнюю","куртку."],"answer":"Она купила красивую зимнюю куртку.","explanation":"Все зависимые слова стоят в винительном женского рода."},
  {"id":"vn-a5","kind":"true_false","difficulty":3,"prompt":"Где или куда?","phrase":"В предложении «Книга лежит на стол» падеж выбран верно.","options":["Верно","Неверно"],"answer":"Неверно","explanation":"Лежит где? На столе — нужен предложный падеж."}
]$json$::jsonb
WHERE id = 'grammar_accusative';

UPDATE grammar_topics SET
practice = practice || $json$[
  {"id":"pp-p4","kind":"choice","difficulty":1,"prompt":"О чём?","phrase":"Она рассказала о ___. (семья)","options":["семью","семье","семьи"],"answer":"семье","explanation":"О чём? О семье."}
]$json$::jsonb,
game = game || $json$[
  {"id":"pp-a1","kind":"true_false","difficulty":1,"prompt":"Верно или нет?","phrase":"«Мы живём в городе» — предложный падеж.","options":["Верно","Неверно"],"answer":"Верно","explanation":"Где? В городе — предложный падеж."},
  {"id":"pp-a2","kind":"order","difficulty":2,"prompt":"Соберите предложение","phrase":"Укажите место действия.","tokens":["Мои","друзья","учатся","в","университете."],"answer":"Мои друзья учатся в университете.","explanation":"Учиться где? В университете."},
  {"id":"pp-a3","kind":"choice","difficulty":2,"prompt":"Форма на -ия","phrase":"Документы лежат в ___. (папка)","options":["папку","папке","папки"],"answer":"папке","explanation":"Где? В папке."},
  {"id":"pp-a4","kind":"order","difficulty":3,"prompt":"Соберите контраст","phrase":"Различите место и тему разговора.","tokens":["Мы","гуляли","в","саду","и","говорили","о","саде."],"answer":"Мы гуляли в саду и говорили о саде.","explanation":"Для места — в саду, для темы — о саде."},
  {"id":"pp-a5","kind":"true_false","difficulty":3,"prompt":"Проверьте окончание","phrase":"Слово на -ие получает форму «о здании».","options":["Верно","Неверно"],"answer":"Верно","explanation":"Существительные на -ие имеют окончание -ии."}
]$json$::jsonb
WHERE id = 'grammar_prepositional';

UPDATE grammar_topics SET
practice = practice || $json$[
  {"id":"pf-p4","kind":"choice","difficulty":1,"prompt":"Прошедшее время","phrase":"Дети вчера игра__. ","options":["л","ла","ли"],"answer":"ли","explanation":"Во множественном числе: играли."}
]$json$::jsonb,
game = game || $json$[
  {"id":"pf-a1","kind":"true_false","difficulty":1,"prompt":"Верно или нет?","phrase":"«Она вчера читал» — правильная форма.","options":["Верно","Неверно"],"answer":"Неверно","explanation":"Для женского рода: она читала."},
  {"id":"pf-a2","kind":"order","difficulty":2,"prompt":"Соберите план","phrase":"Используйте сложное будущее.","tokens":["Завтра","мы","будем","готовиться","к","экзамену."],"answer":"Завтра мы будем готовиться к экзамену.","explanation":"Несовершенный вид образует будущее через будем + инфинитив."},
  {"id":"pf-a3","kind":"choice","difficulty":2,"prompt":"Процесс или результат","phrase":"К восьми часам она ___ ужин.","options":["будет готовить","приготовит","готовила"],"answer":"приготовит","explanation":"Срок к восьми часам требует результата."},
  {"id":"pf-a4","kind":"order","difficulty":3,"prompt":"Соберите историю","phrase":"Согласуйте особую форму.","tokens":["Вчера","она","не","смогла","прийти."],"answer":"Вчера она не смогла прийти.","explanation":"Смочь в прошедшем женского рода — смогла."},
  {"id":"pf-a5","kind":"true_false","difficulty":3,"prompt":"Проверьте вид","phrase":"У совершенного вида есть настоящее время.","options":["Верно","Неверно"],"answer":"Неверно","explanation":"Формы совершенного вида, похожие на настоящее, обозначают будущее."}
]$json$::jsonb
WHERE id = 'grammar_past_future';

UPDATE grammar_topics SET
practice = practice || $json$[
  {"id":"aj-p4","kind":"choice","difficulty":1,"prompt":"Как?","phrase":"Поезд едет очень ___.","options":["быстрый","быстро","быстрее"],"answer":"быстро","explanation":"Едет как? Быстро — это наречие."}
]$json$::jsonb,
game = game || $json$[
  {"id":"aj-a1","kind":"true_false","difficulty":1,"prompt":"Верно или нет?","phrase":"В сочетании «интересная книга» слова согласованы.","options":["Верно","Неверно"],"answer":"Верно","explanation":"Оба слова стоят в женском роде, единственном числе."},
  {"id":"aj-a2","kind":"order","difficulty":2,"prompt":"Соберите предложение","phrase":"Различите признак и образ действия.","tokens":["Новый","сотрудник","работает","быстро."],"answer":"Новый сотрудник работает быстро.","explanation":"Новый — прилагательное, быстро — наречие."},
  {"id":"aj-a3","kind":"choice","difficulty":2,"prompt":"Сравните","phrase":"Сегодня погода ___, чем вчера.","options":["хорошая","хорошо","лучше"],"answer":"лучше","explanation":"Сравнительная форма от хороший/хорошо — лучше."},
  {"id":"aj-a4","kind":"order","difficulty":3,"prompt":"Соберите сравнение","phrase":"Согласуйте прилагательное после предлога.","tokens":["Мы","живём","в","более","тихом","районе."],"answer":"Мы живём в более тихом районе.","explanation":"После в при значении места: в тихом районе."},
  {"id":"aj-a5","kind":"true_false","difficulty":3,"prompt":"Проверьте сравнение","phrase":"Фраза «Этот дом более выше» построена правильно.","options":["Верно","Неверно"],"answer":"Неверно","explanation":"Нельзя смешивать формы: выше или более высокий."}
]$json$::jsonb
WHERE id = 'grammar_adjectives';

UPDATE grammar_topics SET
practice = practice || $json$[
  {"id":"rd-p4","kind":"choice","difficulty":1,"prompt":"Для кого?","phrase":"Это подарок для ___. (сестра)","options":["сестры","сестре","сестру"],"answer":"сестры","explanation":"После для нужен родительный: для сестры."}
]$json$::jsonb,
game = game || $json$[
  {"id":"rd-a1","kind":"true_false","difficulty":1,"prompt":"Верно или нет?","phrase":"«У меня нет времени» — родительный падеж.","options":["Верно","Неверно"],"answer":"Верно","explanation":"Нет чего? Времени."},
  {"id":"rd-a2","kind":"order","difficulty":2,"prompt":"Соберите предложение","phrase":"Используйте два предлога с родительным.","tokens":["После","урока","он","пришёл","из","школы."],"answer":"После урока он пришёл из школы.","explanation":"После и из требуют родительного падежа."},
  {"id":"rd-a3","kind":"choice","difficulty":2,"prompt":"Количество","phrase":"В группе мало ___. (студенты)","options":["студенты","студентов","студентам"],"answer":"студентов","explanation":"После мало нужен родительный множественного."},
  {"id":"rd-a4","kind":"order","difficulty":3,"prompt":"Соберите без ошибки","phrase":"Учтите формы множественного числа.","tokens":["У","моих","друзей","нет","свободного","времени."],"answer":"У моих друзей нет свободного времени.","explanation":"У кого? друзей; нет чего? времени."},
  {"id":"rd-a5","kind":"true_false","difficulty":3,"prompt":"Проверьте значение","phrase":"«Я пришёл с другом» отвечает на вопрос «откуда?».","options":["Верно","Неверно"],"answer":"Неверно","explanation":"С другом означает совместность; откуда — с работы, из школы."}
]$json$::jsonb
WHERE id = 'grammar_genitive';

UPDATE grammar_topics SET
practice = practice || $json$[
  {"id":"dt-p4","kind":"choice","difficulty":1,"prompt":"Кому?","phrase":"Учитель объяснил правило ___. (мы)","options":["нас","нам","нами"],"answer":"нам","explanation":"Объяснить кому? Нам."}
]$json$::jsonb,
game = game || $json$[
  {"id":"dt-a1","kind":"true_false","difficulty":1,"prompt":"Верно или нет?","phrase":"По-русски правильно говорить «Мне холодно».","options":["Верно","Неверно"],"answer":"Верно","explanation":"Состояние выражается дательным падежом."},
  {"id":"dt-a2","kind":"order","difficulty":2,"prompt":"Соберите совет","phrase":"Поставьте адресата в дательный.","tokens":["Врач","посоветовал","мне","больше","отдыхать."],"answer":"Врач посоветовал мне больше отдыхать.","explanation":"Посоветовать кому? Мне."},
  {"id":"dt-a3","kind":"choice","difficulty":2,"prompt":"Форма на -ия","phrase":"Я позвонил ___. (Мария)","options":["Марии","Марию","Марией"],"answer":"Марии","explanation":"Дательный от Мария — Марии."},
  {"id":"dt-a4","kind":"order","difficulty":3,"prompt":"Соберите фразу","phrase":"Используйте безличную конструкцию.","tokens":["Ей","нужно","позвонить","новому","клиенту."],"answer":"Ей нужно позвонить новому клиенту.","explanation":"Ей нужно; позвонить кому? новому клиенту."},
  {"id":"dt-a5","kind":"true_false","difficulty":3,"prompt":"Проверьте предлог","phrase":"После предлога «к» нужен дательный падеж.","options":["Верно","Неверно"],"answer":"Верно","explanation":"К врачу, к другу, ко мне — дательный падеж."}
]$json$::jsonb
WHERE id = 'grammar_dative';

UPDATE grammar_topics SET
practice = practice || $json$[
  {"id":"tv-p4","kind":"choice","difficulty":1,"prompt":"С чем?","phrase":"Чай с ___. (лимон)","options":["лимона","лимоном","лимону"],"answer":"лимоном","explanation":"С чем? С лимоном."}
]$json$::jsonb,
game = game || $json$[
  {"id":"tv-a1","kind":"true_false","difficulty":1,"prompt":"Верно или нет?","phrase":"«Она работает врачом» — творительный падеж.","options":["Верно","Неверно"],"answer":"Верно","explanation":"Работать кем? Врачом."},
  {"id":"tv-a2","kind":"order","difficulty":2,"prompt":"Соберите предложение","phrase":"Назовите инструмент действия.","tokens":["Он","открыл","дверь","старым","ключом."],"answer":"Он открыл дверь старым ключом.","explanation":"Открыл чем? Старым ключом."},
  {"id":"tv-a3","kind":"choice","difficulty":2,"prompt":"Место","phrase":"Машина стоит перед ___. (дом)","options":["дом","домом","доме"],"answer":"домом","explanation":"Перед чем? Перед домом."},
  {"id":"tv-a4","kind":"order","difficulty":3,"prompt":"Соберите без ошибки","phrase":"Согласуйте множественное число.","tokens":["Она","гордится","своими","взрослыми","детьми."],"answer":"Она гордится своими взрослыми детьми.","explanation":"Гордиться кем? Своими взрослыми детьми."},
  {"id":"tv-a5","kind":"true_false","difficulty":3,"prompt":"Проверьте форму","phrase":"Творительный падеж слова «мать» — «матерью».","options":["Верно","Неверно"],"answer":"Верно","explanation":"У слова мать меняется основа: матерью."}
]$json$::jsonb
WHERE id = 'grammar_instrumental';

UPDATE grammar_topics SET
practice = practice || $json$[
  {"id":"as-p4","kind":"choice","difficulty":1,"prompt":"Выберите процесс","phrase":"Сейчас я ___ письмо.","options":["пишу","напишу","написал"],"answer":"пишу","explanation":"Действие идёт сейчас — нужен несовершенный вид."}
]$json$::jsonb,
game = game || $json$[
  {"id":"as-a1","kind":"true_false","difficulty":1,"prompt":"Верно или нет?","phrase":"«Каждый день я прочитаю газету» — нейтральная привычка.","options":["Верно","Неверно"],"answer":"Неверно","explanation":"Для привычки нужен НСВ: каждый день читаю."},
  {"id":"as-a2","kind":"order","difficulty":2,"prompt":"Соберите результат","phrase":"Действие должно быть завершено.","tokens":["Я","уже","прочитал","эту","статью."],"answer":"Я уже прочитал эту статью.","explanation":"Уже прочитал подчёркивает готовый результат."},
  {"id":"as-a3","kind":"choice","difficulty":2,"prompt":"Последовательность","phrase":"Он ___, оделся и вышел.","options":["вставал","встал","встаёт"],"answer":"встал","explanation":"Цепочка завершённых действий требует совершенного вида."},
  {"id":"as-a4","kind":"order","difficulty":3,"prompt":"Соберите контраст","phrase":"Различите процесс и результат.","tokens":["Я","долго","читал","и","наконец","прочитал","книгу."],"answer":"Я долго читал и наконец прочитал книгу.","explanation":"Читал — процесс, прочитал — результат."},
  {"id":"as-a5","kind":"true_false","difficulty":3,"prompt":"Проверьте значение","phrase":"Вопрос «Ты когда-нибудь читал Толстого?» спрашивает о факте опыта.","options":["Верно","Неверно"],"answer":"Верно","explanation":"Для факта опыта обычно используется несовершенный вид."}
]$json$::jsonb
WHERE id = 'grammar_aspect';

-- Separate the unrelated numerals and reflexive-verb material into two topics.
UPDATE grammar_topics SET
    slug = 'chislitelnye',
    title = 'Числительные',
    summary = 'Формы существительных после чисел и составные числительные',
    lesson = lesson - 2,
    practice = practice - 2,
    game = game - 3 - 2
WHERE id = 'grammar_numbers_reflexive';

UPDATE grammar_topics SET
practice = practice || $json$[
  {"id":"nr-p4","kind":"choice","difficulty":1,"prompt":"После числа","phrase":"В комнате три ___. (окно)","options":["окно","окна","окон"],"answer":"окна","explanation":"После 2–4 нужен родительный единственного: три окна."},
  {"id":"nr-p5","kind":"choice","difficulty":1,"prompt":"После числа","phrase":"На столе восемь ___. (чашка)","options":["чашка","чашки","чашек"],"answer":"чашек","explanation":"После пяти и больше нужен родительный множественного: восемь чашек."}
]$json$::jsonb,
game = game || $json$[
  {"id":"nr-a1","kind":"true_false","difficulty":1,"prompt":"Верно или нет?","phrase":"Правильно: «пять книги».","options":["Верно","Неверно"],"answer":"Неверно","explanation":"После пяти: пять книг."},
  {"id":"nr-a2","kind":"order","difficulty":2,"prompt":"Соберите количество","phrase":"Учтите последнюю часть числа.","tokens":["В","офисе","работает","двадцать","один","человек."],"answer":"В офисе работает двадцать один человек.","explanation":"21 оканчивается на один: человек стоит в единственном числе."},
  {"id":"nr-a3","kind":"choice","difficulty":2,"prompt":"Исключение 11–14","phrase":"Мы ждали двенадцать ___. (минута)","options":["минута","минуты","минут"],"answer":"минут","explanation":"12 относится к группе 11–14: двенадцать минут."},
  {"id":"nr-a4","kind":"order","difficulty":3,"prompt":"Соберите фразу","phrase":"Согласуйте составное число.","tokens":["Прошло","сто","двадцать","два","дня."],"answer":"Прошло сто двадцать два дня.","explanation":"122 оканчивается на 2, но не на 12: два дня."},
  {"id":"nr-a5","kind":"true_false","difficulty":3,"prompt":"Проверьте форму","phrase":"Правильно: «сто один студент» и «сто одиннадцать студентов».","options":["Верно","Неверно"],"answer":"Верно","explanation":"101 следует модели один, а 111 — исключительной группе 11–14."},
  {"id":"nr-a6","kind":"choice","difficulty":2,"prompt":"Составное число","phrase":"В книге двести тридцать четыре ___. (страница)","options":["страница","страницы","страниц"],"answer":"страницы","explanation":"234 оканчивается на 4, но не на 14: четыре страницы."},
  {"id":"nr-a7","kind":"choice","difficulty":3,"prompt":"Согласуйте сказуемое","phrase":"Двадцать один студент ___ тест.","options":["написал","написали","написало"],"answer":"написал","explanation":"Сочетание с один требует сказуемого в единственном числе."}
]$json$::jsonb
WHERE id = 'grammar_numbers_reflexive';

INSERT INTO grammar_topics (id, slug, title, summary, level, icon, position, status, lesson, practice, game)
VALUES (
  'grammar_reflexive', 'vozvratnye-glagoly', 'Возвратные глаголы',
  'Формы на -ся/-сь: действие на себя, взаимность и состояние', 'A2', '↔', 13, 'published',
  $json$[
    {"title":"-ся или -сь","text":"После согласной обычно пишется -ся, после гласной — -сь. Постфикс сохраняется при спряжении.","examples":["учиться → я учусь · ты учишься","улыбаться → она улыбается"],"note":"Смотрите на букву прямо перед постфиксом: учу-сь, учит-ся."},
    {"title":"Основные значения","text":"Возвратная форма может обозначать действие на себя, взаимное действие нескольких людей или состояние.","examples":["умываться · одеваться","встречаться · обниматься","бояться · смеяться"],"note":"Значение определяется всем глаголом и контекстом."},
    {"title":"Учить целиком","text":"Некоторые глаголы без -ся не употребляются или меняют значение, поэтому полезно запоминать их как отдельные слова.","examples":["бояться · надеяться · гордиться","вернуть книгу → вернуться домой"],"note":"Возвратиться и вернуть — разные действия."}
  ]$json$::jsonb,
  $json$[
    {"id":"rv-p1","kind":"choice","difficulty":1,"prompt":"Выберите постфикс","phrase":"Я учу__ в университете.","options":["ся","сь","юсь"],"answer":"сь","explanation":"После гласной у: учусь."},
    {"id":"rv-p2","kind":"choice","difficulty":1,"prompt":"Выберите форму","phrase":"Она улыба__.","options":["ется","етсь","ит"],"answer":"ется","explanation":"Улыбаться → она улыбается."},
    {"id":"rv-p3","kind":"choice","difficulty":1,"prompt":"Найдите взаимность","phrase":"Друзья часто ___.","options":["встречают","встречаются","встречает"],"answer":"встречаются","explanation":"Они встречаются друг с другом."},
    {"id":"rv-p4","kind":"choice","difficulty":1,"prompt":"Действие на себя","phrase":"Ребёнок сам ___.","options":["одевает","одевается","одеваюсь"],"answer":"одевается","explanation":"Одевается сам; одевает кого-то другого."}
  ]$json$::jsonb,
  $json$[
    {"id":"rv-g1","kind":"choice","difficulty":1,"prompt":"Выберите форму","phrase":"Мы готовим__ к экзамену.","options":["ся","сь","емся"],"answer":"ся","explanation":"Полная форма: готовимся; пропущен постфикс -ся."},
    {"id":"rv-g2","kind":"true_false","difficulty":1,"prompt":"Верно или нет?","phrase":"«Я надеюсь на хороший результат» — правильная форма.","options":["Верно","Неверно"],"answer":"Верно","explanation":"Надеяться употребляется с -ся."},
    {"id":"rv-g9","kind":"choice","difficulty":1,"prompt":"Выберите форму","phrase":"Ты часто ошиба__?","options":["ешь","ешься","ется"],"answer":"ешься","explanation":"Ошибаться → ты ошибаешься."},
    {"id":"rv-g3","kind":"order","difficulty":2,"prompt":"Соберите предложение","phrase":"Покажите взаимное действие.","tokens":["Мы","встретились","после","работы."],"answer":"Мы встретились после работы.","explanation":"Встретились — взаимное завершённое действие."},
    {"id":"rv-g4","kind":"choice","difficulty":2,"prompt":"Различите значение","phrase":"Я ___ книгу в библиотеку.","options":["вернулся","вернул","вернулась"],"answer":"вернул","explanation":"Вернуть что? Книгу. Вернуться — прийти обратно самому."},
    {"id":"rv-g5","kind":"order","difficulty":2,"prompt":"Соберите привычку","phrase":"Используйте правильную личную форму.","tokens":["Она","всегда","быстро","одевается."],"answer":"Она всегда быстро одевается.","explanation":"Она одевается — действие на себя."},
    {"id":"rv-g6","kind":"choice","difficulty":3,"prompt":"Страдательное значение","phrase":"Новый дом ___ рядом с парком.","options":["строит","строится","строятся"],"answer":"строится","explanation":"Дом строится — действие направлено на предмет."},
    {"id":"rv-g7","kind":"true_false","difficulty":3,"prompt":"Проверьте смысл","phrase":"«Мама одевает ребёнка» и «ребёнок одевается» описывают одну грамматическую модель.","options":["Верно","Неверно"],"answer":"Неверно","explanation":"В первом случае действие направлено на другого, во втором — на себя."},
    {"id":"rv-g8","kind":"order","difficulty":3,"prompt":"Соберите без ошибки","phrase":"Согласуйте глагол и управление.","tokens":["Они","гордятся","своими","результатами."],"answer":"Они гордятся своими результатами.","explanation":"Гордиться чем? Результатами; глагол сохраняет -ся."}
  ]$json$::jsonb
);
