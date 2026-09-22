// Package discussion loads the authored conversation questions Maks pulls from
// when a spoken session needs a fresh direction. The JSON files are the single
// source of truth and are synced into the database at start-up, exactly like the
// grammar bank.
package discussion

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

// MinQuestionsPerSet keeps a set from being so thin that a learner exhausts it
// in one conversation.
const MinQuestionsPerSet = 6

var (
	setIDPattern      = regexp.MustCompile(`^discussion_[a-z0-9_]+$`)
	questionIDPattern = regexp.MustCompile(`^[a-z0-9_]+$`)
	topicIDPattern    = regexp.MustCompile(`^topic_[a-z0-9_]+$`)
)

// Question is one open question, plus the follow-ups that keep it going.
type Question struct {
	ID        string   `json:"id"`
	Question  string   `json:"question"`
	UzbekHint string   `json:"uzbekHint"`
	Level     string   `json:"level"`
	FollowUps []string `json:"followUps"`
}

// Set groups questions. A set with no TopicID is general small talk, usable in
// any conversation.
type Set struct {
	ID        string     `json:"id"`
	TopicID   string     `json:"topicId"`
	Title     string     `json:"title"`
	Position  int        `json:"position"`
	Questions []Question `json:"questions"`
}

// LoadSets reads every embedded discussion set ordered by position.
func LoadSets() ([]Set, error) {
	return loadSets(contents.Discussions, contents.DiscussionsDir)
}

func loadSets(files fs.FS, dir string) ([]Set, error) {
	entries, err := fs.ReadDir(files, dir)
	if err != nil {
		return nil, fmt.Errorf("read discussion content: %w", err)
	}

	sets := make([]Set, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		body, err := fs.ReadFile(files, path.Join(dir, entry.Name()))
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", entry.Name(), err)
		}
		var set Set
		decoder := json.NewDecoder(strings.NewReader(string(body)))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&set); err != nil {
			return nil, fmt.Errorf("decode %s: %w", entry.Name(), err)
		}
		sets = append(sets, set)
	}
	sort.Slice(sets, func(i, j int) bool { return sets[i].Position < sets[j].Position })
	return sets, nil
}

// Validate reports every content problem at once, keeping the authoring loop
// short.
func Validate(sets []Set) []string {
	issues := make([]string, 0)
	if len(sets) == 0 {
		return append(issues, "no discussion sets found")
	}

	seenSet := map[string]bool{}
	seenQuestion := map[string]string{}
	for _, set := range sets {
		label := set.ID
		if label == "" {
			label = set.Title
		}
		if !setIDPattern.MatchString(set.ID) {
			issues = append(issues, fmt.Sprintf("%s: id must match discussion_<lower_snake_case>", label))
		}
		if seenSet[set.ID] {
			issues = append(issues, fmt.Sprintf("%s: duplicate set id", label))
		}
		seenSet[set.ID] = true
		if strings.TrimSpace(set.Title) == "" {
			issues = append(issues, fmt.Sprintf("%s: title is required", label))
		}
		if set.Position < 1 {
			issues = append(issues, fmt.Sprintf("%s: position must be 1 or greater", label))
		}
		if set.TopicID != "" && !topicIDPattern.MatchString(set.TopicID) {
			issues = append(issues, fmt.Sprintf("%s: topicId must match topic_<lower_snake_case> or be empty", label))
		}
		if len(set.Questions) < MinQuestionsPerSet {
			issues = append(issues, fmt.Sprintf("%s: needs at least %d questions, found %d", label, MinQuestionsPerSet, len(set.Questions)))
		}

		for index, question := range set.Questions {
			name := fmt.Sprintf("%s question %d", label, index+1)
			if !questionIDPattern.MatchString(question.ID) {
				issues = append(issues, fmt.Sprintf("%s: id must be lower_snake_case", name))
				continue
			}
			name = label + " " + question.ID
			if previous, exists := seenQuestion[question.ID]; exists {
				issues = append(issues, fmt.Sprintf("%s: duplicate question id, already used by %s", name, previous))
			}
			seenQuestion[question.ID] = label
			if strings.TrimSpace(question.Question) == "" {
				issues = append(issues, fmt.Sprintf("%s: question text is required", name))
			}
			if strings.TrimSpace(question.UzbekHint) == "" {
				issues = append(issues, fmt.Sprintf("%s: uzbekHint is required so a beginner can follow", name))
			}
			if len(question.FollowUps) == 0 {
				issues = append(issues, fmt.Sprintf("%s: needs at least one follow-up", name))
			}
			for _, followUp := range question.FollowUps {
				if strings.TrimSpace(followUp) == "" {
					issues = append(issues, fmt.Sprintf("%s: follow-ups must not be empty", name))
				}
			}
		}
	}
	return issues
}
