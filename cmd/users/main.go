// users changes account roles through trusted server access only.
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"media_library_manager/internal/config"
	"media_library_manager/internal/db"
	domainauth "media_library_manager/internal/domain/auth"
	"media_library_manager/internal/repository"
	authsvc "media_library_manager/internal/service/auth"
)

func main() {
	username := flag.String("username", "", "existing account username")
	role := flag.String("role", "", "admin, general, or demo")
	create := flag.Bool("create", false, "create a new account instead of updating its role")
	passwordStdin := flag.Bool("password-stdin", false, "read a new account password from piped standard input")
	flag.Parse()
	if strings.TrimSpace(*username) == "" || !domainauth.IsRole(*role) {
		fmt.Fprintln(os.Stderr, "Usage: go run ./cmd/users -username <username> -role admin|general|demo")
		os.Exit(2)
	}
	cfg, err := config.Load()
	if err != nil {
		fail(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), cfg.MongoConnectTimeout())
	defer cancel()
	client, err := db.ConnectMongo(ctx, cfg.MongoURI)
	if err != nil {
		fail(err)
	}
	defer client.Disconnect(context.Background())
	users := repository.NewUsersRepository(client.Database(cfg.MongoDatabase))
	if *create {
		if !*passwordStdin {
			fail(fmt.Errorf("account creation requires -password-stdin"))
		}
		stat, err := os.Stdin.Stat()
		if err != nil || stat.Mode()&os.ModeCharDevice != 0 {
			fail(fmt.Errorf("pipe the password from a private source; interactive echo is disabled"))
		}
		raw, err := io.ReadAll(io.LimitReader(os.Stdin, 75))
		if err != nil {
			fail(fmt.Errorf("could not read password"))
		}
		password := strings.TrimSuffix(strings.TrimSuffix(string(raw), "\n"), "\r")
		auth := authsvc.NewService(users, repository.NewSessionsRepository(client.Database(cfg.MongoDatabase)), time.Hour)
		if _, err := auth.CreateAccount(ctx, *username, password, "", *role); err != nil {
			fail(err)
		}
		fmt.Println("Account created.")
		return
	}
	if err := users.SetRole(ctx, strings.ToLower(strings.TrimSpace(*username)), *role); err != nil {
		fail(err)
	}
	fmt.Printf("Role updated to %s. It takes effect on the account's next request.\n", *role)
}
func fail(err error) { fmt.Fprintln(os.Stderr, err); os.Exit(1) }
