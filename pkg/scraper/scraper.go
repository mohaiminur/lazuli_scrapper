package scraper

import (
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/gocolly/colly/v2"
)

// Product defines the structure to hold scraped product information.
type Product struct {
	URL            string
	Breadcrumb     string
	ImageURL       string
	Category       string
	ProductName    string
	Price          string
	AvailableSizes string
	SizeDetails    string
	Description    string
	Keywords       string
}

// Start initiates the scraping process.
func Start() {
	// Create a new Colly collector instance.
	// We allow only 'shop.adidas.jp' domain and set a common User-Agent to mimic a browser.
	c := colly.NewCollector(
		colly.AllowedDomains("shop.adidas.jp"),
		colly.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36"),
		// Add parallelism if needed, but start simple for debugging
		//colly.Async(true),
		// colly.MaxDepth(2),
	)

	// Debugging: Log when a request is made by the main collector.
	c.OnRequest(func(r *colly.Request) {
		fmt.Printf("Visiting (main collector): %s\n", r.URL.String())
	})

	// Debugging: Log errors encountered by the main collector.
	c.OnError(func(r *colly.Response, err error) {
		log.Printf("Error (main collector) visiting %s: %v (Status: %d)\n", r.Request.URL.String(), err, r.StatusCode)
	})

	// Debugging: Log responses received by the main collector.
	c.OnResponse(func(r *colly.Response) {
		fmt.Printf("Received response (main collector) for: %s (Status: %d)\n", r.Request.URL.String(), r.StatusCode)
		// Optionally, print the response body for more in-depth debugging:
		fmt.Println("Response Body (main collector):\n", string(r.Body[:500]), "...")
	})

	var products []Product

	// Create a cloned collector for product detail pages.
	productCollector := c.Clone()

	// Debugging: Log when a request is made by the product collector.
	productCollector.OnRequest(func(r *colly.Request) {
		fmt.Printf("Visiting (product collector): %s\n", r.URL.String())
	})

	// Debugging: Log errors encountered by the product collector.
	productCollector.OnError(func(r *colly.Response, err error) {
		log.Printf("Error (product collector) visiting %s: %v (Status: %d)\n", r.Request.URL.String(), err, r.StatusCode)
	})

	// Debugging: Log responses received by the product collector.
	productCollector.OnResponse(func(r *colly.Response) {
		fmt.Printf("Received response (product collector) for: %s (Status: %d)\n", r.Request.URL.String(), r.StatusCode)
		// Optionally, print the response body for more in-depth debugging:
		fmt.Println("Response Body (product collector):\n", string(r.Body[:500]), "...")
	})

	// OnHTML callback for the main collector to find product listings.
	// It looks for elements with the class 'item-box'.
	c.OnHTML(".item-box", func(e *colly.HTMLElement) {
		link := e.ChildAttr("a", "href")
		if strings.Contains(link, "/products/") {
			fullURL := e.Request.AbsoluteURL(link)
			fmt.Printf("    [Main Collector] Found product link: %s\n", fullURL)
			productCollector.Visit(fullURL)
		} else {
			fmt.Printf("    [Main Collector] Skipping non-product link: %s (from %s)\n", link, e.Request.URL.String())
		}
	})

	// OnHTML callback for the product collector to extract details from individual product pages.
	// It operates on the entire HTML of the product page.
	productCollector.OnHTML("html", func(e *colly.HTMLElement) {
		product := Product{}
		product.URL = e.Request.URL.String()
		product.Breadcrumb = strings.Join(e.ChildTexts(".breadcrumb li"), " > ")
		product.ImageURL = e.ChildAttr(".product-image img", "src")
		product.Category = e.ChildText(".category")
		product.ProductName = e.ChildText(".product-title")
		product.Price = e.ChildText(".product-price .value")
		product.AvailableSizes = strings.Join(e.ChildTexts(".size-selector li"), ", ")
		product.SizeDetails = strings.Join(e.ChildTexts(".size-details .size-info"), "; ")
		product.Description = e.ChildText(".product-description")

		// --- Robust Keywords extraction ---
		keywordsMeta := e.DOM.Find("meta[name='keywords']")
		if keywordsMeta.Length() > 0 {
			content, exists := keywordsMeta.Attr("content")
			if exists {
				product.Keywords = content
			} else {
				product.Keywords = "" // 'content' attribute not found on the meta tag
				fmt.Printf("        [Product Collector] Warning: 'content' attribute not found for keywords meta tag on %s\n", e.Request.URL.String())
			}
		} else {
			product.Keywords = "" // Meta tag 'meta[name='keywords']' not found
			fmt.Printf("        [Product Collector] Warning: Meta tag 'meta[name='keywords']' not found on %s\n", e.Request.URL.String())
		}
		// --- End Keywords extraction ---

		products = append(products, product)
		fmt.Printf("        [Product Collector] Scraped product: %s\n", product.ProductName)
	})

	// Start the main collector by visiting the initial category page.
	fmt.Println("Starting scraping from: https://shop.adidas.jp/men/")
	err := c.Visit("https://shop.adidas.jp/men/")
	if err != nil {
		log.Fatalf("Failed to visit initial URL: %v\n", err)
	}

	// Wait until all visits are finished for both collectors.
	// This is important when using Async(true) or multiple collectors.
	c.Wait()
	productCollector.Wait()

	// Ensure the 'csv' directory exists.
	err = os.MkdirAll("csv", os.ModePerm)
	if err != nil {
		log.Fatalf("Could not create directory 'csv': %v", err)
	}

	// Create or open the CSV file for writing.
	file, err := os.Create("csv/products.csv")
	if err != nil {
		log.Fatalf("Could not create CSV file: %v", err)
	}
	defer file.Close() // Ensure the file is closed when the function exits.

	writer := csv.NewWriter(file)
	defer writer.Flush() // Ensure all buffered data is written to the file.

	// Define CSV headers.
	headers := []string{
		"URL", "Breadcrumb", "ImageURL", "Category", "ProductName",
		"Price", "AvailableSizes", "SizeDetails", "Description", "Keywords",
	}
	writer.Write(headers)

	// Write product data to the CSV file.
	for _, p := range products {
		record := []string{
			p.URL, p.Breadcrumb, p.ImageURL, p.Category, p.ProductName,
			p.Price, p.AvailableSizes, p.SizeDetails, p.Description, p.Keywords,
		}
		writer.Write(record)
	}

	fmt.Printf("Scraping completed. %d products written to csv/products.csv\n", len(products))
}
