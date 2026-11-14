package config

import (
	"os"
	"strings"
	"testing"

	"github.com/chaindead/telegram-mcp/internal/secret"
)

func TestLoadMissing(t *testing.T) {
	c, err := Load(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if *c != (Config{}) {
		t.Errorf("Load() = %+v, want empty", *c)
	}
}

func TestSaveLoad(t *testing.T) {
	key, _ := secret.Generate()
	hash, err := key.Encrypt([]byte("0123456789abcdef0123456789abcdef"))
	if err != nil {
		t.Fatal(err)
	}

	dir := t.TempDir()
	if err := (&Config{APIID: 12345, APIHash: hash, DefaultAccount: 3017676}).Save(dir); err != nil {
		t.Fatal(err)
	}

	raw, err := os.ReadFile(Path(dir))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "api_hash = '''\n-----BEGIN AGE ENCRYPTED FILE-----\n") {
		t.Errorf("api_hash is not a multi-line string:\n%s", raw)
	}

	c, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if c.APIID != 12345 || c.APIHash != hash || c.DefaultAccount != 3017676 {
		t.Errorf("Load() = %+v, want what was saved", *c)
	}
}
