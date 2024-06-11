package runner

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
)

const api = "https://api.binance.com/api/v3/ticker/price?symbol="

type coins struct {
	Symbol string `json:"symbol"`
	Price  string `json:"price"`
}

type Handler struct {
	Coins        []coins
	OutChannel   chan string
	Stop         chan struct{}
	Wg           *sync.WaitGroup
	RequestCount int
}

func (h *Handler) AppendCoinsSymbol(str string) {
	h.Coins = append(h.Coins, coins{str, "0"})
}

func (h *Handler) Run(wg *sync.WaitGroup) {
	defer wg.Done()
	if len(h.Coins) == 0 {
		return
	}
	for i := range h.Coins {
		price, err := getPrice(h.Coins[i].Symbol)
		h.RequestCount++
		if err != nil {
			log.Println(err.Error())
			continue
		}
		h.Coins[i].Price = price
		h.OutChannel <- fmt.Sprintf("%s price:%s", h.Coins[i].Symbol, h.Coins[i].Price)
	}

	for {
		for i := range h.Coins {
			select {
			case _, ok := <-h.Stop:
				if !ok {
					return
				}
			default:
				price, err := getPrice(h.Coins[i].Symbol)
				h.RequestCount++
				if err != nil {
					log.Println(err.Error())
				}
				if price != h.Coins[i].Price {
					h.OutChannel <- fmt.Sprintf("%s price:%s changed", h.Coins[i].Symbol, h.Coins[i].Price)
				} else {
					h.OutChannel <- fmt.Sprintf("%s price:%s", h.Coins[i].Symbol, h.Coins[i].Price)
				}
				h.Coins[i].Price = price
			}
		}
	}
}

func (h *Handler) GetRequestCount() int {
	return h.RequestCount
}

func getPrice(symbol string) (string, error) {
	resp, err := http.Get(api + symbol)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var coin coins
	err = json.Unmarshal(body, &coin)
	if err != nil {
		return "", err
	}
	return coin.Price, nil
}
