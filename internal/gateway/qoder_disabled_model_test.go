package gateway

import (
	"strings"
	"testing"

	"github.com/yetone/magpie/internal/provider"
)

// qoderUnknownModel400 is the qoder auth plugin's answer, with no upstream
// call, for a model the account's listing has off or lacks — as it does
// once the account's credits run out (#5726).
const qoderUnknownModel400 = `{"error":{"message":"unknown or disabled model \"dfmodel\"","type":"qoder_error","code":400}}`

// A Qoder member the plugin refuses the model for hands the chat request
// to the group's next one and rests, as the same refusal already does on
// /v1/messages (#5726). A 400 about the request itself still stops.
func TestGroupFailsOverQoderUnknownModelOnChat(t *testing.T) {
	fresh(t)
	q := &scripted{replies: []reply{{400, "", qoderUnknownModel400}}}
	b := &scripted{replies: []reply{{200, "", chatOK}}}
	serveOn(t, "qoder-test", "k", []string{"dfmodel"}, q)
	serveOn(t, "b", "k", []string{"m"}, b)
	if err := provider.SaveGroup(provider.Group{Name: "G", Members: []string{"qoder-test/dfmodel", "b/m"}, Routing: provider.Ordered}); err != nil {
		t.Fatal(err)
	}
	s := New()
	code, body := postAs(t, s, "", `{"model":"group/g","messages":[{"role":"user","content":"hi"}]}`)
	if code != 200 || !strings.Contains(body, "hello") || q.n != 1 || b.n != 1 {
		t.Fatalf("%d %s (qoder %d, b %d)", code, body, q.n, b.n)
	}
	r := s.trace.routes[len(s.trace.routes)-1]
	if len(r.Tries) != 2 || r.Tries[0].Status != 400 || r.Tries[0].Fail != failOther {
		t.Fatalf("tries %+v", r.Tries)
	}
	if rest := r.Tries[0].Rest; rest == nil || rest.Key != "qoder-test/dfmodel" || rest.By != "backoff" {
		t.Fatalf("the refused model was to rest with backoff: %+v", r.Tries[0].Rest)
	}

	// the rested model is passed over: the next request is answered by the
	// second member alone
	code, body = postAs(t, s, "", `{"model":"group/g","messages":[{"role":"user","content":"hi"}]}`)
	if code != 200 || !strings.Contains(body, "hello") || q.n != 1 || b.n != 2 {
		t.Fatalf("next: %d %s (qoder %d, b %d)", code, body, q.n, b.n)
	}

	if !retryable(400, []byte(qoderUnknownModel400)) {
		t.Error("retryable false: the chat path would hand the 400 to the agent")
	}
	if !modelRefused(400, []byte(qoderUnknownModel400)) {
		t.Error("modelRefused false: the model wouldn't rest alone on its account")
	}
}

// A 400 that names the request itself keeps its old treatment on the chat
// path: the group's next member isn't asked.
func TestGroupKeepsPlain400OnChat(t *testing.T) {
	fresh(t)
	a := &scripted{replies: []reply{{400, "", `{"error":{"message":"messages: field required"}}`}}}
	b := &scripted{replies: []reply{{200, "", chatOK}}}
	serveOn(t, "a", "k", []string{"m"}, a)
	serveOn(t, "b", "k", []string{"m"}, b)
	if err := provider.SaveGroup(provider.Group{Name: "G", Members: []string{"a/m", "b/m"}, Routing: provider.Ordered}); err != nil {
		t.Fatal(err)
	}
	s := New()
	code, body := postAs(t, s, "", `{"model":"group/g","messages":[{"role":"user","content":"hi"}]}`)
	if code != 400 || a.n != 1 || b.n != 0 {
		t.Fatalf("%d %s (a %d, b %d)", code, body, a.n, b.n)
	}
}
