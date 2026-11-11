package setupcli

import (
	"bytes"
	"github.com/joho/godotenv"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInitCreatesPrivateDistinctCredentialsWithoutPrintingThem(t *testing.T) {
	dir := t.TempDir()
	var out bytes.Buffer
	if err := Run([]string{"init", "-origin", "https://library.example", "-dir", dir}, strings.NewReader(""), &out, &out, "test"); err != nil {
		t.Fatal(err)
	}
	prod, err := godotenv.Read(filepath.Join(dir, ".env.prod"))
	if err != nil {
		t.Fatal(err)
	}
	setup, err := godotenv.Read(filepath.Join(dir, ".env.setup"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(prod["MONGODB_URI"], "mlm_setup") || setup["MONGODB_SETUP_URI"] == prod["MONGODB_URI"] {
		t.Fatal("runtime received maintenance credentials")
	}
	admin, err := os.ReadFile(filepath.Join(dir, ".bootstrap-admin-password"))
	if err != nil {
		t.Fatal(err)
	}
	for _, secret := range []string{prod["PROVIDER_KEYS_ENCRYPTION_KEY"], setup["MONGO_INITDB_ROOT_PASSWORD"], setup["MONGODB_RUNTIME_PASSWORD"], strings.TrimSpace(string(admin))} {
		if secret == "" || strings.Contains(out.String(), secret) {
			t.Fatal("secret missing or printed")
		}
	}
	for _, name := range []string{".env.prod", ".env.setup", ".bootstrap-admin-password"} {
		stat, err := os.Stat(filepath.Join(dir, name))
		if err != nil || stat.Mode().Perm() != 0600 {
			t.Fatal("private configuration permissions not enforced")
		}
	}
	before, _ := os.ReadFile(filepath.Join(dir, ".env.prod"))
	if err := Run([]string{"init", "-origin", "https://other.example", "-dir", dir}, nil, &out, &out, "test"); err == nil {
		t.Fatal("existing configuration overwritten")
	}
	after, _ := os.ReadFile(filepath.Join(dir, ".env.prod"))
	if string(before) != string(after) {
		t.Fatal("existing file changed")
	}
}
func TestInitRejectsUnsafeOriginsAndExistingSymlinks(t *testing.T) {
	for _, origin := range []string{"http://library.example", "https://library.example/path", "https://user:password@library.example", "https://library.example\nX=1", "https://library.example#$VAR"} {
		var out bytes.Buffer
		if err := Run([]string{"init", "-origin", origin, "-dir", t.TempDir()}, nil, &out, &out, ""); err == nil {
			t.Fatalf("unsafe origin accepted: %q", origin)
		}
	}
	dir := t.TempDir()
	target := filepath.Join(t.TempDir(), "existing")
	_ = os.WriteFile(target, []byte("keep"), 0600)
	if err := os.Symlink(target, filepath.Join(dir, ".env.prod")); err != nil {
		t.Skip("symlinks unavailable")
	}
	var out bytes.Buffer
	if err := Run([]string{"init", "-origin", "https://library.example", "-dir", dir}, nil, &out, &out, ""); err == nil {
		t.Fatal("symlink followed")
	}
	data, _ := os.ReadFile(target)
	if string(data) != "keep" {
		t.Fatal("symlink target changed")
	}
}
func TestPasswordsAndMaintenanceConfigAreBounded(t *testing.T) {
	for _, value := range []string{"short", strings.Repeat("a", 73), strings.Repeat("a", 100000)} {
		if _, err := readPassword(strings.NewReader(value)); err == nil {
			t.Fatal("invalid password accepted")
		}
	}
	if p, err := readPassword(strings.NewReader("fixture-password\r\n")); err != nil || p != "fixture-password" {
		t.Fatal("CRLF handling failed")
	}
	t.Setenv("MONGODB_URI", "")
	t.Setenv("MONGODB_SETUP_URI", "")
	t.Setenv("MONGODB_DATABASE", "")
	var out bytes.Buffer
	if err := Run([]string{"database"}, nil, &out, &out, "test"); err == nil || !strings.Contains(err.Error(), "required") {
		t.Fatal("missing database configuration accepted")
	}
	if err := Run([]string{"version"}, nil, &out, &out, "fixture-version"); err != nil || !strings.Contains(out.String(), "fixture-version") {
		t.Fatal("version unavailable")
	}
}
