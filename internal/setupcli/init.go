package setupcli

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

func initCommand(args []string, out, stderr io.Writer) error {
	flags := flag.NewFlagSet("init", flag.ContinueOnError)
	flags.SetOutput(stderr)
	dir := flags.String("dir", ".", "installation directory")
	origin := flags.String("origin", "", "public HTTPS origin served by your reverse proxy")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if flags.NArg() != 0 {
		return errors.New("unexpected positional arguments")
	}
	parsed, err := url.Parse(*origin)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" || (parsed.Path != "" && parsed.Path != "/") || strings.ContainsAny(*origin, "\r\n\"'\\$`# ") {
		return errors.New("-origin must be the exact public HTTPS origin, without a path")
	}
	if err := os.MkdirAll(*dir, 0700); err != nil {
		return errors.New("could not create installation directory")
	}
	random := func() []byte {
		b := make([]byte, 32)
		if _, err := rand.Read(b); err != nil {
			panic("secure random source unavailable")
		}
		return b
	}
	rootPassword, runtimePassword, adminPassword := hex.EncodeToString(random()), hex.EncodeToString(random()), base64.RawURLEncoding.EncodeToString(random())
	runtimeURI := "mongodb://mlm_runtime:" + runtimePassword + "@mongo:27017/media_library?authSource=media_library&replicaSet=mlm"
	files := map[string]string{
		".env.prod":                 fmt.Sprintf("APP_ENV=production\nPUBLIC_ORIGIN=%s\nPORT=8080\nMONGODB_URI=%s\nMONGODB_DATABASE=media_library\nPROVIDER_KEYS_ENCRYPTION_KEY=%s\nSESSION_COOKIE_NAME=__Host-mlm_session\nSESSION_COOKIE_SECURE=true\nSIGNUP_ENABLED=false\nENSURE_INDEXES=false\nBACKUP_ENABLED=false\n", strings.TrimSuffix(*origin, "/"), runtimeURI, base64.StdEncoding.EncodeToString(random())),
		".env.setup":                fmt.Sprintf("MONGO_INITDB_ROOT_USERNAME=mlm_setup\nMONGO_INITDB_ROOT_PASSWORD=%s\nMONGODB_SETUP_URI=mongodb://mlm_setup:%s@mongo:27017/admin?directConnection=true\nMONGODB_RUNTIME_USERNAME=mlm_runtime\nMONGODB_RUNTIME_PASSWORD=%s\nBOOTSTRAP_ADMIN_USERNAME=admin\n", rootPassword, rootPassword, runtimePassword),
		".bootstrap-admin-password": adminPassword + "\n",
	}
	return writeConfiguration(*dir, files, out)
}
func writeConfiguration(dir string, files map[string]string, out io.Writer) error {
	// Do not overwrite even one existing configuration file or follow symlinks.
	for name := range files {
		if _, err := os.Lstat(filepath.Join(dir, name)); !errors.Is(err, os.ErrNotExist) {
			return errors.New("configuration already exists or cannot be inspected; nothing was overwritten")
		}
	}
	created := []string{}
	ok := false
	defer func() {
		if !ok {
			for _, name := range created {
				_ = os.Remove(name)
			}
		}
	}()
	for name, value := range files {
		path := filepath.Join(dir, name)
		file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if err != nil {
			return errors.New("could not create private configuration; incomplete files were removed")
		}
		created = append(created, path)
		_, err = file.WriteString(value)
		if err == nil {
			err = file.Sync()
		}
		closeErr := file.Close()
		if err != nil || closeErr != nil {
			return errors.New("could not save private configuration; incomplete files were removed")
		}
	}
	ok = true
	fmt.Fprintln(out, "Created .env.prod, .env.setup, and .bootstrap-admin-password with private permissions.\nKeep these files backed up securely. The initial admin password is in .bootstrap-admin-password; it was not printed.")
	return nil
}
