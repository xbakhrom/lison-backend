CREATE TABLE users (
    telegram_id bigint PRIMARY KEY,
    first_name text NOT NULL DEFAULT '',
    username text NOT NULL DEFAULT '',
    language_code text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE topics (
    id text PRIMARY KEY,
    slug text NOT NULL UNIQUE,
    title text NOT NULL,
    summary text NOT NULL DEFAULT '',
    level text NOT NULL DEFAULT '',
    content_markdown text NOT NULL DEFAULT '',
    cover text NOT NULL DEFAULT '',
    position integer NOT NULL DEFAULT 0,
    status text NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'published', 'archived')),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE vocabulary_categories (
    id text PRIMARY KEY,
    topic_id text NOT NULL REFERENCES topics(id) ON DELETE CASCADE,
    title text NOT NULL,
    position integer NOT NULL DEFAULT 0
);

CREATE TABLE vocabulary_items (
    id text PRIMARY KEY,
    category_id text NOT NULL REFERENCES vocabulary_categories(id) ON DELETE CASCADE,
    topic_id text NOT NULL REFERENCES topics(id) ON DELETE CASCADE,
    russian text NOT NULL,
    uzbek text NOT NULL,
    position integer NOT NULL DEFAULT 0,
    active boolean NOT NULL DEFAULT true
);

CREATE TABLE topic_form_fields (
    id text PRIMARY KEY,
    topic_id text NOT NULL REFERENCES topics(id) ON DELETE CASCADE,
    section_id text NOT NULL,
    section_title text NOT NULL,
    label text NOT NULL,
    input_type text NOT NULL CHECK (input_type IN ('text', 'textarea', 'collection')),
    initial_value jsonb NOT NULL DEFAULT '""'::jsonb,
    position integer NOT NULL DEFAULT 0
);

CREATE TABLE topic_checklist_items (
    id text PRIMARY KEY,
    topic_id text NOT NULL REFERENCES topics(id) ON DELETE CASCADE,
    group_title text NOT NULL,
    prompt text NOT NULL,
    hint text NOT NULL DEFAULT '',
    position integer NOT NULL DEFAULT 0
);

CREATE TABLE user_cards (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id bigint NOT NULL REFERENCES users(telegram_id) ON DELETE CASCADE,
    vocabulary_item_id text NOT NULL REFERENCES vocabulary_items(id) ON DELETE RESTRICT,
    state text NOT NULL DEFAULT 'new' CHECK (state IN ('new', 'review', 'relearning')),
    interval_days integer NOT NULL DEFAULT 0,
    ease_factor numeric(4,2) NOT NULL DEFAULT 2.50,
    due_date date NOT NULL,
    repetitions integer NOT NULL DEFAULT 0,
    lapses integer NOT NULL DEFAULT 0,
    last_reviewed_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (user_id, vocabulary_item_id)
);

CREATE INDEX user_cards_due_idx ON user_cards(user_id, due_date);

CREATE TABLE review_logs (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id bigint NOT NULL REFERENCES users(telegram_id) ON DELETE CASCADE,
    card_id bigint NOT NULL REFERENCES user_cards(id) ON DELETE CASCADE,
    rating text NOT NULL CHECK (rating IN ('again', 'hard', 'good', 'easy')),
    previous_interval integer NOT NULL,
    next_interval integer NOT NULL,
    reviewed_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE user_topic_answers (
    user_id bigint NOT NULL REFERENCES users(telegram_id) ON DELETE CASCADE,
    topic_id text NOT NULL REFERENCES topics(id) ON DELETE CASCADE,
    field_id text NOT NULL REFERENCES topic_form_fields(id) ON DELETE CASCADE,
    value jsonb NOT NULL,
    updated_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, field_id)
);

CREATE TABLE user_checklist_progress (
    user_id bigint NOT NULL REFERENCES users(telegram_id) ON DELETE CASCADE,
    item_id text NOT NULL REFERENCES topic_checklist_items(id) ON DELETE CASCADE,
    checked boolean NOT NULL DEFAULT false,
    updated_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, item_id)
);

