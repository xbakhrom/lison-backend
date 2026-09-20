package httpapi

import (
	"encoding/json"
	"net/http"
	"sort"

	"github.com/go-chi/chi/v5"
)

type topicListItem struct {
	ID         string `json:"id"`
	Slug       string `json:"slug"`
	Title      string `json:"title"`
	Summary    string `json:"summary"`
	Level      string `json:"level"`
	WordCount  int    `json:"wordCount"`
	AddedCount int    `json:"addedCount"`
	DueCount   int    `json:"dueCount"`
}

type vocabularyItem struct {
	ID      string `json:"id"`
	Russian string `json:"russian"`
	Uzbek   string `json:"uzbek"`
	Added   bool   `json:"added"`
}

type vocabularyCategory struct {
	ID    string           `json:"id"`
	Title string           `json:"title"`
	Items []vocabularyItem `json:"items"`
}

type formField struct {
	ID           string          `json:"id"`
	Label        string          `json:"label"`
	InputType    string          `json:"inputType"`
	InitialValue json.RawMessage `json:"initialValue"`
	Value        json.RawMessage `json:"value"`
}

type formSection struct {
	ID     string      `json:"id"`
	Title  string      `json:"title"`
	Fields []formField `json:"fields"`
}

type checklistItem struct {
	ID      string `json:"id"`
	Prompt  string `json:"prompt"`
	Hint    string `json:"hint"`
	Checked bool   `json:"checked"`
}

type checklistGroup struct {
	Title string          `json:"title"`
	Items []checklistItem `json:"items"`
}

func (s *Server) listTopics(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r.Context())
	rows, err := s.db.Query(r.Context(), `
		SELECT t.id, t.slug, t.title, t.summary, t.level,
		       count(DISTINCT v.id)::int,
		       count(DISTINCT uc.id)::int,
		       count(DISTINCT uc.id) FILTER (WHERE uc.due_date <= $2)::int
		FROM topics t
		LEFT JOIN vocabulary_items v ON v.topic_id = t.id AND v.active
		LEFT JOIN user_cards uc ON uc.vocabulary_item_id = v.id AND uc.user_id = $1
		WHERE t.status = 'published'
		GROUP BY t.id
		ORDER BY t.position, t.title`, user.ID, localToday(r))
	if err != nil {
		s.logger.Error("list topics", "error", err)
		writeError(w, http.StatusInternalServerError, "database_error", "Не удалось загрузить темы.")
		return
	}
	defer rows.Close()
	items := make([]topicListItem, 0)
	for rows.Next() {
		var item topicListItem
		if err := rows.Scan(&item.ID, &item.Slug, &item.Title, &item.Summary, &item.Level, &item.WordCount, &item.AddedCount, &item.DueCount); err != nil {
			s.logger.Error("scan topic", "error", err)
			writeError(w, http.StatusInternalServerError, "database_error", "Не удалось загрузить темы.")
			return
		}
		items = append(items, item)
	}
	writeJSON(w, http.StatusOK, map[string]any{"topics": items})
}

func (s *Server) getTopic(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r.Context())
	slug := chi.URLParam(r, "slug")
	var topic struct {
		ID, Slug, Title, Summary, Level, ContentMarkdown string
	}
	err := s.db.QueryRow(r.Context(), `
		SELECT id, slug, title, summary, level, content_markdown
		FROM topics WHERE slug = $1 AND status = 'published'`, slug,
	).Scan(&topic.ID, &topic.Slug, &topic.Title, &topic.Summary, &topic.Level, &topic.ContentMarkdown)
	if err != nil {
		writeError(w, http.StatusNotFound, "topic_not_found", "Тема не найдена.")
		return
	}

	categories, err := s.topicVocabulary(r, topic.ID, user.ID)
	if err != nil {
		s.logger.Error("load vocabulary", "error", err)
		writeError(w, http.StatusInternalServerError, "database_error", "Не удалось загрузить слова.")
		return
	}
	forms, err := s.topicForms(r, topic.ID, user.ID)
	if err != nil {
		s.logger.Error("load forms", "error", err)
		writeError(w, http.StatusInternalServerError, "database_error", "Не удалось загрузить ответы.")
		return
	}
	checklist, err := s.topicChecklist(r, topic.ID, user.ID)
	if err != nil {
		s.logger.Error("load checklist", "error", err)
		writeError(w, http.StatusInternalServerError, "database_error", "Не удалось загрузить вопросы.")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"id": topic.ID, "slug": topic.Slug, "title": topic.Title,
		"summary": topic.Summary, "level": topic.Level,
		"contentMarkdown": topic.ContentMarkdown,
		"vocabulary":      categories, "forms": forms, "checklist": checklist,
	})
}

