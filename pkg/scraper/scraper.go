package scrapper

import (
	"context"
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/chromedp/chromedp"
)

// Product struct designed to hold all the data points required by the PDF.
type Product struct {
	URL                     string
	Name                    string
	ProductID               string
	Price                   string
	ImageURL                string
	Breadcrumb              string
	DescriptionGeneral      string
	DescriptionTitle        string
	DescriptionItemized     string
	AvailableSizes          string
	SenseOfSize             string
	Keywords                string
	SizeChartJSON           string
	ReviewsJSON             string
	OverallRating           string
	NumberOfReviews         string
	RecommendedRate         string
	CoordinatedProductsJSON string
}

const (
	startURL      = "https://shop.adidas.jp/men/"
	numWorkers    = 4
	productTarget = 250
)

// Start is the main entry point for the scraping process.
func Start() {
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", false),
		chromedp.Flag("disable-blink-features", "AutomationControlled"),
		chromedp.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/114.0.0.0 Safari/537.36"),
	)
	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	ctx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	log.Println("STEP 1: Discovering product links...")
	productLinks, err := discoverProductLinks(ctx, startURL)
	if err != nil {
		log.Fatalf("❌ Failed to discover product links: %v", err)
	}
	log.Printf("STEP 1 COMPLETE: ✅ Found %d product links. Will scrape up to %d products.", len(productLinks), productTarget)

	if len(productLinks) == 0 {
		log.Fatalf("Could not find any product links to scrape. The website structure may have changed.")
	}

	log.Printf("STEP 2: Starting %d concurrent workers...", numWorkers)
	jobs := make(chan string, len(productLinks))
	results := make(chan Product, len(productLinks))
	var wg sync.WaitGroup

	for i := 1; i <= numWorkers; i++ {
		wg.Add(1)
		go worker(i, ctx, &wg, jobs, results)
	}

	for i, link := range productLinks {
		if i >= productTarget {
			break
		}
		jobs <- link
	}
	close(jobs)

	wg.Wait()
	close(results)
	log.Println("STEP 2 COMPLETE: All workers have finished.")

	var finalProducts []Product
	for product := range results {
		finalProducts = append(finalProducts, product)
	}

	log.Printf("STEP 3: Writing %d products to CSV...", len(finalProducts))
	writeCSV(finalProducts)
}

func worker(id int, ctx context.Context, wg *sync.WaitGroup, jobs <-chan string, results chan<- Product) {
	defer wg.Done()
	taskCtx, cancel := chromedp.NewContext(ctx)
	defer cancel()

	for url := range jobs {
		log.Printf("[Worker %d] Scraping %s", id, url)
		product, err := scrapeProductPage(taskCtx, url)
		if err != nil {
			log.Printf("[Worker %d] ❌ Error scraping %s: %v", id, url, err)
			continue
		}
		log.Printf("[Worker %d] ✅ Success for %s. Scraped product: '%s'", id, url, product.Name)
		results <- product
	}
}

// discoverProductLinks finds all product detail page URLs from the main category page.
func discoverProductLinks(ctx context.Context, url string) ([]string, error) {
	var links []string
	scrolls := 5

	taskCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	log.Println("-> Navigating to category page and scrolling to load all products...")
	err := chromedp.Run(taskCtx,
		chromedp.Navigate(url),
		chromedp.ActionFunc(func(ctx context.Context) error {
			for i := 0; i < scrolls; i++ {
				log.Printf("--> Scrolling... (%d/%d)", i+1, scrolls)
				err := chromedp.Evaluate(`window.scrollTo(0, document.documentElement.scrollHeight);`, nil).Do(ctx)
				if err != nil {
					return err
				}
				time.Sleep(2 * time.Second)
			}
			return nil
		}),
		chromedp.Evaluate(`Array.from(document.querySelectorAll('a.c-product-card__link')).map(a => a.href)`, &links),
	)

	if err != nil {
		return nil, err
	}

	uniqueLinks := make(map[string]bool)
	var result []string
	for _, link := range links {
		if !uniqueLinks[link] {
			uniqueLinks[link] = true
			result = append(result, link)
		}
	}
	return result, nil
}

