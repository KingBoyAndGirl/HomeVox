package floorplan

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/KingBoyAndGirl/HomeVox/backend/internal/ai"
)

const canonicalParseJSON = `{"rooms":[{"name":"Living","type":"living","approximate_bounds":{"x1":0,"y1":0,"x2":100,"y2":80},"area_ratio":0.5}],"walls":[{"id":"wall-1","x1":0,"y1":0,"x2":200,"y2":0}],"doors":[{"id":"door-1","kind":"door","wallId":"wall-1","position":0.5,"width":40,"source":"ai","confirmed":false}],"windows":[{"id":"window-1","kind":"window","wallId":"wall-1","position":0.2,"width":20,"source":"ai","confirmed":true}],"scale":{"unit":"px","pixel_to_unit":1},"metadata":{"source":"fake","confidence":0.9,"image_width":200,"image_height":80}}`

func TestFirstChoiceContent(t *testing.T) {
	got, err := firstChoiceContent(map[string]any{
		"choices": []any{map[string]any{"message": map[string]any{"content": `{"rooms":[]}`}}},
	})
	if err != nil {
		t.Fatalf("firstChoiceContent returned error: %v", err)
	}
	if got != `{"rooms":[]}` {
		t.Fatalf("content = %q", got)
	}
}

func TestFirstChoiceContentRejectsMissingChoices(t *testing.T) {
	if _, err := firstChoiceContent(map[string]any{}); err == nil {
		t.Fatal("expected error")
	}
}

func TestParseUsesOpenAICompatibleVisionContractAndRejectsInvalidOpeningGeometry(t *testing.T) {
	body := strings.Replace(canonicalParseJSON, `"position":0.5,"width":40`, `"position":0.5,"width":240`, 1)
	server := visionServer(t, body, func(r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Fatalf("authorization = %q", got)
		}
	})
	defer server.Close()
	_, err := NewParser(ai.NewClient(server.URL+"/v1", "test-key", "vision-test")).Parse(context.Background(), "data:image/png;base64,cG5n")
	if err == nil || !strings.Contains(err.Error(), "exceeds wall endpoints") {
		t.Fatalf("error = %v", err)
	}
}

func TestParseRejectsInvalidOpenAIEnvelopes(t *testing.T) {
	for name, body := range map[string]string{
		"non JSON": `not-json`, "empty choices": `{"choices":[]}`, "empty content": `{"choices":[{"message":{"content":""}}]}`,
		"markdown fence": "{\"choices\":[{\"message\":{\"content\":\"```json\\n{}\\n```\"}}]}",
	} {
		t.Run(name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(body)) }))
			defer server.Close()
			if _, err := NewParser(ai.NewClient(server.URL, "test-key", "vision-test")).Parse(context.Background(), "data:image/png;base64,cG5n"); err == nil {
				t.Fatal("expected fail-closed parser error")
			}
		})
	}
}

func TestParseRejectsNonCanonicalAIJSON(t *testing.T) {
	valid := canonicalParseJSON
	cases := map[string]string{
		"unknown top level":  strings.Replace(valid, `"rooms":`, `"extra":true,"rooms":`, 1),
		"unknown nested":     strings.Replace(valid, `"x1":0,"y1":0,"x2":100`, `"x1":0,"unexpected":1,"y1":0,"x2":100`, 1),
		"missing collection": strings.Replace(valid, `"windows":[{"id":"window-1","kind":"window","wallId":"wall-1","position":0.2,"width":20,"source":"ai","confirmed":true}],`, ``, 1),
		"missing object":     strings.Replace(valid, `,"scale":{"unit":"px","pixel_to_unit":1}`, ``, 1),
		"missing scalar":     strings.Replace(valid, `,"confidence":0.9`, ``, 1),
		"partial room":       strings.Replace(valid, `,"area_ratio":0.5`, ``, 1),
		"partial wall":       strings.Replace(valid, `,"y2":0}`, `}`, 1),
		"partial opening":    strings.Replace(valid, `,"source":"ai"`, ``, 1),
		"partial scale":      strings.Replace(valid, `,"pixel_to_unit":1`, ``, 1),
		"partial metadata":   strings.Replace(valid, `,"image_height":80`, ``, 1),
		"null object":        strings.Replace(valid, `"scale":{"unit":"px","pixel_to_unit":1}`, `"scale":null`, 1),
		"null array":         strings.Replace(valid, `"doors":[`, `"doors":null`, 1),
		"wrong type":         strings.Replace(valid, `"width":40`, `"width":"40"`, 1),
		"duplicate key":      strings.Replace(valid, `"id":"wall-1"`, `"id":"wall-1","id":"wall-2"`, 1),
		"trailing JSON":      valid + ` {"ignored":true}`,
		"legacy opening":     strings.Replace(valid, `"confirmed":false`, `"confirmed":false,"type":"door"`, 1),
	}
	for name, content := range cases {
		t.Run(name, func(t *testing.T) {
			server := visionServer(t, content, nil)
			defer server.Close()
			_, err := NewParser(ai.NewClient(server.URL+"/v1", "test-key", "vision-test")).Parse(context.Background(), "data:image/png;base64,cG5n")
			if err == nil {
				t.Fatalf("accepted malformed canonical JSON: %s", content)
			}
		})
	}
}

func TestParseAcceptsOnlyCompleteCanonicalAIJSON(t *testing.T) {
	server := visionServer(t, canonicalParseJSON, nil)
	defer server.Close()
	result, err := NewParser(ai.NewClient(server.URL+"/v1", "test-key", "vision-test")).Parse(context.Background(), "data:image/png;base64,cG5n")
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if result.Walls[0].ID != "wall-1" || result.Doors[0].Source != "ai" || result.Scale.PixelToUnit != 1 {
		t.Fatalf("unexpected canonical result: %#v", result)
	}
}

func visionServer(t *testing.T, content string, check func(*http.Request)) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if check != nil {
			check(r)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{"choices":[{"message":{"content":%q}}]}`, content)
	}))
}
