package scraper

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
)

const (
	ZenRowsAPIKey  = "69067cd911bb38e011af19c3bad233a5f6159f96"
	AdidasMenURL   = "https://shop.adidas.jp/men/"
	SampleJSONFile = "sample.json"
)

type ZenRowsOverallResponse struct {
	HTML string        `json:"html"`
	XHR  []XHRResponse `json:"xhr"`
}

type XHRResponse struct {
	URL  string `json:"url"`
	Body string `json:"body"`
}

type AdidasRecommendationsResponse struct {
	Recommendations []ProductDetails `json:"recommendations"`
	Breadcrumbs     []Breadcrumb     `json:"breadcrumbs"`
}

type Breadcrumb struct {
	Text string `json:"text"`
	Link string `json:"link"`
	Type string `json:"type"`
}

type ProductDetails struct {
	ArticleNumber string         `json:"articleNumber"`
	Name          string         `json:"name"`
	Category      string         `json:"category"`
	Link          string         `json:"link"`
	ImageLink     string         `json:"imageLink"`
	SubTitle      string         `json:"subTitle"`
	Sizes         []string       `json:"sizes"`
	Sport         string         `json:"sport"`
	Surface       []string       `json:"surface"`
	Brand         string         `json:"brand"`
	Pricing       ProductPricing `json:"pricing"`
}

type ProductPricing struct {
	CurrentPrice float64 `json:"currentPrice"`
}

// ScrapeAndSaveToJSON makes an API call to ZenRows and saves the response to a JSON file.
func ScrapeAndSaveToJSON(outputFileName string) error {
	log.Printf("🔎 Scraping Adidas data via ZenRows API and saving to %s...", outputFileName)

	client := &http.Client{}
	zenRowsURL := fmt.Sprintf("https://api.zenrows.com/v1/?apikey=%s&url=%s&js_render=true&json_response=true&js_instructions=%%5B%%7B%%22click%%22%%3A%%22.selector%%22%%7D%%2C%%7B%%22wait%%22%%3A500%%7D%%2C%%7B%%22fill%%22%%3A%%5B%%22.input%%22%%2C%%22value%%22%%5D%%7D%%2C%%7B%%22wait_for%%22%%3A%%22.slow_selector%%22%%7D%%5D&premium_proxy=true&proxy_country=us", ZenRowsAPIKey, AdidasMenURL)

	req, err := http.NewRequest("GET", zenRowsURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make HTTP request to ZenRows: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("ZenRows API returned non-OK status: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read ZenRows API response body: %w", err)
	}

	file, err := os.Create(outputFileName)
	if err != nil {
		return fmt.Errorf("failed to create output file %s: %w", outputFileName, err)
	}
	defer file.Close()

	_, err = file.Write(body)
	if err != nil {
		return fmt.Errorf("failed to write response to file %s: %w", outputFileName, err)
	}

	log.Printf("✅ Successfully saved ZenRows API response to %s\n", outputFileName)
	return nil
}

