package results

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

type File struct {
	PollID  string   `yaml:"poll_id"`
	Ballots []Ballot `yaml:"ballots"`
}

type Ballot struct {
	SubmittedAt    time.Time `yaml:"submitted_at"`
	RespondentName string    `yaml:"respondent_name,omitempty"`
	Ranking        []string  `yaml:"ranking"`
}

var (
	locksMu   sync.Mutex
	fileLocks = map[string]*sync.Mutex{}
)

func AppendBallot(path string, ballot Ballot, pollID string) error {
	mu := lockFor(path)
	mu.Lock()
	defer mu.Unlock()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	resultsFile, err := Load(path, pollID)
	if err != nil {
		return err
	}
	resultsFile.Ballots = append(resultsFile.Ballots, ballot)

	data, err := yaml.Marshal(resultsFile)
	if err != nil {
		return err
	}

	return writeAtomically(path, data)
}

func Load(path string, pollID string) (File, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return File{PollID: pollID, Ballots: []Ballot{}}, nil
		}
		return File{}, err
	}

	var file File
	if err := yaml.Unmarshal(data, &file); err != nil {
		return File{}, err
	}
	if file.PollID == "" {
		file.PollID = pollID
	}
	if file.PollID != pollID {
		return File{}, fmt.Errorf("results file poll_id %q does not match poll %q", file.PollID, pollID)
	}
	if file.Ballots == nil {
		file.Ballots = []Ballot{}
	}
	return file, nil
}

func lockFor(path string) *sync.Mutex {
	locksMu.Lock()
	defer locksMu.Unlock()
	if mu, ok := fileLocks[path]; ok {
		return mu
	}
	mu := &sync.Mutex{}
	fileLocks[path] = mu
	return mu
}

func writeAtomically(path string, data []byte) error {
	dir := filepath.Dir(path)
	file, err := os.CreateTemp(dir, ".bookclubvote-*.yaml")
	if err != nil {
		return err
	}
	tmpPath := file.Name()
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.Remove(tmpPath)
		}
	}()

	if _, err := file.Write(data); err != nil {
		_ = file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmpPath, 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return err
	}
	cleanup = false
	return nil
}
