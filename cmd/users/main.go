// users changes account roles through trusted server access only.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"media_library_manager/internal/config"
	"media_library_manager/internal/db"
	domainauth "media_library_manager/internal/domain/auth"
	"media_library_manager/internal/repository"
)

func main() {
	username := flag.String("username", "", "existing account username")
	role := flag.String("role", "", "admin, general, or demo")
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
	if err := repository.NewUsersRepository(client.Database(cfg.MongoDatabase)).SetRole(ctx, strings.ToLower(strings.TrimSpace(*username)), *role); err != nil {
		fail(err)
	}
	fmt.Printf("Role updated to %s. It takes effect on the account's next request.\n", *role)
}
func fail(err error) { fmt.Fprintln(os.Stderr, err); os.Exit(1) }
