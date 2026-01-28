package atproto

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestLexiconToMarkdown(t *testing.T) {
	lexJSON := `{
  "lexicon": 1,
  "id": "app.bsky.feed.post",
  "defs": {
    "main": {
      "type": "record",
      "description": "Record containing a Bluesky post.",
      "record": {
        "type": "object",
        "required": ["text", "createdAt"],
        "properties": {
          "text": {
            "type": "string",
            "maxLength": 3000,
            "description": "The primary post content."
          },
          "langs": {
            "type": "array",
            "items": { "type": "string", "format": "language" }
          },
          "createdAt": {
            "type": "string",
            "format": "datetime"
          }
        }
      }
    }
  }
}`

	var lex Lexicon
	if err := json.Unmarshal([]byte(lexJSON), &lex); err != nil {
		t.Fatalf("Failed to unmarshal lexicon: %v", err)
	}

	md := LexiconToMarkdown(&lex)

	expectedSubstrings := []string{
		"# app.bsky.feed.post",
		"## Definition: app.bsky.feed.post",
		"Record containing a Bluesky post.",
		"| Name | Type | Required | Description |",
		"| text | string | Yes | The primary post content. |",
		"| langs | array of string | No |  |",
		"| createdAt | string | Yes | (Format: datetime)  |",
	}

	for _, sub := range expectedSubstrings {
		if !strings.Contains(md, sub) {
			t.Errorf("Expected markdown to contain %q, but it didn't.\nGot:\n%s", sub, md)
		}
	}
}
