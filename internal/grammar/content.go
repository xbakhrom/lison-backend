package grammar

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"path"
	"regexp"
	"sort"
	"strings"

	"github.com/xbakhrom/lison-backend/contents"
)

// Every published topic ships the same sized bank so a learner always meets a
// full ladder: BankSize questions per stage, evenly split over the three
// difficulty levels.
const (
	BankSize          = 60
	QuestionsPerLevel = BankSize / MaxDifficulty
)

// Stage names the curriculum block a topic belongs to. The learner walks them
// in order: foundation, verbs, cases, then fluency.
const (
	StageFoundation = "foundation"
	StageVerbs      = "verbs"
	StageCases      = "cases"
	StageFluency    = "fluency"
)

var stages = map[string]bool{
	StageFoundation: true,
	StageVerbs:      true,
	StageCases:      true,
	StageFluency:    true,
}

var kinds = map[string]bool{"choice": true, "true_false": true, "order": true}

var (
	topicIDPattern = regexp.MustCompile(`^grammar_[a-z0-9_]+$`)
	slugPattern    = regexp.MustCompile(`^[a-z0-9-]+$`)
)

// LessonBlock is one explanation card shown before the exercises.
type LessonBlock struct {
	Title    string   `json:"title"`
	Text     string   `json:"text"`
	Examples []string `json:"examples"`
	Note     string   `json:"note"`
}

// Question is one exercise from either the practice or the game bank.
type Question struct {
	ID          string   `json:"id"`
	Kind        string   `json:"kind"`
	Difficulty  int      `json:"difficulty"`
	Prompt      string   `json:"prompt"`
	Phrase      string   `json:"phrase"`
	Options     []string `json:"options,omitempty"`
	Tokens      []string `json:"tokens,omitempty"`
	Answer      string   `json:"answer"`
	Explanation string   `json:"explanation"`
}

// Topic is the authored source of one grammar topic.
type Topic struct {
	ID       string        `json:"id"`
	Slug     string        `json:"slug"`
	Title    string        `json:"title"`
	Summary  string        `json:"summary"`
	Level    string        `json:"level"`
	Icon     string        `json:"icon"`
	Stage    string        `json:"stage"`
	Position int           `json:"position"`
	Status   string        `json:"status"`
	Lesson   []LessonBlock `json:"lesson"`
	Practice []Question    `json:"practice"`
	Game     []Question    `json:"game"`
}

// LoadTopics reads every embedded grammar topic ordered by curriculum position.
func LoadTopics() ([]Topic, error) {
	return loadTopics(contents.Grammar, contents.GrammarDir)
}

func loadTopics(files fs.FS, dir string) ([]Topic, error) {
	entries, err := fs.ReadDir(files, dir)
	if err != nil {
		return nil, fmt.Errorf("read grammar content: %w", err)
	}

	topics := make([]Topic, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		body, err := fs.ReadFile(files, path.Join(dir, entry.Name()))
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", entry.Name(), err)
		}
		var topic Topic
		decoder := json.NewDecoder(strings.NewReader(string(body)))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&topic); err != nil {
			return nil, fmt.Errorf("decode %s: %w", entry.Name(), err)
		}
		topics = append(topics, topic)
	}
	sort.Slice(topics, func(i, j int) bool { return topics[i].Position < topics[j].Position })
	return topics, nil
}

// Validate reports every content problem found in the loaded topics. Returning
// all of them at once keeps the authoring loop short.
func Validate(topics []Topic) []string {
	issues := make([]string, 0)
	if len(topics) == 0 {
		return append(issues, "no grammar topics found")
	}

	seenID := map[string]string{}
	seenSlug := map[string]string{}
	seenPosition := map[int]string{}
	for _, topic := range topics {
		label := topic.ID
		if label == "" {
			label = topic.Slug
		}
		if !topicIDPattern.MatchString(topic.ID) {
			issues = append(issues, fmt.Sprintf("%s: id must match grammar_<lower_snake_case>", label))
		}
		if !slugPattern.MatchString(topic.Slug) {
			issues = append(issues, fmt.Sprintf("%s: slug must contain lowercase Latin letters, digits or hyphens", label))
		}
		if previous, exists := seenID[topic.ID]; exists {
			issues = append(issues, fmt.Sprintf("%s: duplicate id, already used by %s", label, previous))
		}
		seenID[topic.ID] = label
		if previous, exists := seenSlug[topic.Slug]; exists {
			issues = append(issues, fmt.Sprintf("%s: duplicate slug, already used by %s", label, previous))
		}
		seenSlug[topic.Slug] = label
		if previous, exists := seenPosition[topic.Position]; exists {
			issues = append(issues, fmt.Sprintf("%s: duplicate position %d, already used by %s", label, topic.Position, previous))
		}
		seenPosition[topic.Position] = label

		if topic.Position < 1 {
			issues = append(issues, fmt.Sprintf("%s: position must be 1 or greater", label))
		}
		for field, value := range map[string]string{"title": topic.Title, "summary": topic.Summary, "level": topic.Level, "icon": topic.Icon} {
			if strings.TrimSpace(value) == "" {
				issues = append(issues, fmt.Sprintf("%s: %s is required", label, field))
			}
		}
		if !stages[topic.Stage] {
			issues = append(issues, fmt.Sprintf("%s: unknown stage %q", label, topic.Stage))
		}
		if topic.Status != "published" && topic.Status != "draft" && topic.Status != "archived" {
			issues = append(issues, fmt.Sprintf("%s: unknown status %q", label, topic.Status))
		}
		if len(topic.Lesson) < 3 {
			issues = append(issues, fmt.Sprintf("%s: needs at least 3 lesson blocks, found %d", label, len(topic.Lesson)))
		}
		for index, block := range topic.Lesson {
			if strings.TrimSpace(block.Title) == "" || strings.TrimSpace(block.Text) == "" {
				issues = append(issues, fmt.Sprintf("%s: lesson block %d needs a title and text", label, index+1))
			}
			if len(block.Examples) == 0 {
				issues = append(issues, fmt.Sprintf("%s: lesson block %q needs at least one example", label, block.Title))
			}
		}

		issues = append(issues, validateBank(label, "practice", topic.Practice, true)...)
		issues = append(issues, validateBank(label, "game", topic.Game, false)...)
	}
	return issues
}

