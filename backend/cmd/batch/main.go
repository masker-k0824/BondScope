package main

import (
	"bondscope/updater"
	"fmt"
	"log"
)

func main() {
	n, err := updater.UpdateJGBData(updater.MOFSURL)
	if err != nil {
		log.Fatalf("Error updating JGB data: %v", err)
	}
	fmt.Printf("Successfully updated %d records from MOF!\n", n)
}