CREATE TABLE reminder_settings (
    user_id bigint PRIMARY KEY REFERENCES users(telegram_id) ON DELETE CASCADE,
    enabled boolean NOT NULL DEFAULT false,
    reminder_time time NOT NULL DEFAULT '20:00',
    timezone text NOT NULL DEFAULT 'Asia/Tashkent',
    last_sent_on date,
    updated_at timestamptz NOT NULL DEFAULT now()
);

INSERT INTO topics (id, slug, title, summary, level, status, position, content_markdown)
VALUES (
    'topic_gorod',
    'gorod',
    'Город',
    'Город, районы, транспорт и инфраструктура',
    'A2–B1',
    'published',
    1,
$md$
## О чём эта тема

- **Мой город** — где находится, насколько большой, сколько жителей.
- **Районы** — центр и окраина, где я живу.
- **Транспорт** — метро, автобус, пробки, жизнь без машины.
- **Что нравится и не нравится** — сильные и слабые стороны города.
- **Как город изменился** — что изменилось за последние десять лет.
- **Город или деревня** — где лучше жить.
- **Идеальный город** — каким он должен быть.
- **Переезд** — хочется ли переехать в другой город.

## Фразы и конструкции

### Где и какой

- `Я роди́лся в..., но живу́ в...`
- `Мой го́род нахо́дится на ю́ге страны́.`
- `Э́то небольшо́й го́род, там живёт о́коло ста ты́сяч челове́к.`
- `Я живу́ в э́том го́роде уже́ два́дцать лет.`

### Описание

- `В го́роде есть метро́, но нет трамва́я.`
- `В це́нтре шу́мно, а на окра́ине ти́хо.`
- `У нас мно́го парко́в, но ма́ло велодоро́жек.`
- `Э́то го́род, где всё ря́дом.`

### Изменения

- `Го́род си́льно измени́лся за после́дние де́сять лет.`
- `Ра́ньше здесь бы́ли ста́рые дома́, а сейча́с — новостро́йки.`
- `Стро́ят мно́го, но не всегда́ уда́чно.`
- `Го́род стал совреме́ннее, но потеря́л свой хара́ктер.`

### Нравится и не нравится

- `Что мне нра́вится, так э́то...`
- `Что меня́ раздража́ет, так э́то про́бки.`
- `Мне не хвата́ет зе́лени и тишины́.`
- `Я привы́к к шу́му — уже́ не замеча́ю.`

## Выражения

- **В гостя́х хорошо́, а до́ма лу́чше.** — Mehmonda yaxshi, uyda undan yaxshi.
- **Москва́ не сра́зу стро́илась.** — Rim bir kunda qurilmagan.
$md$
);

INSERT INTO vocabulary_categories (id, topic_id, title, position) VALUES
('city_structure', 'topic_gorod', 'Устройство города', 1),
('buildings', 'topic_gorod', 'Здания', 2),
('infrastructure', 'topic_gorod', 'Инфраструктура', 3),
('city_adjectives', 'topic_gorod', 'Какой город?', 4),
('city_verbs', 'topic_gorod', 'Глаголы', 5);

