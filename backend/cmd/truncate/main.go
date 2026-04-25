package main

import (
	"bondscope/database"
	"fmt"
	"log"
)

func main() {
	db, err := database.InitDB()
	if err != nil {
		log.Fatalf("DB initialization failed: %v", err)
	}

	result := db.Exec("TRUNCATE TABLE yield_rates RESTART IDENTITY")
	if result.Error != nil {
		log.Fatalf("truncate failed: %v", result.Error)
	}

	fmt.Println("yield_rates table cleared.")
}
