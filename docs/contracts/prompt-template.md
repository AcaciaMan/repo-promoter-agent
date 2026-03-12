# Prompt Template — v1 (Phase 1)

This document defines the user message template sent to the Gradient agent in `messages[0].content`.

---

## 1. Prompt Template

```
Generate promotional content for this GitHub repository.

REPO DATA:
{{.RepoDataJSON}}

Return a JSON object with exactly this structure:
{{.OutputSchemaJSON}}

RULES:
- Output ONLY the JSON object. No markdown fences, no preamble, no explanation.
- Echo repo_url and repo_name exactly from the input.
- Generate exactly 3 twitter_posts, each ≤280 characters including hashtags and URL.
- Generate 3–5 key_benefits.
- Generate 5–8 tags, expanding on the input topics.
- linkedin_post should be 150–300 words with professional tone.
- If target_channel is "twitter", optimize tone for Twitter. If "linkedin", optimize for LinkedIn. Always populate all fields regardless.
- Stay faithful to the repo data. Do not invent features not described in the input.
- Tailor content to target_audience if provided.
```

---

## 2. Go Code Snippet

**Recommendation: `text/template`** over `fmt.Sprintf`.

`text/template` is the right choice because:
- The template has two clearly named placeholders rather than positional `%s` args.
- If the repo data JSON contains `%` characters, `fmt.Sprintf` would break. `text/template` has no such issue.
- Easy to extend with additional placeholders later without reordering arguments.

```go
package agent

import (
	"bytes"
	"text/template"
)

const promptTemplate = `Generate promotional content for this GitHub repository.

REPO DATA:
{{.RepoDataJSON}}

Return a JSON object with exactly this structure:
{{.OutputSchemaJSON}}

RULES:
- Output ONLY the JSON object. No markdown fences, no preamble, no explanation.
- Echo repo_url and repo_name exactly from the input.
- Generate exactly 3 twitter_posts, each ≤280 characters including hashtags and URL.
- Generate 3–5 key_benefits.
- Generate 5–8 tags, expanding on the input topics.
- linkedin_post should be 150–300 words with professional tone.
- If target_channel is "twitter", optimize tone for Twitter. If "linkedin", optimize for LinkedIn. Always populate all fields regardless.
- Stay faithful to the repo data. Do not invent features not described in the input.
- Tailor content to target_audience if provided.`

const outputSchema = `{
  "repo_url": "string",
  "repo_name": "string",
  "headline": "string",
  "summary": "string",
  "key_benefits": ["string"],
  "tags": ["string"],
  "twitter_posts": ["string"],
  "linkedin_post": "string",
  "call_to_action": "string"
}`

var promptTmpl = template.Must(template.New("prompt").Parse(promptTemplate))

// BuildPrompt renders the user message with the given repo data JSON.
func BuildPrompt(repoDataJSON string) (string, error) {
	var buf bytes.Buffer
	err := promptTmpl.Execute(&buf, map[string]string{
		"RepoDataJSON":     repoDataJSON,
		"OutputSchemaJSON": outputSchema,
	})
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}
```

---

## 3. Fallback Extraction Strategy

If the agent wraps JSON in markdown fences or adds preamble text, the Go backend should apply a two-step extraction before parsing:

1. **Strip markdown fences**: Use a regex to detect `` ```json ... ``` `` or `` ``` ... ``` `` wrapping. If found, extract the content between the fences.
2. **Find the JSON object**: If step 1 didn't apply (or the result still fails `json.Unmarshal`), scan the response string for the first `{` and last `}`, extract that substring, and attempt to parse it.

This covers the two most common LLM failure modes (markdown fences and "Here is the JSON:" preamble) with minimal code. If the extracted substring still fails `json.Unmarshal`, return an error to the caller — don't attempt further repair.
