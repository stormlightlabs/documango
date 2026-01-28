package rust

import (
	"strings"
	"testing"

	"github.com/PuerkitoBio/goquery"
)

func TestParseRustdocHTMLStream(t *testing.T) {
	tests := []struct {
		name     string
		html     string
		expected []string
	}{
		{
			name: "Simple trait with docblock",
			html: `
				<main>
					<div class="main-heading">
						<h1>Trait Deserialize <button id="copy-path">Copy</button></h1>
					</div>
					<pre class="rust item-decl"><code>pub trait Deserialize { fn deserialize(); }</code></pre>
					<div class="docblock">
						<p>A data structure that can be deserialized.</p>
						<p>More info here.</p>
					</div>
				</main>`,
			expected: []string{
				"# Trait Deserialize",
				"```",
				"pub trait Deserialize { fn deserialize(); }",
				"```",
				"A data structure that can be deserialized.",
				"More info here.",
			},
		},
		{
			name: "Headers with anchor symbols",
			html: `
				<main>
					<h1>Trait Deserialize</h1>
					<h2 id="lifetime"><a class="doc-anchor" href="#lifetime">§</a>Lifetime</h2>
					<div class="docblock">
						<p>Lifetime info.</p>
					</div>
					<h2 id="required-methods" class="section-header">Required Methods<a href="#required-methods" class="anchor">§</a></h2>
				</main>`,
			expected: []string{
				"# Trait Deserialize",
				"## Lifetime",
				"Lifetime info.",
				"## Required Methods",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := goquery.NewDocumentFromReader(strings.NewReader(tt.html))
			if err != nil {
				t.Fatalf("failed to parse HTML: %v", err)
			}

			got := parseRustdocHTMLFromDoc(doc)
			gotLines := strings.Split(got, "\n")

			j := 0
			for i := 0; i < len(tt.expected); i++ {
				found := false
				for j < len(gotLines) {
					if strings.TrimSpace(gotLines[j]) == strings.TrimSpace(tt.expected[i]) {
						found = true
						j++
						break
					}
					j++
				}
				if !found {
					t.Errorf("Expected snippet %q not found in order. Got:\n%s", tt.expected[i], got)
				}
			}
		})
	}
}
