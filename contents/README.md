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