// scrapeProductPage extracts all required information from a single product page.
func scrapeProductPage(ctx context.Context, url string) (Product, error) {
	var p Product
	var sizeChartRaw, reviewsRaw, coordinatedRaw string

	// ** FIXED: ProductID is reliably parsed from the URL. **
	p.URL = url
	urlParts := strings.Split(strings.Trim(url, "/"), "/")
	if len(urlParts) > 0 {
		p.ProductID = urlParts[len(urlParts)-1]
	}

	sizeChartScript := `(() => { const table = document.querySelector('.p-size-chart__table table'); if (!table) return null; const headers = Array.from(table.querySelectorAll('thead th')).map(th => th.innerText.trim()); const rows = Array.from(table.querySelectorAll('tbody tr')).map(tr => { const rowData = {}; Array.from(tr.querySelectorAll('td')).forEach((td, i) => { const header = headers[i]; if (header) rowData[header] = td.innerText.trim(); }); return rowData; }); return JSON.stringify(rows); })();`
	reviewsScript := `(() => { const reviews = []; document.querySelectorAll('.BVRRDisplayContentReview').forEach(el => { reviews.push({ rating: el.querySelector('.BVRRRatingNumber')?.innerText.trim(), title: el.querySelector('.BVRRValueTitle')?.innerText.trim(), date: el.querySelector('.BVRRValueDate')?.innerText.trim(), reviewer: el.querySelector('.BVRRValueNickname')?.innerText.trim(), description: el.querySelector('.BVRRReviewTextContainer')?.innerText.trim() }); }); return JSON.stringify(reviews); })();`
	coordinatedProductsScript := `(() => { const items = []; document.querySelectorAll('.p-coordinated-items__list-item a').forEach(el => { items.push({ name: el.querySelector('.p-coordinated-items__name')?.innerText.trim(), price: el.querySelector('.c-price__value')?.innerText.trim(), url: el.href, imageUrl: el.querySelector('img')?.src }); }); return JSON.stringify(items); })();`

	scrapeCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	log.Printf("-> [%s] Running browser actions...", url)
	err := chromedp.Run(scrapeCtx,
		chromedp.Navigate(url),
		chromedp.WaitReady(`h1.p-article__name`),

		// ** FIXED: All selectors are based on the provided HTML files. **
		chromedp.Text(`h1.p-article__name`, &p.Name),
		chromedp.Text(`.p-article__breadcrumb`, &p.Breadcrumb),
		chromedp.Text(`.p-price-group__price-item .c-price__value`, &p.Price),
		chromedp.Text(`.c-description__title`, &p.DescriptionTitle),
		chromedp.Text(`.c-description__text`, &p.DescriptionGeneral),
		chromedp.Text(`[data-test-id="pdp-description-itemized"]`, &p.DescriptionItemized),
		chromedp.Text(`[data-test-id="sense-of-size-info"]`, &p.SenseOfSize),
		chromedp.Text(`[data-test-id="reviews-overall-rating"]`, &p.OverallRating),
		chromedp.Text(`[data-test-id="reviews-number-of-reviews"]`, &p.NumberOfReviews),
		chromedp.Text(`[data-test-id="reviews-recommended-rate"]`, &p.RecommendedRate),
		chromedp.AttributeValue(`.c-article-image__img`, "src", &p.ImageURL, nil), // Get image URL

		// ** FIXED: Extracting other data points directly. **
		chromedp.Evaluate(`Array.from(document.querySelectorAll('.p-size-selector__item-button')).map(btn => btn.innerText.trim()).join(', ')`, &p.AvailableSizes),
		chromedp.Evaluate(`Array.from(document.querySelectorAll('.p-tags__list-item-link')).map(a => a.innerText.trim()).join(', ')`, &p.Keywords),
		chromedp.Evaluate(coordinatedProductsScript, &coordinatedRaw),
		chromedp.Evaluate(reviewsScript, &reviewsRaw),

		// ** FIXED: Clicking size chart link. **
		chromedp.Click(`.p-size-chart__link`, chromedp.NodeVisible),
		chromedp.WaitVisible(`.p-size-chart__table`),
		chromedp.Evaluate(sizeChartScript, &sizeChartRaw),
	)

	if err != nil {
		return Product{}, fmt.Errorf("browser actions failed: %w", err)
	}

	p.SizeChartJSON = sizeChartRaw
	p.ReviewsJSON = reviewsRaw
	p.CoordinatedProductsJSON = coordinatedRaw

	return p, nil
}

func writeCSV(products []Product) {
	if err := os.MkdirAll("data", os.ModePerm); err != nil {
		log.Fatalf("Failed to create data directory: %v", err)
	}
	file, err := os.Create("data/products.csv")
	if err != nil {
		log.Fatalf("Failed to create CSV file: %v", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	headers := []string{
		"URL", "ProductID", "Name", "Price", "ImageURL", "Breadcrumb",
		"DescriptionGeneral", "DescriptionTitle", "DescriptionItemized",
		"AvailableSizes", "SenseOfSize", "Keywords",
		"SizeChartJSON", "ReviewsJSON", "OverallRating", "NumberOfReviews", "RecommendedRate",
		"CoordinatedProductsJSON",
	}
	if err := writer.Write(headers); err != nil {
		log.Fatalf("Failed to write CSV headers: %v", err)
	}

	for _, p := range products {
		row := []string{
			p.URL, p.ProductID, p.Name, p.Price, p.ImageURL, p.Breadcrumb,
			p.DescriptionGeneral, p.DescriptionTitle, p.DescriptionItemized,
			p.AvailableSizes, p.SenseOfSize, p.Keywords,
			p.SizeChartJSON, p.ReviewsJSON, p.OverallRating, p.NumberOfReviews, p.RecommendedRate,
			p.CoordinatedProductsJSON,
		}
		if err := writer.Write(row); err != nil {
			log.Printf("Warning: failed to write row for %s: %v\n", p.Name, err)
		}
	}
	log.Printf("✅ Success! Data for %d products written to data/products.csv", len(products))
}
