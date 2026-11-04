package gateway

import (
	"strings"
	"testing"

	"github.com/tidwall/gjson"
	"github.com/yetone/magpie/internal/provider"
)

// A Messages request translated to a Chat-only provider keeps the
// client's cache_control breakpoints (#5125): Claude Code on a Chat relay
// saw 0 cache writes and 0 reads, its system mark dropped in the
// translation, while the same body on /v1/chat/completions was cached.
func TestMessagesToChatKeepsCacheControl(t *testing.T) {
	f := &fake{t: t, reply: chatReply}
	setup(t, provider.Chat, f)
	body := `{"model":"m1","max_tokens":16,
		"system":[
			{"type":"text","text":"x-anthropic-billing-header: cc_version=2.1.0; cch=abc;"},
			{"type":"text","text":"you are helpful","cache_control":{"type":"ephemeral","ttl":"1h"}},
			{"type":"text","text":" and careful","cache_control":{"type":"ephemeral","ttl":"1h"}}],
		"messages":[
			{"role":"user","content":[{"type":"text","text":"one","cache_control":{"type":"ephemeral"}}]},
			{"role":"assistant","content":[{"type":"text","text":"ok"}]},
			{"role":"user","content":[{"type":"text","text":"two","cache_control":{"type":"ephemeral"}}]}]}`
	code, reply := post(t, "/v1/messages", body)
	if code != 200 {
		t.Fatalf("status %d: %s", code, reply)
	}

	system := gjson.GetBytes(f.got, `messages.#(role=="system").content`)
	if !system.IsArray() || len(system.Array()) != 2 {
		t.Fatalf("system content is not two marked parts: %s", f.got)
	}
	for i, want := range []string{"you are helpful", " and careful"} {
		part := system.Array()[i]
		if part.Get("text").String() != want || part.Get("cache_control.type").String() != "ephemeral" || part.Get("cache_control.ttl").String() != "1h" {
			t.Errorf("system part %d in %s", i, f.got)
		}
	}
	if strings.Contains(string(f.got), "x-anthropic-billing-header") {
		t.Errorf("billing line kept: %s", f.got)
	}

	first := gjson.GetBytes(f.got, `messages.#(role=="user").content.0`)
	if first.Get("cache_control.type").String() != "ephemeral" {
		t.Errorf("first user mark lost: %s", f.got)
	}
	users := gjson.GetBytes(f.got, `messages.#(role=="user")#.content.0`)
	if len(users.Array()) != 2 || users.Array()[1].Get("cache_control.type").String() != "ephemeral" {
		t.Errorf("last user mark lost: %s", f.got)
	}
	if got := gjson.GetBytes(f.got, `messages.#(role=="assistant").content`); got.Type != gjson.String || got.String() != "ok" {
		t.Errorf("assistant text changed: %s", f.got)
	}
}

// A Messages request without a mark gains none when it is translated: the
// system and user content stay the plain strings they are today.
func TestMessagesToChatAddsNoCacheControl(t *testing.T) {
	f := &fake{t: t, reply: chatReply}
	setup(t, provider.Chat, f)
	code, reply := post(t, "/v1/messages", `{"model":"m1","max_tokens":16,
		"system":[{"type":"text","text":"you are helpful"}],
		"messages":[{"role":"user","content":"hi"}]}`)
	if code != 200 {
		t.Fatalf("status %d: %s", code, reply)
	}
	if strings.Contains(string(f.got), "cache_control") {
		t.Errorf("a mark appeared: %s", f.got)
	}
	if got := gjson.GetBytes(f.got, `messages.#(role=="system").content`); got.Type != gjson.String {
		t.Errorf("system content is not a string: %s", f.got)
	}
	if got := gjson.GetBytes(f.got, `messages.#(role=="user").content`); got.Type != gjson.String {
		t.Errorf("user content is not a string: %s", f.got)
	}
}
