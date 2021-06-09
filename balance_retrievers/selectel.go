package balance_retrievers

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"
)

const selectelUrl = "https://my.selectel.ru/api/v3/billing/balance"

type SelectelConfig struct {
	ApiKey string
}

type SelectelBalanceRetriever struct {
	config SelectelConfig
}

type BalanceResponse struct {
	Data struct {
		Primary selectelBalance `json:"primary"`
		Storage selectelBalance `json:"storage"`
		Vmware  selectelBalance `json:"vmware"`
		Vpc     selectelBalance `json:"vpc"`
	} `json:"data"`
}

type selectelBalance struct {
	Balance float64 `json:"main"`
}

func NewSelectelRetriever(config SelectelConfig) BalanceRetriever {
	return SelectelBalanceRetriever{
		config: config,
	}
}

func (bf SelectelBalanceRetriever) GetName() string {
	return "selectel"
}

func (bf SelectelBalanceRetriever) GetBalance() ([]ServiceBalance, error) {
	body, err := bf.loadBody()
	if err != nil {
		return []ServiceBalance{}, errors.New(fmt.Sprintf("Error fetching balance: %s", err.Error()))
	}

	balanceResponse := BalanceResponse{}
	if err := json.Unmarshal(body, &balanceResponse); err != nil {
		return []ServiceBalance{}, errors.New(fmt.Sprintf("Response parse error: %s", err.Error()))
	}

	return []ServiceBalance{
		{Name: "primary", Balance: balanceResponse.Data.Primary.Balance / 100},
		{Name: "storage", Balance: balanceResponse.Data.Storage.Balance / 100},
		{Name: "vmware", Balance: balanceResponse.Data.Vmware.Balance / 100},
		{Name: "vpc", Balance: balanceResponse.Data.Vpc.Balance / 100},
	}, nil
}

func (bf SelectelBalanceRetriever) loadBody() ([]byte, error) {
	client := http.Client{
		Timeout: time.Second * 2,
	}
	req, err := http.NewRequest(http.MethodGet, selectelUrl, nil)
	if err != nil {
		return []byte{}, errors.New(fmt.Sprintf("Cannot create request: %s", err.Error()))
	}

	req.Header.Add("X-token", bf.config.ApiKey)
	res, err := client.Do(req)
	if err != nil {
		return []byte{}, errors.New(fmt.Sprintf("Request error: %s", err.Error()))
	}

	defer func() {
		err := res.Body.Close()
		if err != nil {
			log.Println(fmt.Sprintf("Cannot close response body: %s", err.Error()))
		}
	}()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return []byte{}, errors.New(fmt.Sprintf("Cannot read response body: %s", err.Error()))
	}

	return body, nil
}
