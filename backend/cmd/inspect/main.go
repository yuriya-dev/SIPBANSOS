package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("missing DATABASE_URL")
	}

	ctx := context.Background()
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		log.Fatalf("failed to parse dsn: %v", err)
	}
	config.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		log.Fatalf("failed to connect db: %v", err)
	}
	defer pool.Close()

	var userCount int
	err = pool.QueryRow(ctx, "SELECT COUNT(*) FROM users").Scan(&userCount)
	if err != nil {
		log.Fatalf("query count failed: %v", err)
	}
	fmt.Printf("Total users: %d\n", userCount)

	if userCount > 0 {
		rows, err := pool.Query(ctx, "SELECT id, username, role, full_name FROM users")
		if err != nil {
			log.Fatalf("query list failed: %v", err)
		}
		defer rows.Close()

		fmt.Println("Users:")
		for rows.Next() {
			var id, username, role, fullName string
			if err := rows.Scan(&id, &username, &role, &fullName); err != nil {
				log.Fatalf("scan failed: %v", err)
			}
			fmt.Printf("- %s | %s | %s | %s\n", id, username, role, fullName)
		}
	}
}
