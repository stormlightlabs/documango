package atproto

import (
	"fmt"
	"sort"
	"strings"
)

type Lexicon struct {
	Lexicon int                   `json:"lexicon"`
	ID      string                `json:"id"`
	Defs    map[string]Definition `json:"defs"`
}

type Definition struct {
	Type        string              `json:"type"`
	Description string              `json:"description,omitempty"`
	Parameters  *Schema             `json:"parameters,omitempty"`
	Input       *Schema             `json:"input,omitempty"`
	Output      *Schema             `json:"output,omitempty"`
	Record      *Schema             `json:"record,omitempty"`
	Properties  map[string]Property `json:"properties,omitempty"`
	Required    []string            `json:"required,omitempty"`
}

type Schema struct {
	Type        string              `json:"type"`
	Properties  map[string]Property `json:"properties,omitempty"`
	Required    []string            `json:"required,omitempty"`
	Description string              `json:"description,omitempty"`
}

type Property struct {
	Type        string    `json:"type"`
	Description string    `json:"description,omitempty"`
	Ref         string    `json:"ref,omitempty"`
	Refs        []string  `json:"refs,omitempty"`
	Items       *Property `json:"items,omitempty"`
	Format      string    `json:"format,omitempty"`
	Minimum     *int      `json:"minimum,omitempty"`
	Maximum     *int      `json:"maximum,omitempty"`
	MinLength   *int      `json:"minLength,omitempty"`
	MaxLength   *int      `json:"maxLength,omitempty"`
}

func LexiconToMarkdown(lex *Lexicon) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# %s\n\n", lex.ID))

	keys := make([]string, 0, len(lex.Defs))
	for k := range lex.Defs {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	if def, ok := lex.Defs["main"]; ok {
		renderDefinition(&sb, lex.ID, "main", def)
	}

	for _, k := range keys {
		if k == "main" {
			continue
		}
		renderDefinition(&sb, lex.ID, k, lex.Defs[k])
	}

	return sb.String()
}

func renderDefinition(sb *strings.Builder, nsid, name string, def Definition) {
	if name == "main" {
		sb.WriteString(fmt.Sprintf("## Definition: %s\n\n", nsid))
	} else {
		sb.WriteString(fmt.Sprintf("## Definition: %s#%s\n\n", nsid, name))
	}

	if def.Description != "" {
		sb.WriteString(def.Description + "\n\n")
	}

	fmt.Fprintf(sb, "- **Type**: %s\n", def.Type)

	switch def.Type {
	case "record":
		if def.Record != nil {
			sb.WriteString("\n### Record Properties\n\n")
			renderProperties(sb, def.Record.Properties, def.Record.Required)
		}
	case "query", "procedure":
		if def.Parameters != nil && len(def.Parameters.Properties) > 0 {
			sb.WriteString("\n### Parameters\n\n")
			renderProperties(sb, def.Parameters.Properties, def.Parameters.Required)
		}
		if def.Input != nil && def.Input.Type == "object" && len(def.Input.Properties) > 0 {
			sb.WriteString("\n### Input\n\n")
			renderProperties(sb, def.Input.Properties, def.Input.Required)
		}
		if def.Output != nil && def.Output.Type == "object" && len(def.Output.Properties) > 0 {
			sb.WriteString("\n### Output\n\n")
			renderProperties(sb, def.Output.Properties, def.Output.Required)
		}
	case "object":
		sb.WriteString("\n### Properties\n\n")
		renderProperties(sb, def.Properties, def.Required)
	}

	sb.WriteString("\n")
}

func renderProperties(sb *strings.Builder, props map[string]Property, required []string) {
	if len(props) == 0 {
		return
	}

	reqMap := make(map[string]bool)
	for _, r := range required {
		reqMap[r] = true
	}

	sb.WriteString("| Name | Type | Required | Description |\n")
	sb.WriteString("| ---- | ---- | -------- | ----------- |\n")

	keys := make([]string, 0, len(props))
	for k := range props {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		p := props[k]
		typeStr := p.Type
		if p.Ref != "" {
			typeStr = fmt.Sprintf("ref(%s)", p.Ref)
		} else if p.Type == "union" {
			typeStr = fmt.Sprintf("union(%s)", strings.Join(p.Refs, ", "))
		} else if p.Type == "array" && p.Items != nil {
			itemType := p.Items.Type
			if p.Items.Ref != "" {
				itemType = fmt.Sprintf("ref(%s)", p.Items.Ref)
			}
			typeStr = fmt.Sprintf("array of %s", itemType)
		}

		req := "No"
		if reqMap[k] {
			req = "Yes"
		}

		desc := p.Description
		if p.Format != "" {
			desc = fmt.Sprintf("(Format: %s) %s", p.Format, desc)
		}

		sb.WriteString(fmt.Sprintf("| %s | %s | %s | %s |\n", k, typeStr, req, desc))
	}
}