func (s *Server) topicVocabulary(r *http.Request, topicID string, userID int64) ([]vocabularyCategory, error) {
	rows, err := s.db.Query(r.Context(), `
		SELECT c.id, c.title, v.id, v.russian, v.uzbek, uc.id IS NOT NULL
		FROM vocabulary_categories c
		JOIN vocabulary_items v ON v.category_id = c.id AND v.active
		LEFT JOIN user_cards uc ON uc.vocabulary_item_id = v.id AND uc.user_id = $2
		WHERE c.topic_id = $1
		ORDER BY c.position, v.position`, topicID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	byID := map[string]*vocabularyCategory{}
	order := make([]string, 0)
	for rows.Next() {
		var categoryID, title string
		var item vocabularyItem
		if err := rows.Scan(&categoryID, &title, &item.ID, &item.Russian, &item.Uzbek, &item.Added); err != nil {
			return nil, err
		}
		category := byID[categoryID]
		if category == nil {
			category = &vocabularyCategory{ID: categoryID, Title: title, Items: []vocabularyItem{}}
			byID[categoryID] = category
			order = append(order, categoryID)
		}
		category.Items = append(category.Items, item)
	}
	result := make([]vocabularyCategory, 0, len(order))
	for _, id := range order {
		result = append(result, *byID[id])
	}
	return result, rows.Err()
}

func (s *Server) topicForms(r *http.Request, topicID string, userID int64) ([]formSection, error) {
	rows, err := s.db.Query(r.Context(), `
		SELECT f.section_id, f.section_title, f.id, f.label, f.input_type,
		       f.initial_value, COALESCE(a.value, f.initial_value)
		FROM topic_form_fields f
		LEFT JOIN user_topic_answers a ON a.field_id = f.id AND a.user_id = $2
		WHERE f.topic_id = $1
		ORDER BY CASE f.section_id
		    WHEN 'my_city' THEN 1 WHEN 'my_stories' THEN 2
		    WHEN 'my_theses' THEN 3 ELSE 4 END, f.position`, topicID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	sections := map[string]*formSection{}
	order := make([]string, 0)
	for rows.Next() {
		var sectionID, sectionTitle string
		var field formField
		if err := rows.Scan(&sectionID, &sectionTitle, &field.ID, &field.Label, &field.InputType, &field.InitialValue, &field.Value); err != nil {
			return nil, err
		}
		section := sections[sectionID]
		if section == nil {
			section = &formSection{ID: sectionID, Title: sectionTitle, Fields: []formField{}}
			sections[sectionID] = section
			order = append(order, sectionID)
		}
		section.Fields = append(section.Fields, field)
	}
	result := make([]formSection, 0, len(order))
	for _, id := range order {
		result = append(result, *sections[id])
	}
	return result, rows.Err()
}

func (s *Server) topicChecklist(r *http.Request, topicID string, userID int64) ([]checklistGroup, error) {
	rows, err := s.db.Query(r.Context(), `
		SELECT i.group_title, i.id, i.prompt, i.hint, COALESCE(p.checked, false)
		FROM topic_checklist_items i
		LEFT JOIN user_checklist_progress p ON p.item_id = i.id AND p.user_id = $2
		WHERE i.topic_id = $1 ORDER BY i.position`, topicID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	groups := map[string]*checklistGroup{}
	order := make([]string, 0)
	for rows.Next() {
		var title string
		var item checklistItem
		if err := rows.Scan(&title, &item.ID, &item.Prompt, &item.Hint, &item.Checked); err != nil {
			return nil, err
		}
		group := groups[title]
		if group == nil {
			group = &checklistGroup{Title: title, Items: []checklistItem{}}
			groups[title] = group
			order = append(order, title)
		}
		group.Items = append(group.Items, item)
	}
	result := make([]checklistGroup, 0, len(order))
	for _, title := range order {
		result = append(result, *groups[title])
	}
	return result, rows.Err()
}

func sortedKeys[T any](values map[string]T) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
