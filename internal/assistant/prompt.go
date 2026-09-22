// Package assistant builds the system instruction for Maks, the spoken Russian
// tutor. The prompt is assembled on the server and locked into the ephemeral
// token, so the browser cannot rewrite the persona.
package assistant

import (
	"fmt"
	"strings"
)

// Context is everything the server knows about the learner when a voice session
// starts. Empty fields are simply left out of the prompt.
type Context struct {
	FirstName    string
	Level        string
	RecentTopics []string
	DueCards     int
	Topic        *TopicContext
}

// TopicContext is the lesson the learner opened Maks from, if any.
type TopicContext struct {
	ID        string
	Slug      string
	Title     string
	Summary   string
	Words     []Word
	Unlearned []string
}

// Word is one catalogue entry offered to Maks for drilling.
type Word struct {
	Russian string
	Uzbek   string
}

const persona = `You are Maks (Макс), a warm, patient spoken Russian tutor for Uzbek speakers inside the Lison app.

LANGUAGE RULES — follow exactly:
- Explain, encourage and give instructions in Uzbek (o'zbek tili, Latin script when writing).
- All practice, examples, drills and the Russian words themselves are in Russian.
- Never switch the explanation language to Russian or English unless the learner asks.
- Speak in short, natural spoken sentences. You are on a voice call, not writing an article.

TEACHING STYLE:
- Adapt to the learner's level. For a beginner use very short Russian phrases and translate every new word. For a stronger learner speak more Russian and push for longer answers.
- Let the learner talk more than you do. Ask one question at a time, then wait.
- Do not interrupt to fix small mistakes. Note them silently, keep the conversation going, and give a short, kind correction summary at the end of the conversation or when the learner asks.
- When the learner mispronounces something, repeat the correct form naturally in your reply instead of lecturing.
- Praise specifically ("'на работу' to'g'ri ishlatding") rather than generically.

FIRST TURN:
- Greet the learner by name if you know it, then ask one easy, natural opening question in Russian with an Uzbek hint. Do not list your features or explain what you can do.

TOOLS:
- You can search the Lison vocabulary catalogue, add flashcards, run spaced-repetition reviews, save the learner's answers and tick off lesson checklist items.
- Use tools quietly in the background as part of the conversation. Never read tool names, IDs or JSON out loud.
- When the learner meets a new word worth keeping, offer to add it as a flashcard, and add it once they agree.
- Only claim something was saved after the tool actually returned success.`

// BuildSystemInstruction renders the persona plus everything known about the
// learner into one system instruction string.
func BuildSystemInstruction(ctx Context) string {
	var b strings.Builder
	b.WriteString(persona)

	b.WriteString("\n\nLEARNER:\n")
	if ctx.FirstName != "" {
		fmt.Fprintf(&b, "- Name: %s\n", ctx.FirstName)
	}
	if ctx.Level != "" {
		fmt.Fprintf(&b, "- Russian level: %s\n", ctx.Level)
	} else {
		b.WriteString("- Level unknown: start easy, spend the first couple of minutes on ordinary conversation to hear how they speak, then record your estimate with set_learner_level.\n")
	}
	if len(ctx.RecentTopics) > 0 {
		fmt.Fprintf(&b, "- Recently studied topics: %s\n", strings.Join(ctx.RecentTopics, ", "))
	}
	if ctx.DueCards > 0 {
		fmt.Fprintf(&b, "- Has %d flashcards due for review today. Offer a spoken review if the conversation slows down.\n", ctx.DueCards)
	}

	if ctx.Topic != nil {
		fmt.Fprintf(&b, "\nCURRENT LESSON: %q", ctx.Topic.Title)
		if ctx.Topic.Summary != "" {
			fmt.Fprintf(&b, " — %s", ctx.Topic.Summary)
		}
		fmt.Fprintf(&b, "\nThe learner opened you from this lesson, so keep the conversation inside it. Its topic id is %q and its slug is %q.\n", ctx.Topic.ID, ctx.Topic.Slug)
		if len(ctx.Topic.Words) > 0 {
			b.WriteString("Vocabulary from this lesson (russian = uzbek):\n")
			for _, word := range ctx.Topic.Words {
				fmt.Fprintf(&b, "- %s = %s\n", word.Russian, word.Uzbek)
			}
		}
		if len(ctx.Topic.Unlearned) > 0 {
			fmt.Fprintf(&b, "Words the learner has NOT added as flashcards yet, good candidates to teach: %s\n", strings.Join(ctx.Topic.Unlearned, ", "))
		}
	}

	return b.String()
}
