package main

import (
    "context"
    "flag"
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promhttp"
    "github.com/prometheus/common/version"
    "github.com/spf13/viper"
    "log"
    "net/http"
    "os"
    "os/signal"
    "selectel_balance_exporter/balance_retrievers"
    "sync"
    "syscall"
    "time"
)

var addr = flag.String("listen-address", ":9600", "The address to listen on for HTTP requests.")
var config = flag.String("config", "", "Config file")

var retrievers []balance_retrievers.BalanceRetriever
var balanceGauge *prometheus.GaugeVec

var mutex sync.RWMutex

func init() {
    balanceGauge = prometheus.NewGaugeVec(
        prometheus.GaugeOpts{
            Subsystem: "balance",
            Name:      "selectel",
            Help:      "Balance for service in selectel account",
        },
        []string{"service"},
    )

    prometheus.MustRegister(balanceGauge)
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
        registerRetriever(balance_retrievers.NewSelectelRetriever(balance_retrievers.SelectelConfig{ApiKey: apiKey}))
    }

    log.Println("Starting Selectel balance exporter", version.Info())
    log.Println("Build context", version.BuildContext())

    loadBalance()

    interval := v.GetInt("interval")
    go startBalanceUpdater(interval)

    srv := &http.Server{
        Addr:         *addr,
        WriteTimeout: time.Second * 2,
        ReadTimeout:  time.Second * 2,
        IdleTimeout:  time.Second * 60,

        Handler: nil,
    }

    http.Handle("/metrics", promhttp.Handler())
    http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        http.ServeFile(w, r, "static/index.html")
    })

    go func() {
        log.Fatal(srv.ListenAndServe())
    }()

    log.Printf("Selectel balance exporter has been started at address %s", *addr)

    c := make(chan os.Signal, 1)

    signal.Notify(c, os.Interrupt)
    signal.Notify(c, syscall.SIGTERM)

    <-c

    log.Println("Selectel balance exporter shutdown")
    ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
    defer cancel()

    err := srv.Shutdown(ctx)
    if err != nil {
        log.Fatal(err)
    }

    os.Exit(0)
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

func registerRetriever(fetcher balance_retrievers.BalanceRetriever) {
    mutex.Lock()
    retrievers = append(retrievers, fetcher)
    mutex.Unlock()
}

func startBalanceUpdater(interval int) {
    for {
        time.Sleep(time.Second * time.Duration(interval))
        loadBalance()
    }
}

func loadBalance() {
    for _, f := range retrievers {
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