INSERT INTO vocabulary_items (id, category_id, topic_id, russian, uzbek, position) VALUES
('stolitsa', 'city_structure', 'topic_gorod', 'Столи́ца', 'poytaxt', 1),
('rayon', 'city_structure', 'topic_gorod', 'Райо́н', 'tuman', 2),
('okraina', 'city_structure', 'topic_gorod', 'Окра́ина', 'shahar cheti', 3),
('prigorod', 'city_structure', 'topic_gorod', 'При́город', 'shahar atrofi', 4),
('kvartal', 'city_structure', 'topic_gorod', 'Кварта́л', 'kvartal', 5),
('mikrorayon', 'city_structure', 'topic_gorod', 'Микрорайо́н', 'mikrorayon', 6),
('dvor', 'city_structure', 'topic_gorod', 'Двор', 'hovli', 7),
('naselenie', 'city_structure', 'topic_gorod', 'Населе́ние', 'aholi', 8),
('zhitel', 'city_structure', 'topic_gorod', 'Жи́тель', 'yashovchi', 9),
('zdanie', 'buildings', 'topic_gorod', 'Зда́ние', 'bino', 1),
('novostroyka', 'buildings', 'topic_gorod', 'Новостро́йка', 'yangi qurilgan uy', 2),
('mnogoetazhka', 'buildings', 'topic_gorod', 'Многоэта́жка', 'ko‘p qavatli uy', 3),
('etazh', 'buildings', 'topic_gorod', 'Эта́ж', 'qavat', 4),
('neboskreb', 'buildings', 'topic_gorod', 'Небоскрёб', 'osmono‘par', 5),
('podezd', 'buildings', 'topic_gorod', 'Подъе́зд', 'podyezd', 6),
('zabroshennyy', 'buildings', 'topic_gorod', 'Забро́шенный', 'tashlandiq', 7),
('ostanovka', 'infrastructure', 'topic_gorod', 'Остано́вка', 'bekat', 1),
('stantsiya', 'infrastructure', 'topic_gorod', 'Ста́нция', 'stansiya', 2),
('most', 'infrastructure', 'topic_gorod', 'Мост', 'ko‘prik', 3),
('podzemnyy_perekhod', 'infrastructure', 'topic_gorod', 'Подзе́мный перехо́д', 'yer osti o‘tish joyi', 4),
('probka', 'infrastructure', 'topic_gorod', 'Про́бка', 'tirbandlik', 5),
('obshchestvennyy_transport', 'infrastructure', 'topic_gorod', 'Обще́ственный тра́нспорт', 'jamoat transporti', 6),
('rynok', 'infrastructure', 'topic_gorod', 'Ры́нок', 'bozor', 7),
('torgovyy_tsentr', 'infrastructure', 'topic_gorod', 'Торго́вый центр', 'savdo markazi', 8),
('bolnitsa', 'infrastructure', 'topic_gorod', 'Больни́ца', 'shifoxona', 9),
('biblioteka', 'infrastructure', 'topic_gorod', 'Библиоте́ка', 'kutubxona', 10),
('muzey', 'infrastructure', 'topic_gorod', 'Музе́й', 'muzey', 11),
('skver', 'infrastructure', 'topic_gorod', 'Сквер', 'kichik bog‘', 12),
('naberezhnaya', 'infrastructure', 'topic_gorod', 'На́бережная', 'qirg‘oq bo‘yi', 13),
('uyutnyy', 'city_adjectives', 'topic_gorod', 'Ую́тный', 'shinam', 1),
('shumnyy', 'city_adjectives', 'topic_gorod', 'Шу́мный', 'shovqinli', 2),
('tikhiy', 'city_adjectives', 'topic_gorod', 'Ти́хий', 'tinch', 3),
('gryaznyy', 'city_adjectives', 'topic_gorod', 'Гря́зный', 'iflos', 4),
('chistyy', 'city_adjectives', 'topic_gorod', 'Чи́стый', 'toza', 5),
('sovremennyy', 'city_adjectives', 'topic_gorod', 'Совреме́нный', 'zamonaviy', 6),
('starinnyy', 'city_adjectives', 'topic_gorod', 'Стари́нный', 'qadimiy', 7),
('udobnyy', 'city_adjectives', 'topic_gorod', 'Удо́бный', 'qulay', 8),
('prostornyy', 'city_adjectives', 'topic_gorod', 'Просто́рный', 'keng', 9),
('tesnyy', 'city_adjectives', 'topic_gorod', 'Те́сный', 'tor', 10),
('zelenyy', 'city_adjectives', 'topic_gorod', 'Зелёный', 'ko‘kalamzor', 11),
('stroit', 'city_verbs', 'topic_gorod', 'Стро́ить / постро́ить', 'qurmoq', 1),
('snosit', 'city_verbs', 'topic_gorod', 'Сноси́ть / снести́', 'buzmoq', 2),
('pereezhat', 'city_verbs', 'topic_gorod', 'Переезжа́ть / перее́хать', 'ko‘chib o‘tmoq', 3),
('razvivatsya', 'city_verbs', 'topic_gorod', 'Развива́ться', 'rivojlanmoq', 4),
('menyatsya', 'city_verbs', 'topic_gorod', 'Меня́ться', 'o‘zgarmoq', 5),
('skuchat', 'city_verbs', 'topic_gorod', 'Скуча́ть (по чему)', 'sog‘inmoq', 6),
('dobiratsya', 'city_verbs', 'topic_gorod', 'Добира́ться (до чего)', 'yetib bormoq', 7);

