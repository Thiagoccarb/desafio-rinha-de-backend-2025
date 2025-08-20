package migrations

import (
	"database/sql"
	"log"
	"payment-processor/config"

	_ "github.com/lib/pq"
)

func CreateRinhaTable() {
	config := config.LoadConfig()
	connStr := config.Database.ConnectionString()

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	_, err = db.Exec(`
        CREATE TABLE IF NOT EXISTS rinha (
            id SERIAL PRIMARY KEY NOT NULL,
            uuid UUID UNIQUE NOT NULL,
            amount DECIMAL(10,5) NOT NULL,
			type SMALLINT NOT NULL DEFAULT 1,
			created_at TIMESTAMPTZ NOT NULL     
		);
    `)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Migration ran successfully")
}
