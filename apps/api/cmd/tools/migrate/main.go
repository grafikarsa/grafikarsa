package main

import (
	"context"
	"fmt"
	"os"

	"grafikarsa/api/internal/config"
	"grafikarsa/api/internal/db"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	ctx := context.Background()
	dir := "up"
	if len(os.Args) > 1 {
		dir = os.Args[1]
	}
	switch dir {
	case "up":
		err = db.MigrateUp(ctx, cfg)
	case "down":
		err = db.MigrateDown(ctx, cfg)
	default:
		fmt.Fprintln(os.Stderr, "usage: migrate [up|down]")
		os.Exit(1)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println("migrate OK:", dir)
}
