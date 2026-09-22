# Lison content format

Topics are authored as UTF-8 Markdown and live under `contents/topics/<slug>/topic.md`.

Each topic must contain YAML front matter with these required fields:

```yaml
schema_version: 1
id: topic_gorod
slug: gorod
title: Город
summary: Город, районы, транспорт и инфраструктура
level: A2-B1
learning_language: ru
translation_language: uz
status: published
order: 1
```

Vocabulary uses the following syntax inside the `Новые слова` section:

```markdown
Столи́ца :: poytaxt <!-- lison:id=stolitsa -->
```

Stable IDs must contain only lowercase Latin letters, digits and underscores. Never change a published topic, vocabulary, form-field or checklist ID: user progress refers to it.

Editable sections and fields use invisible HTML comments:

```markdown
## Про мой город <!-- lison:section=my_city type=form -->

| Вопрос | Ответ |
| --- | --- |
| Город / район <!-- lison:field=city_district type=text --> | |
```

Supported field types are `text`, `textarea` and `collection`. Checklists use normal Markdown task items followed by `<!-- lison:id=... -->`.

The first implementation seeds the same structured data through the PostgreSQL migration. A content importer will make these Markdown files the only publishing source in the next iteration.

Run `make content-check` from the repository root before publishing content. It checks metadata, stable IDs, duplicate IDs, vocabulary syntax and unsafe script markup.

## Grammar

Grammar lives in `contents/grammar/<position>-<slug>.json`, one file per topic.
These files are the only source of truth: the API embeds them and rewrites
`grammar_topics` on every start, so an edit here ships with the next deploy.

```json
{
  "id": "grammar_prepositional",
  "slug": "predlozhnyy-padezh",
  "title": "Предложный падеж",
  "summary": "Где? О чём? — место действия и тема разговора",
  "level": "A1",
  "icon": "◎",
  "stage": "cases",
  "position": 11,
  "status": "published",
  "lesson": [{ "title": "…", "text": "…", "examples": ["…"], "note": "…" }],
  "practice": [{ "id": "pp-p01", "kind": "choice", "difficulty": 1, "prompt": "Где?",
                 "phrase": "Книга лежит ___. (стол)", "options": ["на стол", "на столе", "на стола"],
                 "answer": "на столе", "explanation": "Где? На столе — предложный падеж." }],
  "game": []
}
```

Rules the linter enforces:

- `stage` is one of `foundation`, `verbs`, `cases`, `fluency`, and `position` is
  unique across the route. The learner walks the stages in that order.
- Each bank holds at least 60 questions with at least 20 per difficulty level
  (1 easy, 2 confident, 3 hard — mixed topics and exceptions).
- `kind` is `choice` (3–4 options), `true_false` (exactly 2) or `order`. The
  game bank takes only tappable kinds: it runs on a countdown, so sentence
  building stays in the warm-up.
- `answer` must be one of `options`; for `order` questions it must equal
  `tokens` joined by single spaces.
- Every question needs a prompt, a phrase and an explanation.

Never change a published topic `id`: `user_grammar_progress` refers to it. A
topic dropped from this directory is archived rather than deleted, so progress
rows survive. Question ids may change freely — nothing references them.

Run `make content-check` to validate the bank, or `go test ./internal/grammar/`
for the same checks inside the test suite.
