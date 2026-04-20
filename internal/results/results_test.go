package results

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestAppendBallotRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "results", "poll.yaml")
	ballot := Ballot{
		SubmittedAt:    time.Date(2026, 4, 20, 12, 0, 0, 0, time.UTC),
		RespondentName: "Alice",
		Ranking:        []string{"kindred", "left-hand-of-darkness"},
	}

	if err := AppendBallot(path, ballot, "may-2026"); err != nil {
		t.Fatalf("AppendBallot() error = %v", err)
	}

	loaded, err := Load(path, "may-2026")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if loaded.PollID != "may-2026" {
		t.Fatalf("unexpected poll id %q", loaded.PollID)
	}
	if len(loaded.Ballots) != 1 {
		t.Fatalf("expected 1 ballot, got %d", len(loaded.Ballots))
	}
	if got := loaded.Ballots[0].RespondentName; got != ballot.RespondentName {
		t.Fatalf("unexpected respondent name %q", got)
	}
	if len(loaded.Ballots[0].Ranking) != 2 || loaded.Ballots[0].Ranking[0] != "kindred" {
		t.Fatalf("unexpected ranking %+v", loaded.Ballots[0].Ranking)
	}
}

func TestLoadRejectsMismatchedPollID(t *testing.T) {
	path := filepath.Join(t.TempDir(), "poll.yaml")
	if err := AppendBallot(path, Ballot{SubmittedAt: time.Now().UTC(), Ranking: []string{"a", "b"}}, "poll-a"); err != nil {
		t.Fatalf("AppendBallot() error = %v", err)
	}

	_, err := Load(path, "poll-b")
	if err == nil {
		t.Fatal("expected poll id mismatch error")
	}
}

func TestAppendBallotPreservesExistingBallots(t *testing.T) {
	path := filepath.Join(t.TempDir(), "poll.yaml")
	first := Ballot{SubmittedAt: time.Date(2026, 4, 20, 12, 0, 0, 0, time.UTC), Ranking: []string{"a", "b"}}
	second := Ballot{SubmittedAt: time.Date(2026, 4, 20, 13, 0, 0, 0, time.UTC), RespondentName: "Bob", Ranking: []string{"b", "a"}}

	if err := AppendBallot(path, first, "poll-a"); err != nil {
		t.Fatalf("AppendBallot() first error = %v", err)
	}
	if err := AppendBallot(path, second, "poll-a"); err != nil {
		t.Fatalf("AppendBallot() second error = %v", err)
	}

	loaded, err := Load(path, "poll-a")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(loaded.Ballots) != 2 {
		t.Fatalf("expected 2 ballots, got %d", len(loaded.Ballots))
	}
	if loaded.Ballots[1].RespondentName != "Bob" {
		t.Fatalf("expected second respondent Bob, got %q", loaded.Ballots[1].RespondentName)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if len(data) == 0 {
		t.Fatal("expected persisted YAML content")
	}
}
