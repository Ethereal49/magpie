package redact

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Only a file's data written whole into a string (a base64 data URL) is
// kept as it is. A text that merely starts with "data:" — an event stream a
// tool printed, a log line, a line the user wrote — went to the vendor with
// its key, was archived with it, and had its placeholder left in the answer.
func TestTextStartingWithDataIsMasked(t *testing.T) {
	const image = "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR4nGNgYGD4DwABBAEAwS2OUAAAAABJRU5ErkJggg=="
	for _, text := range []string{
		`data: {"api_key":"` + openaiKey + `"}` + "\n\n", // a tool's result: an SSE stream it fetched
		"data: request failed with key " + openaiKey,     // a log line
		"data: my key is " + openaiKey,                   // a user's line
		"data:text/plain," + openaiKey,                   // a data URL that is not base64 is text
		image + " " + openaiKey,                          // a data URL with more after it
	} {
		body, _ := json.Marshal(map[string]any{"messages": []any{
			map[string]any{"role": "tool", "tool_call_id": "call_1", "content": text},
			map[string]any{"role": "user", "content": []any{map[string]any{"type": "image_url", "image_url": map[string]any{"url": image}}}},
		}})
		out, n := MaskJSON(body, Options{Secrets: true})
		if n == 0 || strings.Contains(string(out), openaiKey) || !strings.Contains(string(out), image) {
			t.Errorf("%q: masked %d: %s", text, n, out)
		}
		if got := string(ScrubJSON(body)); strings.Contains(got, openaiKey) || !strings.Contains(got, image) {
			t.Errorf("%q archived as %s", text, got)
		}
	}

	// the answer: a placeholder in a text that starts with "data:" is put back
	p, _ := Mask(openaiKey, Options{Secrets: true})
	b, _ := json.Marshal(map[string]any{"choices": []any{map[string]any{"message": map[string]any{"role": "assistant", "content": "data: " + p}}}})
	rec := httptest.NewRecorder()
	rec.Header().Set("Content-Type", "text/event-stream")
	w := NewWriter(rec)
	w.WriteHeader(http.StatusOK)
	w.Write(append(append([]byte("data: "), b...), '\n', '\n'))
	w.Finish()
	if got := rec.Body.String(); !strings.Contains(got, "data: "+openaiKey) {
		t.Errorf("not put back: %s", got)
	}
}

// What fileData keeps: a base64 data URL with or without a media type,
// with parameters, in URL-safe base64; and nothing else.
func TestFileData(t *testing.T) {
	for s, want := range map[string]bool{
		"data:image/png;base64,iVBORw0KGgo=":                   true,
		"data:;base64,QUJD":                                    true,
		"data:image/svg+xml;charset=utf-8;base64,PHN2Zz4=":     true,
		"data:application/octet-stream;base64,-_-_":            true,
		"data:image/png;base64,":                               true,
		"data:text/plain,hello":                                false,
		"data: {\"a\":1}":                                      false,
		"data:image/png;base64,iVBORw0KGgo= and more":          false,
		"data:image/png;base64,iVBORw0KGgo=\n":                 false,
		"Data:image/png;base64,iVBORw0KGgo=":                   false,
		"the data:image/png;base64,iVBORw0KGgo= inside a text": false,
	} {
		if got := fileData(s); got != want {
			t.Errorf("fileData(%q) = %v, want %v", s, got, want)
		}
	}
}
