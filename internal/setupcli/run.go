package setupcli

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"golang.org/x/term"
	"media_library_manager/internal/db"
	"media_library_manager/internal/dbsetup"
)

func Run(args []string, input io.Reader, out, stderr io.Writer, version string) error {
	if len(args) == 0 || args[0] == "help" || args[0] == "--help" {
		fmt.Fprintln(out, "mlm-setup init -origin https://library.example.com [-dir .]\nmlm-setup database [-env-file setup.env] [-init-replica -provision-runtime]\nmlm-setup bootstrap-admin [-env-file setup.env] [-username admin] [-password-stdin]\nmlm-setup version")
		return nil
	}
	if args[0] == "version" {
		fmt.Fprintln(out, version)
		return nil
	}
	if args[0] == "init" {
		return initCommand(args[1:], out, stderr)
	}
	if args[0] != "database" && args[0] != "bootstrap-admin" {
		return errors.New("unknown setup command; run mlm-setup help")
	}
	flags := flag.NewFlagSet(args[0], flag.ContinueOnError)
	flags.SetOutput(stderr)
	envFile := flags.String("env-file", "", "private environment file (optional; process environment takes precedence)")
	replica := flags.Bool("init-replica", false, "initialize bundled MongoDB replica set")
	provision := flags.Bool("provision-runtime", false, "create bundled runtime database role and user")
	username := flags.String("username", "", "initial application administrator username")
	passwordStdin := flags.Bool("password-stdin", false, "read administrator password from piped stdin")
	passwordFile := flags.String("admin-password-file", "", "initial administrator password file (database command only)")
	if err := flags.Parse(args[1:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if flags.NArg() != 0 {
		return errors.New("unexpected positional arguments")
	}
	if args[0] == "bootstrap-admin" && (*replica || *provision || *passwordFile != "") {
		return errors.New("database configuration flags cannot be used with bootstrap-admin")
	}
	values := map[string]string{}
	if *envFile != "" {
		var err error
		values, err = godotenv.Read(*envFile)
		if err != nil {
			return errors.New("could not read setup environment file")
		}
	}
	get := func(key string) string {
		if value, ok := os.LookupEnv(key); ok {
			return strings.TrimSpace(value)
		}
		return strings.TrimSpace(values[key])
	}
	uri := get("MONGODB_SETUP_URI")
	if uri == "" {
		uri = get("MONGODB_URI")
	}
	name := get("MONGODB_DATABASE")
	if uri == "" || name == "" {
		return errors.New("MONGODB_URI (or MONGODB_SETUP_URI) and MONGODB_DATABASE are required")
	}
	if *username == "" {
		*username = get("BOOTSTRAP_ADMIN_USERNAME")
		if *username == "" {
			*username = "admin"
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	client, err := db.ConnectMongo(ctx, uri)
	if err != nil {
		return errors.New("database connection failed; check credentials and network access")
	}
	defer client.Disconnect(context.Background())
	if *replica {
		if err := dbsetup.Replica(ctx, client); err != nil {
			return err
		}
	} else if err := dbsetup.WaitPrimary(ctx, client); err != nil {
		return err
	}
	database := client.Database(name)
	if err := dbsetup.Initialize(ctx, database); err != nil {
		return errors.New("database initialization failed; check maintenance permissions")
	}
	if *provision {
		if err := dbsetup.RuntimeUser(ctx, database, get("MONGODB_RUNTIME_USERNAME"), get("MONGODB_RUNTIME_PASSWORD")); err != nil {
			return err
		}
		// Detect mismatched saved credentials without resetting an existing password.
		runtimeURI := get("MONGODB_URI")
		if runtimeURI == "" || runtimeURI == uri {
			return errors.New("a separate runtime MONGODB_URI is required")
		}
		runtimeClient, err := db.ConnectMongo(ctx, runtimeURI)
		if err != nil {
			return errors.New("runtime database authentication failed; restore the original configuration instead of regenerating credentials")
		}
		_ = runtimeClient.Disconnect(context.Background())
	}
	fmt.Fprintln(out, "Database collections and indexes are ready.")
	if args[0] == "bootstrap-admin" || *passwordFile != "" {
		var password string
		if *passwordFile != "" {
			if args[0] != "database" {
				return errors.New("password file is only supported by database bootstrap")
			}
			f, err := os.Open(*passwordFile)
			if err != nil {
				return errors.New("could not read bootstrap password file")
			}
			password, err = readPassword(f)
			_ = f.Close()
			if err != nil {
				return err
			}
		} else if *passwordStdin {
			if f, ok := input.(*os.File); ok && term.IsTerminal(int(f.Fd())) {
				return errors.New("pipe the password or use the hidden interactive prompt")
			}
			password, err = readPassword(input)
			if err != nil {
				return err
			}
		} else {
			f, ok := input.(*os.File)
			if !ok || !term.IsTerminal(int(f.Fd())) {
				return errors.New("interactive terminal required; use -password-stdin for automation")
			}
			fmt.Fprint(stderr, "Administrator password: ")
			raw, err := term.ReadPassword(int(f.Fd()))
			fmt.Fprintln(stderr)
			if err != nil {
				return errors.New("could not read password")
			}
			fmt.Fprint(stderr, "Confirm password: ")
			again, err := term.ReadPassword(int(f.Fd()))
			fmt.Fprintln(stderr)
			if err != nil || string(raw) != string(again) {
				return errors.New("passwords did not match")
			}
			password = string(raw)
		}
		created, err := dbsetup.BootstrapAdmin(ctx, database, *username, password)
		if err != nil {
			return errors.New("administrator bootstrap failed; existing accounts were not changed")
		}
		if created {
			fmt.Fprintln(out, "Initial administrator created.")
		} else {
			fmt.Fprintln(out, "An administrator already exists; account credentials were left unchanged.")
		}
	}
	return nil
}
func readPassword(input io.Reader) (string, error) {
	// Read at most a bcrypt-sized password and a possible CRLF, never unbounded input.
	reader := bufio.NewReader(io.LimitReader(input, 75))
	raw, err := io.ReadAll(reader)
	if err != nil {
		return "", errors.New("could not read password")
	}
	password := strings.TrimSuffix(strings.TrimSuffix(string(raw), "\n"), "\r")
	if len(password) < 8 || len(password) > 72 {
		return "", errors.New("password must be 8 to 72 bytes")
	}
	return password, nil
}
