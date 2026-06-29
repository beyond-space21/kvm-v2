package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"

	"kvm_v2/services/config"
	"kvm_v2/services/db"
)

var tables = []string{
	"fee_payments",
	"fee_invoices",
	"attendance_audit_log",
	"attendance_records",
	"sessions",
	"session_templates",
	"enrollments",
	"batches",
	"subject_offering_fee_history",
	"subject_offerings",
	"academic_years",
	"subjects",
	"academic_classes",
}

func main() {
	yes := flag.Bool("yes", false, "confirm deletion of all data except super_admin users")
	flag.Parse()

	if !*yes {
		fmt.Println("This will delete all LMS data except users with role 'super_admin'.")
		fmt.Println("Run again with --yes to continue.")
		os.Exit(1)
	}

	cfg := config.Load()
	if cfg.DatabaseURL == "" {
		log.Fatal("DATABASE_URL is not set. Add it to .env or export it before running.")
	}

	database, err := db.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer database.Close()

	if err := clearData(database); err != nil {
		log.Fatalf("clear data: %v", err)
	}

	fmt.Println("Deleted all data except super_admin users.")
}

func clearData(database *sql.DB) error {
	tx, err := database.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, table := range tables {
		if _, err := tx.Exec("DELETE FROM " + table); err != nil {
			return fmt.Errorf("delete from %s: %w", table, err)
		}
	}

	res, err := tx.Exec("DELETE FROM users WHERE role <> 'super_admin'")
	if err != nil {
		return fmt.Errorf("delete non-admin users: %w", err)
	}

	removed, _ := res.RowsAffected()
	if err := tx.Commit(); err != nil {
		return err
	}

	fmt.Printf("Removed %d non-admin user(s).\n", removed)
	return nil
}
