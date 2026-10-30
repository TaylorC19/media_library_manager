package providerkeys

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestPrivatePersistenceAndAccountIsolation(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "keys")
	store := NewStore(dir)
	alice, bob := primitive.NewObjectID(), primitive.NewObjectID()
	if keys, err := store.Get(alice); err != nil || len(keys) != 0 {
		t.Fatal("missing file should use defaults")
	}
	if err := store.Update(alice, Keys{"tmdb": "alice-key", "rawg": "game-key"}); err != nil {
		t.Fatal(err)
	}
	if err := store.Update(bob, Keys{"tmdb": "bob-key"}); err != nil {
		t.Fatal(err)
	}
	if err := store.Update(alice, Keys{"tmdb": "replacement", "rawg": ""}); err != nil {
		t.Fatal(err)
	}
	restored := NewStore(dir)
	keys, err := restored.Get(alice)
	if err != nil {
		t.Fatal(err)
	}
	if keys["tmdb"] != "replacement" || keys["rawg"] != "" {
		t.Error("replacement/removal did not persist")
	}
	bobKeys, err := restored.Get(bob)
	if err != nil {
		t.Fatal(err)
	}
	if bobKeys["tmdb"] != "bob-key" {
		t.Error("changed another account's key")
	}
	for path, want := range map[string]os.FileMode{dir: 0700, filepath.Join(dir, alice.Hex()+".json"): 0600} {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm() != want {
			t.Errorf("permissions=%v, want %v", info.Mode().Perm(), want)
		}
	}
}

func TestRequestCredentialsAreImmutableAndIsolated(t *testing.T) {
	keys := Keys{"tmdb": "personal"}
	ctx := WithKeys(context.Background(), keys)
	keys["tmdb"] = "changed"
	if Resolve(ctx, "tmdb", "server") != "personal" {
		t.Error("context must snapshot credentials")
	}
	if Resolve(ctx, "rawg", "server") != "server" {
		t.Error("missing override must use server default")
	}
	if Resolve(context.Background(), "tmdb", "server") != "server" {
		t.Error("default context was mutated")
	}
	var pending sync.WaitGroup
	for _, value := range []string{"alice", "bob"} {
		pending.Add(1)
		go func(key string) {
			defer pending.Done()
			own := WithKeys(context.Background(), Keys{"tmdb": key})
			for i := 0; i < 100; i++ {
				if Resolve(own, "tmdb", "") != key {
					t.Error("credentials crossed requests")
				}
			}
		}(value)
	}
	pending.Wait()
}
