package main

import (
	"log"
	"os"

	"lazuli/pkg/scraper"
)

func main() {
	log.Println("🚀 Starting Adidas product data scraping and CSV generation...")

	// Step 1: Scrape data from Adidas using ZenRows and save to sample.json
	err := scraper.ScrapeAndSaveToJSON(scraper.SampleJSONFile)
	if err != nil {
		log.Fatalf("❌ Failed to scrape data and save to JSON: %v", err)
	}

	// Step 2: Read product data from sample.json and write to CSV
	err = scraper.ProcessJSONAndWriteToCSV(scraper.SampleJSONFile)
	if err != nil {
		log.Fatalf("❌ Failed to process JSON and write to CSV: %v", err)
	}

	// Step 3: Delete the sample.json file
	log.Printf("🗑️ Deleting temporary file: %s...", scraper.SampleJSONFile)
	if err := os.Remove(scraper.SampleJSONFile); err != nil {
		log.Printf("⚠️ Failed to delete %s: %v", scraper.SampleJSONFile, err)
	} else {
		log.Printf("✅ Successfully deleted %s\n", scraper.SampleJSONFile)
	}

	log.Println("✅ All processes completed successfully!")
}
