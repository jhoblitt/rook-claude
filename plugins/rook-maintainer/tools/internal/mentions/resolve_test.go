package mentions

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// stubLookup answers from a fixed directory and records what it was asked, so
// the caching and hyphen-retry rules are observable without a network call.
// Like the real API it is case-insensitive and answers with the canonical
// spelling.
func stubLookup(users map[string]string, calls *[]string) LoginResolver {
	return func(_ context.Context, token string) (string, bool, error) {
		*calls = append(*calls, token)
		login, ok := users[strings.ToLower(token)]
		return login, ok, nil
	}
}

func TestResolve(t *testing.T) {
	users := map[string]string{"alice": "Alice", "sinner-": "sinner-", "bob": "bob"}

	tests := []struct {
		name, token, want string
		wantCalls         []string
	}{
		{"exact", "alice", "Alice", []string{"alice"}},
		{"canonical case wins", "ALICE", "Alice", []string{"ALICE"}},
		{"legacy trailing hyphen tried verbatim first", "sinner-", "sinner-",
			[]string{"sinner-"}},
		{"trailing hyphen stripped on retry", "bob-", "bob", []string{"bob-", "bob"}},
		{"all hyphens stripped", "bob---", "bob", []string{"bob---", "bob"}},
		{"unresolvable", "nope", "", []string{"nope"}},
	}
	for _, tc := range tests {
		var calls []string
		got, err := Resolve(context.Background(), tc.token, NewCache(),
			stubLookup(users, &calls))
		if err != nil {
			t.Errorf("%s: %v", tc.name, err)
			continue
		}
		if got != tc.want {
			t.Errorf("%s: Resolve(%q) = %q, want %q", tc.name, tc.token, got, tc.want)
		}
		if !slices.Equal(calls, tc.wantCalls) {
			t.Errorf("%s: looked up %v, want %v", tc.name, calls, tc.wantCalls)
		}
	}
}

func TestResolveCachesBothOutcomes(t *testing.T) {
	var calls []string
	cache := NewCache()
	lookup := stubLookup(map[string]string{"alice": "Alice"}, &calls)

	for _, tok := range []string{"alice", "ALICE", "Alice", "ghost", "GHOST"} {
		if _, err := Resolve(context.Background(), tok, cache, lookup); err != nil {
			t.Fatal(err)
		}
	}
	if want := []string{"alice", "ghost"}; !slices.Equal(calls, want) {
		t.Errorf("looked up %v, want %v", calls, want)
	}
	if got := cache.Unresolvable(); got != 1 {
		t.Errorf("Unresolvable() = %d, want 1", got)
	}
}

func TestResolveDoesNotCacheAToolFailure(t *testing.T) {
	boom := errors.New("gh is missing")
	cache := NewCache()
	lookup := func(context.Context, string) (string, bool, error) { return "", false, boom }

	if _, err := Resolve(context.Background(), "alice", cache, lookup); !errors.Is(err, boom) {
		t.Fatalf("err = %v, want %v", err, boom)
	}
	if _, ok := cache.Get("alice"); ok {
		t.Error("a failed lookup was cached as an absent user")
	}
}

func TestCacheRoundTripAndFormat(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cache.json")
	alice := "Alice"
	c := NewCache()
	c.Set("alice", &alice)
	c.Set("ghost", nil)
	if err := c.Save(path); err != nil {
		t.Fatal(err)
	}

	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	want := "{\n \"alice\": \"Alice\",\n \"ghost\": null\n}"
	if string(b) != want {
		t.Errorf("cache = %q, want %q", b, want)
	}

	back, err := LoadCache(path)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(back.keys, []string{"alice", "ghost"}) {
		t.Errorf("key order = %v", back.keys)
	}
	if v, ok := back.Get("alice"); !ok || v == nil || *v != "Alice" {
		t.Errorf("alice round-tripped as %v", v)
	}
	if v, ok := back.Get("ghost"); !ok || v != nil {
		t.Errorf("ghost round-tripped as %v, want a remembered miss", v)
	}
}

func TestLoadCacheMissingFile(t *testing.T) {
	c, err := LoadCache(filepath.Join(t.TempDir(), "absent.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(c.keys) != 0 {
		t.Errorf("keys = %v, want none", c.keys)
	}
}

func TestEmptyCacheSavesAsAnEmptyObject(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cache.json")
	if err := NewCache().Save(path); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "{}" {
		t.Errorf("cache = %q, want {}", b)
	}
}
