package main

import (
	"log"

	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type ExchangeRateAdapter interface {
	GetExchangeRate(from, to string) (float64, error)
	ConvertCurrency(amount float64, from, to string) (float64, error)
}

type OpenExchangeRatesAdapter struct {
	apiKey string
}

func (o *OpenExchangeRatesAdapter) GetExchangeRate(from, to string) (float64, error) {
	// Build the API URL
	url := fmt.Sprintf("https://openexchangerates.org/api/latest.json?app_id=%s&symbols=%s,%s", o.apiKey, from, to)

	// Send a GET request to the API
	resp, err := http.Get(url)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}

	// Unmarshal the JSON response
	var data struct {
		Rates map[string]float64 `json:"rates"`
	}
	if err := json.Unmarshal(body, &data); err != nil {
		return 0, err
	}

	// Return the exchange rate
	return data.Rates[to] / data.Rates[from], nil
}

func (o *OpenExchangeRatesAdapter) ConvertCurrency(amount float64, from, to string) (float64, error) {
	rate, err := o.GetExchangeRate(from, to)
	if err != nil {
		return 0, err
	}
	return amount * rate, nil
}

type FixerIoAdapter struct {
	apiKey string
}

func (f *FixerIoAdapter) GetExchangeRate(from, to string) (float64, error) {
	// Build the API URL
	url := fmt.Sprintf("http://data.fixer.io/api/latest?access_key=%s&symbols=%s,%s", f.apiKey, from, to)

	// Send a GET request to the API
	resp, err := http.Get(url)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}

	// Unmarshal the JSON response
	var data struct {
		Rates map[string]float64 `json:"rates"`
		Base  string             `json:"base"`
	}
	if err := json.Unmarshal(body, &data); err != nil {
		return 0, err
	}

	// Return the exchange rate
	return data.Rates[to] / data.Rates[from], nil
}

func (f *FixerIoAdapter) ConvertCurrency(amount float64, from, to string) (float64, error) {
	rate, err := f.GetExchangeRate(from, to)
	if err != nil {
		return 0, err
	}
	return amount * rate, nil
}

func MoneyExchangeHandler(c *gin.Context) {
	var adapter ExchangeRateAdapter
	source := c.Query("source")
	apiKey := c.Query("key")

	if apiKey == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "API key is required!", "name": "key"})
		return
	}

	switch source {
	case "openexchangerates":
		adapter = &OpenExchangeRatesAdapter{
			apiKey: apiKey,
		}
	case "fixerio":
		adapter = &FixerIoAdapter{
			apiKey: apiKey,
		}
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid exchange rate source", "name": "source"})
		return
	}

	// Perform currency conversion using the selected adapter
	amountStr := c.Query("amount")
	from := c.Query("from")
	to := c.Query("to")

	amount, err := strconv.ParseFloat(amountStr, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid amount", "name": "amount"})
		return
	}

	convertedAmount, err := adapter.ConvertCurrency(amount, from, to)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"source":    source,
		"from":      from,
		"to":        to,
		"amount":    amount,
		"converted": convertedAmount,
	})
}

func main() {
	r := gin.Default()

	r.GET("/exchange", MoneyExchangeHandler)

	log.Println("Exchanger server is started!")
	err := r.Run()
	if err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
