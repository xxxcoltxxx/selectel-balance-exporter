package main

import (
    "balance_exporter/balance_retrievers"
    "flag"
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promhttp"
    "github.com/prometheus/common/version"
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
var balanceGauge *prometheus.GaugeVec

var mutex sync.RWMutex

func init() {
    balanceGauge = newGauge()
    prometheus.MustRegister(balanceGauge)
}

func newGauge() *prometheus.GaugeVec {
    return prometheus.NewGaugeVec(
        prometheus.GaugeOpts{
            Subsystem:   "balance",
            Help:        "Balance for service in selectel account",
            Name:        "selectel",
        },
        []string{"service"},
    )
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

    log.Println("Starting Selectel balance exporter", version.Info())
    log.Println("Build context", version.BuildContext())

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
            balanceGauge.With(prometheus.Labels{"service": b.Name}).Set(b.Balance)
        }
    }
}
