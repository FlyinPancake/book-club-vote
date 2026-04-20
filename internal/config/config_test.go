package config

import (
	"strings"
	"testing"
	"time"
)

func TestValidateAcceptsValidConfig(t *testing.T) {
	cfg := validConfig()

	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate() returned error: %v", err)
	}
}

func TestValidateRejectsDuplicatePollID(t *testing.T) {
	cfg := validConfig()
	cfg.Polls = append(cfg.Polls, cfg.Polls[0])

	err := cfg.Validate()
	if err == nil || !strings.Contains(err.Error(), "duplicate poll id") {
		t.Fatalf("expected duplicate poll id error, got %v", err)
	}
}

func TestValidateRejectsBadDates(t *testing.T) {
	cfg := validConfig()
	cfg.Polls[0].End = cfg.Polls[0].Start

	err := cfg.Validate()
	if err == nil || !strings.Contains(err.Error(), ".start must be before .end") {
		t.Fatalf("expected date ordering error, got %v", err)
	}
}

func TestActivePolls(t *testing.T) {
	now := time.Date(2026, 4, 20, 12, 0, 0, 0, time.UTC)
	cfg := validConfig()
	cfg.Polls = append(cfg.Polls,
		Poll{
			ID:                   "future",
			Name:                 "Future",
			Description:          "Later",
			Start:                now.Add(24 * time.Hour),
			End:                  now.Add(48 * time.Hour),
			RecordRespondentName: false,
			ResultsPath:          "future.yaml",
			Books:                cfg.Polls[0].Books,
		},
	)

	active := cfg.ActivePolls(now)
	if len(active) != 1 || active[0].ID != cfg.Polls[0].ID {
		t.Fatalf("unexpected active polls: %+v", active)
	}
}

func validConfig() Config {
	now := time.Date(2026, 4, 20, 0, 0, 0, 0, time.UTC)
	return Config{
		Server: Server{
			Listen:      ":23234",
			HostKeyPath: "./data/ssh_host_ed25519_key",
			Accessible:  false,
		},
		Polls: []Poll{{
			ID:                   "may-2026",
			Name:                 "May 2026",
			Description:          "Vote for next month",
			Start:                now,
			End:                  now.Add(24 * time.Hour),
			RecordRespondentName: true,
			ResultsPath:          "./data/results/may-2026.yaml",
			Books: []Book{
				{
					ID:           "kindred",
					Author:       "Octavia E. Butler",
					Title:        "Kindred",
					GoodreadsURL: "https://www.goodreads.com/book/show/60931.Kindred",
					MolyURL:      "https://moly.hu/konyvek/octavia-e-butler-kindred",
				},
				{
					ID:           "left-hand-of-darkness",
					Author:       "Ursula K. Le Guin",
					Title:        "The Left Hand of Darkness",
					GoodreadsURL: "https://www.goodreads.com/book/show/18423.The_Left_Hand_of_Darkness",
					MolyURL:      "https://moly.hu/konyvek/ursula-k-le-guin-the-left-hand-of-darkness",
				},
			},
		}},
	}
}