INSERT INTO topic_form_fields (id, topic_id, section_id, section_title, label, input_type, initial_value, position) VALUES
('city_district', 'topic_gorod', 'my_city', 'Про мой город', 'Город / район', 'text', '""', 1),
('years_here', 'topic_gorod', 'my_city', 'Про мой город', 'Сколько живу здесь', 'text', '""', 2),
('population', 'topic_gorod', 'my_city', 'Про мой город', 'Население', 'text', '""', 3),
('commute', 'topic_gorod', 'my_city', 'Про мой город', 'Как добираюсь до работы', 'textarea', '""', 4),
('favorite_place', 'topic_gorod', 'my_city', 'Про мой город', 'Любимое место', 'textarea', '""', 5),
('show_guest', 'topic_gorod', 'my_city', 'Про мой город', 'Что бы показал гостю', 'textarea', '""', 6),
('change_city', 'topic_gorod', 'my_city', 'Про мой город', 'Что бы изменил', 'textarea', '""', 7),
('story_1_title', 'topic_gorod', 'my_stories', 'Мои истории', 'История 1 — название', 'text', '""', 1),
('story_1_context', 'topic_gorod', 'my_stories', 'Мои истории', 'Кто / где / когда', 'textarea', '""', 2),
('story_1_event', 'topic_gorod', 'my_stories', 'Мои истории', 'Что произошло', 'textarea', '""', 3),
('story_1_reason', 'topic_gorod', 'my_stories', 'Мои истории', 'Почему я это запомнил', 'textarea', '""', 4),
('story_1_questions', 'topic_gorod', 'my_stories', 'Мои истории', 'Для каких вопросов подходит', 'textarea', '""', 5),
('story_2_title', 'topic_gorod', 'my_stories', 'Мои истории', 'История 2 — название', 'text', '""', 6),
('story_2_context', 'topic_gorod', 'my_stories', 'Мои истории', 'Кто / где / когда', 'textarea', '""', 7),
('story_2_event', 'topic_gorod', 'my_stories', 'Мои истории', 'Что произошло', 'textarea', '""', 8),
('story_2_reason', 'topic_gorod', 'my_stories', 'Мои истории', 'Почему я это запомнил', 'textarea', '""', 9),
('story_2_questions', 'topic_gorod', 'my_stories', 'Мои истории', 'Для каких вопросов подходит', 'textarea', '""', 10),
('story_3_title', 'topic_gorod', 'my_stories', 'Мои истории', 'История 3 — название', 'text', '""', 11),
('story_3_context', 'topic_gorod', 'my_stories', 'Мои истории', 'Кто / где / когда', 'textarea', '""', 12),
('story_3_event', 'topic_gorod', 'my_stories', 'Мои истории', 'Что произошло', 'textarea', '""', 13),
('story_3_reason', 'topic_gorod', 'my_stories', 'Мои истории', 'Почему я это запомнил', 'textarea', '""', 14),
('story_3_questions', 'topic_gorod', 'my_stories', 'Мои истории', 'Для каких вопросов подходит', 'textarea', '""', 15),
('thesis_1', 'topic_gorod', 'my_theses', 'Мои тезисы', 'Тезис 1', 'textarea', '"Город — это не здания, а люди."', 1),
('thesis_2', 'topic_gorod', 'my_theses', 'Мои тезисы', 'Тезис 2', 'textarea', '"Главное в городе — не красота, а удобство."', 2),
('thesis_3', 'topic_gorod', 'my_theses', 'Мои тезисы', 'Тезис 3', 'textarea', '"Мой город стал современнее, но потерял свой характер."', 3),
('thesis_4', 'topic_gorod', 'my_theses', 'Мои тезисы', 'Тезис 4', 'textarea', '"Идеальный город — это где можно дойти пешком до всего, что нужно."', 4),
('thesis_5', 'topic_gorod', 'my_theses', 'Мои тезисы', 'Тезис 5', 'textarea', '"Нельзя строить новое, уничтожая старое."', 5),
('thesis_6', 'topic_gorod', 'my_theses', 'Мои тезисы', 'Тезис 6', 'textarea', '"Человек привыкает к любому городу, вопрос только во времени."', 6),
('corrected_errors', 'topic_gorod', 'after_lesson', 'После урока', 'Исправленные ошибки', 'collection', '[]', 1),
('missing_words', 'topic_gorod', 'after_lesson', 'После урока', 'Слова, которых мне не хватило', 'textarea', '""', 2),
('unexpected_question', 'topic_gorod', 'after_lesson', 'После урока', 'Неожиданный вопрос', 'textarea', '""', 3),
('next_time', 'topic_gorod', 'after_lesson', 'После урока', 'В следующий раз', 'textarea', '""', 4);