// validateBank checks one question bank. Game banks stay Kahoot shaped: only
// tappable option questions, so a countdown never blocks on sentence building.
func validateBank(label, bank string, questions []Question, allowOrder bool) []string {
	issues := make([]string, 0)
	if len(questions) < BankSize {
		issues = append(issues, fmt.Sprintf("%s: %s bank needs %d questions, found %d", label, bank, BankSize, len(questions)))
	}

	perLevel := map[int]int{}
	seen := map[string]bool{}
	for _, question := range questions {
		name := fmt.Sprintf("%s %s %s", label, bank, question.ID)
		if question.ID == "" {
			issues = append(issues, fmt.Sprintf("%s: %s bank has a question without an id", label, bank))
			continue
		}
		if seen[question.ID] {
			issues = append(issues, fmt.Sprintf("%s: duplicate question id", name))
		}
		seen[question.ID] = true
		if !kinds[question.Kind] {
			issues = append(issues, fmt.Sprintf("%s: unknown kind %q", name, question.Kind))
		}
		if question.Kind == "order" && !allowOrder {
			issues = append(issues, fmt.Sprintf("%s: the game bank only accepts choice and true_false questions", name))
		}
		if question.Difficulty < MinDifficulty || question.Difficulty > MaxDifficulty {
			issues = append(issues, fmt.Sprintf("%s: difficulty must be between %d and %d", name, MinDifficulty, MaxDifficulty))
		} else {
			perLevel[question.Difficulty]++
		}
		if strings.TrimSpace(question.Prompt) == "" || strings.TrimSpace(question.Phrase) == "" {
			issues = append(issues, fmt.Sprintf("%s: prompt and phrase are required", name))
		}
		if strings.TrimSpace(question.Explanation) == "" {
			issues = append(issues, fmt.Sprintf("%s: explanation is required", name))
		}
		if strings.TrimSpace(question.Answer) == "" {
			issues = append(issues, fmt.Sprintf("%s: answer is required", name))
		}

		switch question.Kind {
		case "order":
			if len(question.Tokens) < 3 {
				issues = append(issues, fmt.Sprintf("%s: order questions need at least 3 tokens", name))
			}
			if strings.Join(question.Tokens, " ") != question.Answer {
				issues = append(issues, fmt.Sprintf("%s: answer must equal the tokens joined by single spaces", name))
			}
			if len(question.Options) > 0 {
				issues = append(issues, fmt.Sprintf("%s: order questions must not carry options", name))
			}
		case "true_false":
			if len(question.Options) != 2 {
				issues = append(issues, fmt.Sprintf("%s: true_false questions need exactly 2 options", name))
			}
			issues = append(issues, validateOptions(name, question)...)
		case "choice":
			if len(question.Options) < 3 || len(question.Options) > 4 {
				issues = append(issues, fmt.Sprintf("%s: choice questions need 3 or 4 options, found %d", name, len(question.Options)))
			}
			issues = append(issues, validateOptions(name, question)...)
		}
	}

	if len(questions) >= BankSize {
		for difficulty := MinDifficulty; difficulty <= MaxDifficulty; difficulty++ {
			if perLevel[difficulty] < QuestionsPerLevel {
				issues = append(issues, fmt.Sprintf("%s: %s bank needs %d questions of difficulty %d, found %d",
					label, bank, QuestionsPerLevel, difficulty, perLevel[difficulty]))
			}
		}
	}
	return issues
}

func validateOptions(name string, question Question) []string {
	issues := make([]string, 0)
	if len(question.Tokens) > 0 {
		issues = append(issues, fmt.Sprintf("%s: only order questions may carry tokens", name))
	}
	seen := map[string]bool{}
	found := false
	for _, option := range question.Options {
		if strings.TrimSpace(option) == "" {
			issues = append(issues, fmt.Sprintf("%s: options must not be empty", name))
		}
		if seen[option] {
			issues = append(issues, fmt.Sprintf("%s: duplicate option %q", name, option))
		}
		seen[option] = true
		if option == question.Answer {
			found = true
		}
	}
	if !found {
		issues = append(issues, fmt.Sprintf("%s: answer %q is not one of the options", name, question.Answer))
	}
	return issues
}
