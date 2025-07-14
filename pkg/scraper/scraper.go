package scraper

import (
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"strings"
	"time" // Import the time package

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
	c := colly.NewCollector(
		colly.AllowedDomains("shop.adidas.jp", "www.adidas.jp"),
		colly.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36"),
	)

	// Add a rate limit with a delay between requests
	c.Limit(&colly.LimitRule{
		DomainGlob:  "*",             // Apply this rule to all domains allowed by the collector
		Delay:       2 * time.Second, // Wait 2 seconds between requests
		RandomDelay: 1 * time.Second, // Add a random delay of up to 1 second for more natural behavior
	})

	// Define a map of default headers to apply to all requests
	defaultHeaders := map[string]string{
		"Accept":                    "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7",
		"Accept-Language":           "en-US,en;q=0.9,ja;q=0.8",
		"Connection":                "keep-alive",
		"Cache-Control":             "max-age=0",
		"Upgrade-Insecure-Requests": "1",
		"Referer":                   "https://www.google.com/", // Simulate a Google search referrer
	}

	c.OnRequest(func(r *colly.Request) {
		// Apply default headers to every request made by the main collector
		for key, value := range defaultHeaders {
			r.Headers.Set(key, value)
		}
		fmt.Printf("Visiting (main collector): %s\n", r.URL.String())
	})

	c.OnError(func(r *colly.Response, err error) {
		log.Printf("Error (main collector) visiting %s: %v (Status: %d)\n", r.Request.URL.String(), err, r.StatusCode)
	})

	c.OnResponse(func(r *colly.Response) {
		fmt.Printf("Received response (main collector) for: %s (Status: %d)\n", r.Request.URL.String(), r.StatusCode)
	})

	var products []Product

	productCollector := c.Clone()

	// It's crucial to also apply the delay to the cloned collector,
	// as it will be making its own requests to product pages.
	productCollector.Limit(&colly.LimitRule{
		DomainGlob:  "*",             // Apply this rule to all domains allowed by the collector
		Delay:       2 * time.Second, // Wait 2 seconds between requests
		RandomDelay: 1 * time.Second, // Add a random delay of up to 1 second
	})

	productCollector.OnRequest(func(r *colly.Request) {
		// Apply default headers to every request made by the product collector
		for key, value := range defaultHeaders {
			r.Headers.Set(key, value)
		}
		fmt.Printf("Visiting (product collector): %s\n", r.URL.String())
	})

	productCollector.OnError(func(r *colly.Response, err error) {
		log.Printf("Error (product collector) visiting %s: %v (Status: %d)\n", r.Request.URL.String(), err, r.StatusCode)
	})

	productCollector.OnResponse(func(r *colly.Response) {
		fmt.Printf("Received response (product collector) for: %s (Status: %d)\n", r.Request.URL.String(), r.StatusCode)
	})

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

		keywordsMeta := e.DOM.Find("meta[name='keywords']")
		if keywordsMeta.Length() > 0 {
			content, exists := keywordsMeta.Attr("content")
			if exists {
				product.Keywords = content
			} else {
				product.Keywords = ""
				fmt.Printf("        [Product Collector] Warning: 'content' attribute not found for keywords meta tag on %s\n", e.Request.URL.String())
			}
		} else {
			product.Keywords = ""
			fmt.Printf("        [Product Collector] Warning: Meta tag 'meta[name='keywords']' not found on %s\n", e.Request.URL.String())
		}

		products = append(products, product)
		fmt.Printf("        [Product Collector] Scraped product: %s\n", product.ProductName)
	})

	fmt.Println("Starting scraping from: https://shop.adidas.jp/men/")
	err := c.Visit("https://shop.adidas.jp/men/")
	if err != nil {
		log.Fatalf("Failed to visit initial URL: %v\n", err)
	}

	c.Wait()
	productCollector.Wait()

	err = os.MkdirAll("csv", os.ModePerm)
	if err != nil {
		log.Fatalf("Could not create directory 'csv': %v", err)
	}

	file, err := os.Create("csv/products.csv")
	if err != nil {
		log.Fatalf("Could not create CSV file: %v", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	headers := []string{
		"URL", "Breadcrumb", "ImageURL", "Category", "ProductName",
		"Price", "AvailableSizes", "SizeDetails", "Description", "Keywords",
	}
	writer.Write(headers)

	for _, p := range products {
		record := []string{
			p.URL, p.Breadcrumb, p.ImageURL, p.Category, p.ProductName,
			p.Price, p.AvailableSizes, p.SizeDetails, p.Description, p.Keywords,
		}
		writer.Write(record)
	}

	fmt.Printf("Scraping completed. %d products written to csv/products.csv\n", len(products))
}
