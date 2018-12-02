package balance_retrievers

import (
    "encoding/json"
    "io/ioutil"
    "log"
    "net/http"
    "time"
)

const selectelUrl = "https://my.selectel.ru/api/v2/billing/balance"

type SelectelConfig struct {
    ApiKey string
}

type SelectelBalanceRetriever struct {
    config SelectelConfig
}

type selectelResponse struct {
    Data struct {
        Selectel selectelBalance `json:"selectel"`
        Storage  selectelBalance `json:"storage"`
        Vmware   selectelBalance `json:"vmware"`
        Vpc      selectelBalance `json:"vpc"`
    } `json:"data"`
}

type selectelBalance struct {
    Balance float64 `json:"selectelBalance"`
}

func NewSelectelBalanceFetcher(config SelectelConfig) BalanceRetriever {
    return SelectelBalanceRetriever{
        config: config,
    }
}

func (bf SelectelBalanceRetriever) GetName() string {
    return "selectel"
}

func (bf SelectelBalanceRetriever) GetBalance() (balances []ServiceBalance, err error) {
    body, err := bf.loadBody()
    if err != nil {
        log.Printf("Error fetching selectelBalance: %s", err.Error())
    }

    jsonResponse := selectelResponse{}
    err = json.Unmarshal(body, &jsonResponse)
    if err != nil {
        log.Printf("Error fetching selectelBalance: %s", err.Error())
    }

    return []ServiceBalance{
        {Name: "selectel", Balance: jsonResponse.Data.Selectel.Balance / 100},
        {Name: "storage", Balance: jsonResponse.Data.Storage.Balance / 100},
        {Name: "vmware", Balance: jsonResponse.Data.Vmware.Balance / 100},
        {Name: "vpc", Balance: jsonResponse.Data.Vpc.Balance / 100},
    }, nil
}

func (bf SelectelBalanceRetriever) loadBody() ([]byte, error) {
    client := http.Client{
        Timeout: time.Second * 2,
    }
    req, err := http.NewRequest(http.MethodGet, selectelUrl, nil)

    if err != nil {
        log.Printf("Error fetching selectelBalance: %s", err.Error())
        return []byte{}, err
    }

    req.Header.Add("X-token", bf.config.ApiKey)
    res, err := client.Do(req)
    defer func() {
        err := res.Body.Close()
        if err != nil {
            log.Printf("Error fetching selectelBalance: %s", err.Error())
        }
    }()

    body, err := ioutil.ReadAll(res.Body)
    if err != nil {
        log.Printf("Error fetching selectelBalance: %s", err.Error())
    }

    return body, nil
}
