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
	case "down-to":
		// migrate down-to <versi> — mengembalikan skema ke versi tertentu.
		// Dipakai saat rollback deploy supaya image lama di-deploy ulang dengan skema yang cocok.
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "usage: migrate down-to <versi>")
			os.Exit(1)
		}
		var target int64
		if _, serr := fmt.Sscanf(os.Args[2], "%d", &target); serr != nil {
			fmt.Fprintln(os.Stderr, "versi harus angka")
			os.Exit(1)
		}
		err = db.MigrateDownTo(ctx, cfg, target)
	case "status":
		err = db.MigrateStatus(ctx, cfg)
	case "version":
		err = db.MigrateVersion(ctx, cfg)
	default:
		fmt.Fprintln(os.Stderr, "usage: migrate [up|down|down-to <versi>|status|version]")
		os.Exit(1)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println("migrate OK:", dir)
}
