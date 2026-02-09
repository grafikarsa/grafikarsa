package main

import (
	"context"
	"flag"
	"log"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"

	"grafikarsa/internal/config"
)

func main() {
	// Parse flags with defaults
	username := flag.String("username", "admin", "Admin username")
	email := flag.String("email", "admin@grafikarsa.com", "Admin email")
	password := flag.String("password", "admin123", "Admin password")
	name := flag.String("name", "Administrator", "Admin display name")
	flag.Parse()

	log.Println("Starting admin seeder...")

	cfg := config.Load()

	// Connect to database
	// We might need to wait for DB readiness if running via docker compose immediately after up
	ctx := context.Background()
	db, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v", err)
	}
	defer db.Close()

	if err := db.Ping(ctx); err != nil {
		log.Fatalf("Unable to ping database: %v", err)
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(*password), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("Failed to hash password: %v", err)
	}

	// Prepare user data
	adminID := uuid.New() // Generate a new UUID if creating new

	// Check if admin already exists by username or email
	var existingID uuid.UUID
	err = db.QueryRow(ctx, "SELECT id FROM users WHERE username = $1 OR email = $2", *username, *email).Scan(&existingID)

	if err == nil {
		// Admin exists, update password and name
		log.Printf("User with username/email already exists (ID: %s). Updating...", existingID)
		_, err = db.Exec(ctx, "UPDATE users SET password_hash = $1, name = $2, role = 'admin', updated_at = NOW() WHERE id = $3",
			string(hashedPassword), *name, existingID)
		if err != nil {
			log.Fatalf("Failed to update admin: %v", err)
		}
		log.Println("Admin updated successfully.")
	} else {
		// Create new admin
		log.Println("Creating new admin user...")
		_, err = db.Exec(ctx,
			`INSERT INTO users (id, username, email, name, password_hash, role, created_at, updated_at) 
			 VALUES ($1, $2, $3, $4, $5, 'admin', NOW(), NOW())`,
			adminID, *username, *email, *name, string(hashedPassword))

		if err != nil {
			log.Fatalf("Failed to insert admin: %v", err)
		}

		// Create empty profile/social links if needed (depending on schema triggers/constraints)
		// Assuming minimal insert is fine based on schema docs.

		log.Printf("Admin user created successfully. ID: %s", adminID)
	}

	log.Printf("Credentials -> Username: %s | Email: %s | Name: %s | Password: [HIDDEN]", *username, *email, *name)
}
