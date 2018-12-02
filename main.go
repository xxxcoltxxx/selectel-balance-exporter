package main

import (
    "balance_exporter/balance_retrievers"
    "flag"
    "fmt"
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promhttp"
    "github.com/spf13/viper"
    "log"
    "net/http"
    "sync"
    "time"
)

var addr = flag.String("listen-address", ":9600", "The address to listen on for HTTP requests.")
var config = flag.String("config", "", "Config file")

var balanceFetchers []balance_retrievers.BalanceRetriever
var collectors = make(map[string]prometheus.Gauge)

var mutex sync.RWMutex

func init() {
    for _, service := range []string{"selectel", "storage", "vmware", "vpc"} {
        gauge := newGauge(service)
        collectors[service] = gauge
        prometheus.MustRegister(gauge)
    }
}

func newGauge(name string) prometheus.Gauge {
    return prometheus.NewGauge(struct {
        Namespace   string
        Subsystem   string
        Name        string
        Help        string
        ConstLabels prometheus.Labels
    }{
        Namespace:   "balance",
        Subsystem:   "selectel",
        Help:        fmt.Sprintf("Balance for service %s", name),
        Name:        name,
        ConstLabels: map[string]string{"service": name},
    })
}

func main() {
    v := readConfig()

    if apiKey := v.GetString("apiKey"); apiKey == "" {
        if *config != "" {
            log.Fatalf("Key \"apiKey\" in file %s is not set\n", *config)
        } else {
            log.Fatalf("Environment value \"SELECTEL_API_KEY\" is not set\n")
        }
        return
    } else {
        registerFetcher(balance_retrievers.NewSelectelBalanceFetcher(balance_retrievers.SelectelConfig{ApiKey: apiKey}))
    }

    log.Printf("Starting Selectel balance exporter at address %s\n", *addr)

    loadBalance()

    interval := v.GetInt("interval")
    go startBalanceUpdater(interval)

    http.Handle("/metrics", promhttp.Handler())
    http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        http.ServeFile(w, r, "static/index.html")
    })

    log.Fatal(http.ListenAndServe(*addr, nil))
}

func readConfig() *viper.Viper {
    flag.Parse()
    v := viper.New()
    v.SetDefault("apiKey", "")
    v.SetDefault("interval", 3600)

    if *config != "" {
        v.SetConfigFile(*config)
        v.AddConfigPath(".")
        err := v.ReadInConfig()
        if err != nil {
            log.Println("error", err.Error())
        }
    } else {
        err := v.BindEnv("apiKey", "SELECTEL_API_KEY")
        if err != nil {
            log.Println("error", err.Error())
        }
        v.AutomaticEnv()
    }

    return v
}

func registerFetcher(fetcher balance_retrievers.BalanceRetriever) {
    mutex.Lock()
    balanceFetchers = append(balanceFetchers, fetcher)
    mutex.Unlock()
}

func startBalanceUpdater(interval int) {
    for {
        time.Sleep(time.Second * time.Duration(interval))
        loadBalance()
    }
}

func loadBalance() {
    for _, f := range balanceFetchers {
        results, err := f.GetBalance()
        if err != nil {
            log.Print(err.Error())
            return
        }

        for _, b := range results {
            if gauge := collectors[b.Name]; gauge != nil {
                gauge.Set(b.Balance)
            }
        }
    }
}
