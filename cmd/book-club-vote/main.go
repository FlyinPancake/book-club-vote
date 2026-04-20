package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"bookclubvote/internal/config"
	"bookclubvote/internal/schema"
	"bookclubvote/internal/server"
)

func main() {
	var (
		configPath   string
		schemaOut    string
		writeSchema  bool
		validateOnly bool
	)

	flag.StringVar(&configPath, "config", "config.yaml", "Path to the YAML config file")
	flag.StringVar(&schemaOut, "schema-out", "schema/config.schema.json", "Path to write the generated JSON Schema")
	flag.BoolVar(&writeSchema, "write-schema", false, "Write the generated JSON Schema and exit")
	flag.BoolVar(&validateOnly, "validate-config", false, "Validate the config file and exit")
	flag.Parse()

	if writeSchema {
		if err := schema.WriteFile(schemaOut); err != nil {
			fatalf("write schema: %v", err)
		}
		fmt.Printf("wrote schema to %s\n", schemaOut)
		if !validateOnly {
			return
		}
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		fatalf("load config: %v", err)
	}

	if validateOnly {
		fmt.Printf("config %s is valid\n", configPath)
		return
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := server.Run(ctx, cfg); err != nil && !errors.Is(err, context.Canceled) {
		fatalf("server failed: %v", err)
	}
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