// ProcessJSONAndWriteToCSV reads product data from a JSON file and writes it to a CSV file.
func ProcessJSONAndWriteToCSV(jsonFileName string) error {
	log.Printf("🔎 Processing data from %s and writing to CSV...", jsonFileName)

	products, breadcrumbs, err := getProductDataFromFile(jsonFileName)
	if err != nil {
		return fmt.Errorf("failed to get product data from file: %w", err)
	}

	if len(products) == 0 {
		log.Println("⚠️ No product data found in the JSON file. Skipping CSV creation.")
		return nil
	}

	log.Printf("✅ Found %d products. Writing to CSV...\n", len(products))

	// Ensure the 'csv' directory exists
	if err := os.MkdirAll("csv", os.ModePerm); err != nil {
		return fmt.Errorf("failed to create 'csv' directory: %w", err)
	}

	// Create and write to CSV file
	csvFileName := "csv/products.csv"
	file, err := os.Create(csvFileName)
	if err != nil {
		return fmt.Errorf("failed to create CSV file %s: %w", csvFileName, err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write CSV header
	header := []string{"ID", "URL", "ProductName", "Category", "Price", "ImageURL", "AvailableSizes", "SizeDetails", "Description", "Keywords", "Breadcrumbs"}
	if err := writer.Write(header); err != nil {
		return fmt.Errorf("failed to write CSV header: %w", err)
	}

	// Prepare breadcrumbs string (assuming one set of breadcrumbs for the page/file)
	breadcrumbsString := ""
	if len(breadcrumbs) > 0 {
		var bcTexts []string
		for _, bc := range breadcrumbs {
			bcTexts = append(bcTexts, bc.Text)
		}
		breadcrumbsString = strings.Join(bcTexts, " > ")
	}

	// Write data for ALL products found to CSV
	for _, product := range products {
		id := product.ArticleNumber
		if id == "" {
			id = "N/A"
		}

		fullURL := product.Link
		if !strings.HasPrefix(fullURL, "http") {
			fullURL = "https://www.adidas.jp" + fullURL
		}

		productName := product.Name
		if productName == "" {
			productName = "N/A"
		}

		category := product.Category
		if category == "" {
			category = "N/A"
		}

		price := fmt.Sprintf("%.2f", product.Pricing.CurrentPrice)

		imageURL := product.ImageLink
		if imageURL == "" {
			imageURL = "N/A"
		}

		availableSizes := strings.Join(product.Sizes, ", ")
		if availableSizes == "" {
			availableSizes = "N/A"
		}

		description := product.SubTitle
		if description == "" {
			description = "N/A"
		}

		keywords := []string{}
		if product.Sport != "" {
			keywords = append(keywords, product.Sport)
		}
		if len(product.Surface) > 0 {
			keywords = append(keywords, product.Surface...)
		}
		if product.Brand != "" {
			keywords = append(keywords, product.Brand)
		}
		if product.Category != "" {
			keywords = append(keywords, product.Category)
		}
		keywordsString := strings.Join(keywords, ", ")
		if keywordsString == "" {
			keywordsString = "N/A"
		}

		row := []string{
			id,
			fullURL,
			productName,
			category,
			price,
			imageURL,
			availableSizes,
			availableSizes,
			description,
			keywordsString,
			breadcrumbsString,
		}
		if err := writer.Write(row); err != nil {
			log.Printf("⚠️ Failed to write row for product %s to CSV: %v", id, err)
		}
	}

	log.Printf("✅ Successfully created CSV file: %s\n", csvFileName)
	return nil
}

// getProductDataFromFile reads the content from the specified file and extracts product data.
// This is an internal helper, so it starts with a lowercase letter.
func getProductDataFromFile(filename string) ([]ProductDetails, []Breadcrumb, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open file %s: %w", filename, err)
	}
	defer file.Close()

	rawBody, err := io.ReadAll(file)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read file content: %w", err)
	}

	var zenRowsResp ZenRowsOverallResponse
	if err := json.Unmarshal(rawBody, &zenRowsResp); err != nil {
		return nil, nil, fmt.Errorf("failed to unmarshal JSON from file: %w. Raw Content Start: %s...", err, string(rawBody[:50]))
	}

	var allProducts []ProductDetails
	var breadcrumbs []Breadcrumb

	// Iterate through XHR responses to find the one with product data (known to be "recs/api/products")
	for _, xhrItem := range zenRowsResp.XHR {
		if strings.Contains(xhrItem.URL, "recs/api/products") {
			var recommendationsAndBreadcrumbs struct {
				Recommendations []ProductDetails `json:"recommendations"`
				Breadcrumbs     []Breadcrumb     `json:"json_breadcrumbs"` // Corrected breadcrumb key as observed in samples
			}
			// Unmarshal the body string of the specific XHR item
			if err := json.Unmarshal([]byte(xhrItem.Body), &recommendationsAndBreadcrumbs); err == nil {
				allProducts = recommendationsAndBreadcrumbs.Recommendations
				breadcrumbs = recommendationsAndBreadcrumbs.Breadcrumbs

				// Since this XHR is confirmed to contain the primary product list, return it.
				return allProducts, breadcrumbs, nil
			}
		}
	}

	return nil, nil, fmt.Errorf("no product data found in the XHR bodies with 'recs/api/products' URL in the file")
}
