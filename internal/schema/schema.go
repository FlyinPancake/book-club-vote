package schema

import (
	"encoding/json"
	"os"
	"path/filepath"
)

func Generate() ([]byte, error) {
	schema := map[string]any{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"$id":     "https://example.com/book-club-vote/config.schema.json",
		"title":   "Book Club Vote Config",
		"type":    "object",
		"additionalProperties": false,
		"required":             []string{"server", "polls"},
		"properties": map[string]any{
			"server": map[string]any{
				"type":                 "object",
				"additionalProperties": false,
				"required":             []string{"listen", "host_key_path", "accessible"},
				"properties": map[string]any{
					"listen": map[string]any{
						"type":      "string",
						"minLength": 1,
					},
					"host_key_path": map[string]any{
						"type":      "string",
						"minLength": 1,
					},
					"accessible": map[string]any{
						"type": "boolean",
					},
				},
			},
			"polls": map[string]any{
				"type":     "array",
				"minItems": 1,
				"items": map[string]any{
					"type":                 "object",
					"additionalProperties": false,
					"required": []string{
						"id",
						"name",
						"description",
						"start",
						"end",
						"record_respondent_name",
						"results_path",
						"books",
					},
					"properties": map[string]any{
						"id": map[string]any{"type": "string", "minLength": 1},
						"name": map[string]any{"type": "string", "minLength": 1},
						"description": map[string]any{"type": "string", "minLength": 1},
						"start": map[string]any{"type": "string", "format": "date-time"},
						"end": map[string]any{"type": "string", "format": "date-time"},
						"record_respondent_name": map[string]any{"type": "boolean"},
						"results_path": map[string]any{"type": "string", "minLength": 1},
						"books": map[string]any{
							"type":     "array",
							"minItems": 2,
							"items": map[string]any{
								"type":                 "object",
								"additionalProperties": false,
								"required": []string{"id", "author", "title", "goodreads_url", "moly_url"},
								"properties": map[string]any{
									"id": map[string]any{"type": "string", "minLength": 1},
									"author": map[string]any{"type": "string", "minLength": 1},
									"title": map[string]any{"type": "string", "minLength": 1},
									"goodreads_url": map[string]any{"type": "string", "format": "uri"},
									"moly_url": map[string]any{"type": "string", "format": "uri"},
								},
							},
						},
					},
				},
			},
		},
	}

	return json.MarshalIndent(schema, "", "  ")
}

func WriteFile(path string) error {
	data, err := Generate()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o644)
}
