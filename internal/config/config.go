package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server Server `yaml:"server"`
	Polls  []Poll `yaml:"polls"`
}

type Server struct {
	Listen      string `yaml:"listen"`
	HostKeyPath string `yaml:"host_key_path"`
	Accessible  bool   `yaml:"accessible"`
}

type Poll struct {
	ID                   string    `yaml:"id"`
	Name                 string    `yaml:"name"`
	Description          string    `yaml:"description"`
	Start                time.Time `yaml:"start"`
	End                  time.Time `yaml:"end"`
	RecordRespondentName bool      `yaml:"record_respondent_name"`
	ResultsPath          string    `yaml:"results_path"`
	Books                []Book    `yaml:"books"`
}

type Book struct {
	ID           string `yaml:"id"`
	Author       string `yaml:"author"`
	Title        string `yaml:"title"`
	GoodreadsURL string `yaml:"goodreads_url"`
	MolyURL      string `yaml:"moly_url"`
}

func Load(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, err
	}

	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func (c Config) Validate() error {
	var problems []string

	if strings.TrimSpace(c.Server.Listen) == "" {
		problems = append(problems, "server.listen is required")
	}
	if strings.TrimSpace(c.Server.HostKeyPath) == "" {
		problems = append(problems, "server.host_key_path is required")
	}
	if len(c.Polls) == 0 {
		problems = append(problems, "at least one poll is required")
	}

	pollIDs := make(map[string]struct{}, len(c.Polls))
	for i, poll := range c.Polls {
		prefix := fmt.Sprintf("polls[%d]", i)

		if strings.TrimSpace(poll.ID) == "" {
			problems = append(problems, prefix+".id is required")
		} else {
			if _, ok := pollIDs[poll.ID]; ok {
				problems = append(problems, fmt.Sprintf("duplicate poll id %q", poll.ID))
			}
			pollIDs[poll.ID] = struct{}{}
		}

		if strings.TrimSpace(poll.Name) == "" {
			problems = append(problems, prefix+".name is required")
		}
		if strings.TrimSpace(poll.Description) == "" {
			problems = append(problems, prefix+".description is required")
		}
		if poll.Start.IsZero() {
			problems = append(problems, prefix+".start is required")
		}
		if poll.End.IsZero() {
			problems = append(problems, prefix+".end is required")
		}
		if !poll.Start.IsZero() && !poll.End.IsZero() && !poll.Start.Before(poll.End) {
			problems = append(problems, prefix+".start must be before .end")
		}
		if strings.TrimSpace(poll.ResultsPath) == "" {
			problems = append(problems, prefix+".results_path is required")
		}
		if len(poll.Books) < 2 {
			problems = append(problems, prefix+" must contain at least two books")
		}

		bookIDs := make(map[string]struct{}, len(poll.Books))
		for j, book := range poll.Books {
			bookPrefix := fmt.Sprintf("%s.books[%d]", prefix, j)
			if strings.TrimSpace(book.ID) == "" {
				problems = append(problems, bookPrefix+".id is required")
			} else {
				if _, ok := bookIDs[book.ID]; ok {
					problems = append(problems, fmt.Sprintf("duplicate book id %q in poll %q", book.ID, poll.ID))
				}
				bookIDs[book.ID] = struct{}{}
			}
			if strings.TrimSpace(book.Author) == "" {
				problems = append(problems, bookPrefix+".author is required")
			}
			if strings.TrimSpace(book.Title) == "" {
				problems = append(problems, bookPrefix+".title is required")
			}
			if err := validateURL(book.GoodreadsURL); err != nil {
				problems = append(problems, fmt.Sprintf("%s.goodreads_url %v", bookPrefix, err))
			}
			if err := validateURL(book.MolyURL); err != nil {
				problems = append(problems, fmt.Sprintf("%s.moly_url %v", bookPrefix, err))
			}
		}
	}

	if len(problems) > 0 {
		sort.Strings(problems)
		return errors.New(strings.Join(problems, "; "))
	}

	return nil
}

func (c Config) ActivePolls(now time.Time) []Poll {
	active := make([]Poll, 0, len(c.Polls))
	for _, poll := range c.Polls {
		if !now.Before(poll.Start) && now.Before(poll.End) {
			active = append(active, poll)
		}
	}
	return active
}

func (p Poll) BookLabel(bookID string) string {
	for _, book := range p.Books {
		if book.ID == bookID {
			return fmt.Sprintf("%s by %s", book.Title, book.Author)
		}
	}
	return bookID
}

func validateURL(raw string) error {
	if strings.TrimSpace(raw) == "" {
		return fmt.Errorf("is required")
	}
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("must be a valid absolute URL")
	}
	if !u.IsAbs() || u.Host == "" {
		return fmt.Errorf("must be a valid absolute URL")
	}
	return nil
}
