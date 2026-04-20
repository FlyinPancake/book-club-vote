# Book Club Vote

`book-club-vote` is a Wish-based SSH app for running ranked-choice book club polls from the terminal.

## Features

- SSH server built with `wish`
- Bubble Tea voting flow for poll selection, optional name entry, ranking, and review
- YAML config for server and poll definitions
- JSON Schema for config validation at `schema/config.schema.json`
- Multiple polls can be open at the same time
- Per-poll YAML ballot storage
- Optional respondent name recording per poll
- Full ranking across all books in a poll

## Config

Start from `config.example.yaml`.

Top-level sections:

- `server`
- `polls`

`server` fields:

- `listen`
- `host_key_path`
- `accessible`

Each poll defines:

- `id`
- `name`
- `description`
- `start`
- `end`
- `record_respondent_name`
- `results_path`
- `books`

Each book defines:

- `id`
- `author`
- `title`
- `goodreads_url`
- `moly_url`

Example:

```yaml
server:
  listen: ":23234"
  host_key_path: "./data/ssh_host_ed25519_key"
  accessible: false

polls:
  - id: "may-2026"
    name: "May 2026 Book"
    description: "Vote for next month's read"
    start: "2026-04-20T00:00:00Z"
    end: "2026-04-27T23:59:59Z"
    record_respondent_name: true
    results_path: "./data/results/may-2026.yaml"
    books:
      - id: "kindred"
        author: "Octavia E. Butler"
        title: "Kindred"
        goodreads_url: "https://www.goodreads.com/book/show/60931.Kindred"
        moly_url: "https://moly.hu/konyvek/octavia-e-butler-kindred"
      - id: "left-hand-of-darkness"
        author: "Ursula K. Le Guin"
        title: "The Left Hand of Darkness"
        goodreads_url: "https://www.goodreads.com/book/show/18423.The_Left_Hand_of_Darkness"
        moly_url: "https://moly.hu/konyvek/ursula-k-le-guin-the-left-hand-of-darkness"
```

## Results Format

Each poll writes to its own YAML file.

```yaml
poll_id: "may-2026"
ballots:
  - submitted_at: "2026-04-20T12:30:00Z"
    respondent_name: "Alice"
    ranking:
      - "kindred"
      - "left-hand-of-darkness"
```

`respondent_name` is omitted when blank.
Rankings are stored by `book.id`, while the UI displays title and author.

## CLI

Write the schema and exit:

```bash
go run ./cmd/book-club-vote -write-schema
```

Validate a config file and exit:

```bash
go run ./cmd/book-club-vote -config ./config.yaml -validate-config
```

Write the schema, then validate the config:

```bash
go run ./cmd/book-club-vote -write-schema -validate-config -config ./config.yaml
```

Run the SSH server:

```bash
go run ./cmd/book-club-vote -config ./config.yaml
```

## Usage

Connect to the server:

```bash
ssh -p 23234 localhost
```

Voting flow:

1. If multiple polls are active, choose one.
2. If the poll records respondent names, enter a name or leave it blank.
3. Rank every book in order.
4. Review the ballot.
5. Submit or restart ranking.

## Behavior

- Only polls active at the current time are shown.
- If exactly one poll is active, the app goes straight into that poll.
- Rankings must include every book exactly once.
- Respondent names are only recorded when `record_respondent_name: true` for that poll.
- `server.host_key_path` controls the SSH host key location.

## Storage Notes

- Results storage is designed for a single running server instance writing local YAML files.
- Ballot writes use atomic temp-file replacement to reduce the chance of truncated results files.
- This app does not currently implement cross-process file locking.

## Testing

```bash
go test ./...
go build ./cmd/book-club-vote
```
