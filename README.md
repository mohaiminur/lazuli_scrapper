
#  Product Scraper (Go)

This project contains a Go application designed to scrape product details from the men's section of the Adidas Japan (adidas.jp) e-commerce website. It utilizes the `gocolly/colly` library for web crawling and saves the extracted data into a CSV file.

## Project Overview

The primary goal of this scraper is to acquire specific information from Adidas product detail pages, including:

* [cite_start]**URL** [cite: 15]
* [cite_start]**Breadcrumb (Category)** [cite: 25]
* [cite_start]**Image URL** [cite: 31]
* [cite_start]**Category** [cite: 29]
* [cite_start]**Product Name** [cite: 45]
* [cite_start]**Price (Pricing)** [cite: 42]
* [cite_start]**Available Sizes** [cite: 49]
* [cite_start]**Size Details (Sense of the size / Size information)** [cite: 52, 98]
* [cite_start]**Description (General Description of the product)** [cite: 75]
* [cite_start]**Keywords** [cite: 247]

The scraped data is then organized and saved into a CSV file named `products.csv`. [cite_start]The target quantity of products to crawl is between 200 and 300.

## Requirements

* Go (version 1.16 or higher recommended)

## Setup and Installation

1.  **Clone the repository:**
    ```bash
    git clone [https://github.com/your-username/your-repo-name.git](https://github.com/your-username/your-repo-name.git)
    cd your-repo-name
    ```

2.  **Install dependencies:**
    The project uses `gocolly/colly/v2`. You can install it using Go Modules:
    ```bash
    go mod tidy
    ```

## Usage

To run the scraper, execute the `main.go` file (assuming your `main.go` calls `scraper.Start()`).

```bash
go run main.go