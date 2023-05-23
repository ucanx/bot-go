package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	binance "github.com/adshao/go-binance/v2"
)

func getHistoricalData(client *binance.Client, symbol, interval string, limit int) ([]*binance.Kline, error) {
	klines, err := client.NewKlinesService().Symbol(symbol).Interval(interval).Limit(limit).Do(context.Background())
	if err != nil {
		return nil, err
	}
	return klines, nil
}

func calculateMovingAverage(data []float64, window int) float64 {
	sum := 0.0
	for _, value := range data {
		sum += value
	}
	return sum / float64(window)
}

func getBalance(client *binance.Client, asset string) (float64, error) {
	account, err := client.NewGetAccountService().Do(context.Background())
	if err != nil {
		return 0, err
	}

	for _, balance := range account.Balances {
		if balance.Asset == asset {
			freeBalance, err := strconv.ParseFloat(balance.Free, 64)
			if err != nil {
				return 0, err
			}
			return freeBalance, nil
		}
	}

	return 0, fmt.Errorf("Asset %s not found in the account", asset)
}

// check Side(side) >>> Side(binance.SideType(side))
func createMarketOrder(client *binance.Client, symbol, side string, quantity float64) (*binance.CreateOrderResponse, error) {
	order, err := client.NewCreateOrderService().Symbol(symbol).Side(binance.SideType(side)).Type(binance.OrderTypeMarket).Quantity(fmt.Sprintf("%.8f", quantity)).Do(context.Background())
	if err != nil {
		return nil, err
	}
	return order, nil
}

func main() {
	apiKey := os.Getenv("BINANCE_API_KEY")
	secretKey := os.Getenv("BINANCE_API_SECRET_KEY")

	client := binance.NewClient(apiKey, secretKey)

	symbol := "BTCUSDT"
	interval := "1h"
	shortWindow := 20
	longWindow := 50
	riskPercentage := 0.01

	balance, err := getBalance(client, "USDT")
	if err != nil {
		log.Fatalf("Error fetching balance: %v", err)
	}
	positionSize := balance * riskPercentage

	for {
		historicalData, err := getHistoricalData(client, symbol, interval, longWindow)
		if err != nil {
			log.Printf("Error fetching historical data: %v", err)
			time.Sleep(60 * time.Second)
			continue
		}

		var closePrices []float64
		for _, kline := range historicalData {
			closePrice, _ := strconv.ParseFloat(kline.Close, 64)
			closePrices = append(closePrices, closePrice)
		}

		shortMovingAverage := calculateMovingAverage(closePrices[len(closePrices)-shortWindow:], shortWindow)
		longMovingAverage := calculateMovingAverage(closePrices, longWindow)

		if shortMovingAverage > longMovingAverage {
			log.Println("Buy signal")
			order, err := createMarketOrder(client, symbol, string(binance.SideTypeBuy), positionSize)
			if err != nil {
				log.Printf("Error creating market order: %v", err)
			} else {
				log.Printf("Market order created: %v", order)
			}
		} else if shortMovingAverage < longMovingAverage {
			log.Println("Sell signal")
			order, err := createMarketOrder(client, symbol, string(binance.SideTypeSell), positionSize)
			if err != nil {
				log.Printf("Error creating market order: %v", err)
			} else {
				log.Printf("Market order created: %v", order)
			}
		}

		time.Sleep(60 * time.Second)
	}
}