INSERT INTO topic_checklist_items (id, topic_id, group_title, prompt, hint, position) VALUES
('q_describe_city', 'topic_gorod', 'Конкретные', 'Расскажите о городе, где вы живёте.', '', 1),
('q_district', 'topic_gorod', 'Конкретные', 'В каком районе вы живёте? Вам там нравится?', '', 2),
('q_commute', 'topic_gorod', 'Конкретные', 'Как вы добираетесь до работы?', '', 3),
('q_favorite_place', 'topic_gorod', 'Конкретные', 'Какое ваше любимое место в городе?', '', 4),
('q_weekend_walk', 'topic_gorod', 'Конкретные', 'Где вы обычно гуляете в выходные?', '', 5),
('q_show_guest', 'topic_gorod', 'Конкретные', 'Что бы вы показали гостю в первую очередь?', '', 6),
('q_likes', 'topic_gorod', 'Средние', 'Что вам нравится в вашем городе, а что нет?', 'Две стороны', 7),
('q_changed', 'topic_gorod', 'Средние', 'Как ваш город изменился за последние 10 лет?', 'Линия времени', 8),
('q_move', 'topic_gorod', 'Средние', 'Вы хотели бы переехать в другой город?', '', 9),
('q_center', 'topic_gorod', 'Средние', 'Где лучше жить — в центре или на окраине?', '', 10),
('q_without_car', 'topic_gorod', 'Средние', 'Легко ли в вашем городе жить без машины?', '', 11),
('q_difference', 'topic_gorod', 'Средние', 'Чем ваш город отличается от других?', '', 12),
('q_good_city', 'topic_gorod', 'Абстрактные', 'Что делает город хорошим для жизни?', 'Три причины', 13),
('q_city_or_village', 'topic_gorod', 'Абстрактные', 'Город или деревня — где лучше жить?', '', 14),
('q_ideal_city', 'topic_gorod', 'Абстрактные', 'Каким должен быть идеальный город?', '', 15),
('q_character', 'topic_gorod', 'Абстрактные', 'Влияет ли город на характер человека?', '', 16),
('q_old_buildings', 'topic_gorod', 'Абстрактные', 'Нужно ли сносить старые здания ради новых?', '', 17),
('q_love_city', 'topic_gorod', 'Абстрактные', 'Можно ли полюбить город, в котором ты не родился?', '', 18),
('q_hometown', 'topic_gorod', 'Абстрактные', 'Что для вас «родной город»?', '', 19);
