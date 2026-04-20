package ui

import (
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"bookclubvote/internal/config"
)

func TestValidateRankingAcceptsFullUniqueOrder(t *testing.T) {
	poll := testPoll()
	err := validateRanking(poll, []string{"kindred", "left-hand-of-darkness", "piranesi"})
	if err != nil {
		t.Fatalf("validateRanking() error = %v", err)
	}
}

func TestValidateRankingRejectsDuplicateBook(t *testing.T) {
	poll := testPoll()
	err := validateRanking(poll, []string{"kindred", "kindred", "piranesi"})
	if err == nil || !strings.Contains(err.Error(), "more than once") {
		t.Fatalf("expected duplicate-book error, got %v", err)
	}
}

func TestRemainingBooksFiltersSelectedChoices(t *testing.T) {
	poll := testPoll()
	remaining := remainingBooks(poll, []string{"kindred", "piranesi"})
	if len(remaining) != 1 {
		t.Fatalf("expected 1 remaining book, got %d", len(remaining))
	}
	if remaining[0].ID != "left-hand-of-darkness" {
		t.Fatalf("unexpected remaining book %q", remaining[0].ID)
	}
}

func TestUpdateRankCanStepBackOneChoice(t *testing.T) {
	m := Model{
		selected:    testPoll(),
		state:       stateRank,
		ranking:     []string{"kindred", "", ""},
		currentRank: 1,
		rankCursor:  1,
	}

	updated, _ := m.updateRank(tea.KeyPressMsg(tea.Key{Text: "h", Code: 'h'}))
	got := updated.(Model)

	if got.currentRank != 0 {
		t.Fatalf("expected currentRank 0, got %d", got.currentRank)
	}
	if got.ranking[0] != "" {
		t.Fatalf("expected first ranking slot cleared, got %q", got.ranking[0])
	}
}

func TestUpdateConfirmEditRankingRestartsFlow(t *testing.T) {
	m := Model{
		selected:      testPoll(),
		state:         stateConfirm,
		ranking:       []string{"kindred", "left-hand-of-darkness", "piranesi"},
		confirmCursor: 1,
	}

	updated, _ := m.updateConfirm(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	got := updated.(Model)

	if got.state != stateRank {
		t.Fatalf("expected stateRank, got %v", got.state)
	}
	if got.currentRank != 0 {
		t.Fatalf("expected currentRank reset to 0, got %d", got.currentRank)
	}
	for i, bookID := range got.ranking {
		if bookID != "" {
			t.Fatalf("expected cleared ranking at %d, got %q", i, bookID)
		}
	}
}

func TestUpdateNameMovesToRankingAndPreservesInput(t *testing.T) {
	m := Model{
		activePolls: []config.Poll{testPoll()},
		selected:    testPoll(),
		state:       stateEnterName,
		ranking:     make([]string, len(testPoll().Books)),
	}

	updated, _ := m.updateName(tea.KeyPressMsg(tea.Key{Text: "A", Code: 'A'}))
	got := updated.(Model)
	updated, _ = got.updateName(tea.KeyPressMsg(tea.Key{Text: "l", Code: 'l'}))
	got = updated.(Model)
	updated, _ = got.updateName(tea.KeyPressMsg(tea.Key{Text: "i", Code: 'i'}))
	got = updated.(Model)
	updated, _ = got.updateName(tea.KeyPressMsg(tea.Key{Text: "c", Code: 'c'}))
	got = updated.(Model)
	updated, _ = got.updateName(tea.KeyPressMsg(tea.Key{Text: "e", Code: 'e'}))
	got = updated.(Model)
	updated, _ = got.updateName(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	got = updated.(Model)

	if got.state != stateRank {
		t.Fatalf("expected stateRank, got %v", got.state)
	}
	if got.respondentName != "Alice" {
		t.Fatalf("expected respondentName Alice, got %q", got.respondentName)
	}
}

func TestUpdatePollSelectionChoosesHighlightedPoll(t *testing.T) {
	polls := []config.Poll{testPoll(), secondPoll()}
	m := Model{activePolls: polls, state: stateChoosePoll, pollCursor: 1}

	updated, _ := m.updatePollSelection(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	got := updated.(Model)

	if got.selected.ID != secondPoll().ID {
		t.Fatalf("expected selected poll %q, got %q", secondPoll().ID, got.selected.ID)
	}
}

func testPoll() config.Poll {
	now := time.Date(2026, 4, 20, 0, 0, 0, 0, time.UTC)
	return config.Poll{
		ID:                   "may-2026",
		Name:                 "May 2026",
		Description:          "Vote for next month",
		Start:                now,
		End:                  now.Add(24 * time.Hour),
		RecordRespondentName: true,
		ResultsPath:          "./data/results/may-2026.yaml",
		Books: []config.Book{
			{ID: "kindred", Author: "Octavia E. Butler", Title: "Kindred"},
			{ID: "left-hand-of-darkness", Author: "Ursula K. Le Guin", Title: "The Left Hand of Darkness"},
			{ID: "piranesi", Author: "Susanna Clarke", Title: "Piranesi"},
		},
	}
}

func secondPoll() config.Poll {
	now := time.Date(2026, 4, 20, 0, 0, 0, 0, time.UTC)
	return config.Poll{
		ID:                   "june-2026",
		Name:                 "June 2026",
		Description:          "Vote for June",
		Start:                now,
		End:                  now.Add(24 * time.Hour),
		RecordRespondentName: false,
		ResultsPath:          "./data/results/june-2026.yaml",
		Books: []config.Book{
			{ID: "piranesi", Author: "Susanna Clarke", Title: "Piranesi"},
			{ID: "the-dispossessed", Author: "Ursula K. Le Guin", Title: "The Dispossessed"},
		},
	}
}
