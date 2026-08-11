package mentions

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/jhoblitt/rook-claude/plugins/rook-maintainer/tools/internal/ghx"
)

// LoginResolver answers whether a token names a real user, and under which
// canonical spelling. found=false is a definitive "no such user" and gets
// cached; an error is a broken toolchain and must not be, or one missing `gh`
// poisons the cache with absent users forever.
type LoginResolver func(ctx context.Context, token string) (login string, found bool, err error)

// GHLogin is the live resolver. The lookup carries no timeout because a slow
// answer is still an answer, while a timed-out one would be indistinguishable
// from an absent user.
func GHLogin(ctx context.Context, token string) (string, bool, error) {
	out, err := ghx.Run(ctx, 0, "api", "users/"+token, "--jq", ".login")
	if err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return "", false, err
		}
		return "", false, nil
	}
	return strings.TrimSpace(string(out)), true, nil
}

// Cache is the on-disk record of token -> canonical login, keyed by the
// lower-cased token. A nil login is a remembered miss.
type Cache struct {
	keys []string
	vals map[string]*string
}

func NewCache() *Cache {
	return &Cache{vals: make(map[string]*string)}
}

// LoadCache reads path, treating a missing file as an empty cache.
func LoadCache(path string) (*Cache, error) {
	c := NewCache()
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return c, nil
	}
	if err != nil {
		return nil, err
	}
	err = decodeObject(b, func(key string, dec *json.Decoder) error {
		var login *string
		if err := dec.Decode(&login); err != nil {
			return err
		}
		c.Set(key, login)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return c, nil
}

func (c *Cache) Get(key string) (*string, bool) {
	v, ok := c.vals[key]
	return v, ok
}

func (c *Cache) Set(key string, login *string) {
	if _, ok := c.vals[key]; !ok {
		c.keys = append(c.keys, key)
	}
	c.vals[key] = login
}

// Unresolvable counts the tokens remembered as naming no user.
func (c *Cache) Unresolvable() int {
	n := 0
	for _, k := range c.keys {
		if v := c.vals[k]; v == nil || *v == "" {
			n++
		}
	}
	return n
}

func (c *Cache) Save(path string) error {
	b, err := indentedObject(c.keys, func(k string) any { return c.vals[k] })
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}

// Resolve maps one mention token to a canonical login, "" when it names no
// user. The trailing-hyphen retry exists because a mention swallows a trailing
// hyphen that usually belongs to the prose — but not always, since legacy
// logins like `sinner-` are real, so the token is tried verbatim first.
func Resolve(ctx context.Context, token string, cache *Cache, lookup LoginResolver) (string, error) {
	key := strings.ToLower(token)
	if v, ok := cache.Get(key); ok {
		if v == nil {
			return "", nil
		}
		return *v, nil
	}
	var resolved *string
	for _, cand := range lookupOrder(token) {
		login, found, err := lookup(ctx, cand)
		if err != nil {
			return "", err
		}
		if found {
			resolved = &login
			break
		}
	}
	cache.Set(key, resolved)
	if resolved == nil {
		return "", nil
	}
	return *resolved, nil
}

func lookupOrder(token string) []string {
	var out []string
	for _, cand := range []string{token, strings.TrimRight(token, "-")} {
		if cand != "" && (len(out) == 0 || out[0] != cand) {
			out = append(out, cand)
		}
	}
	return out
}
