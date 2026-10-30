package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/mongo"

	"media_library_manager/internal/backup"
	"media_library_manager/internal/config"
	"media_library_manager/internal/db"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(2)
	}

	switch os.Args[1] {
	case "export":
		os.Exit(runExport(os.Args[2:]))
	case "import":
		os.Exit(runImport(os.Args[2:]))
	default:
		printUsage()
		os.Exit(2)
	}
}

func printUsage() {
	fmt.Fprintf(os.Stderr, "Usage:\n")
	fmt.Fprintf(os.Stderr, "  %s export -o <file.mlm-backup.json.gz>\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "  %s import -i <file.mlm-backup.json.gz> --confirm <database_name>\n", os.Args[0])
}

func runExport(args []string) int {
	fs := flag.NewFlagSet("export", flag.ExitOnError)
	out := fs.String("o", "", "output backup file path")
	_ = fs.Parse(args)
	if *out == "" {
		fmt.Fprintln(os.Stderr, "export requires -o")
		return 2
	}

	cfg, database, cleanup, err := connect()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	data, err := backup.ExportDatabase(ctx, database)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if err := os.WriteFile(*out, data, 0o600); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	fmt.Printf("exported database %q to %s (%d bytes)\n", cfg.MongoDatabase, *out, len(data))
	return 0
}

func runImport(args []string) int {
	fs := flag.NewFlagSet("import", flag.ExitOnError)
	in := fs.String("i", "", "input backup file path")
	confirm := fs.String("confirm", "", "must match MONGODB_DATABASE to proceed")
	_ = fs.Parse(args)
	if *in == "" || *confirm == "" {
		fmt.Fprintln(os.Stderr, "import requires -i and --confirm")
		return 2
	}

	cfg, database, cleanup, err := connect()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	defer cleanup()

	if *confirm != cfg.MongoDatabase {
		fmt.Fprintf(os.Stderr, "confirmation %q does not match database %q\n", *confirm, cfg.MongoDatabase)
		return 2
	}

	f, err := os.Open(*in)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	defer f.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	result, err := backup.ImportDatabase(ctx, database, f, backup.ImportOptions{RequireDatabaseMatch: true})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	for _, name := range backup.CollectionNames {
		fmt.Printf("imported %s: %d documents\n", name, result.Counts[name])
	}
	return 0
}

func connect() (config.Config, *mongo.Database, func(), error) {
	cfg, err := config.Load()
	if err != nil {
		return config.Config{}, nil, nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), cfg.MongoConnectTimeout())
	defer cancel()

	client, err := db.ConnectMongo(ctx, cfg.MongoURI)
	if err != nil {
		return config.Config{}, nil, nil, err
	}
	database := client.Database(cfg.MongoDatabase)
	return cfg, database, func() {
		disconnectCtx, disconnectCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer disconnectCancel()
		_ = client.Disconnect(disconnectCtx)
	}, nil
}
