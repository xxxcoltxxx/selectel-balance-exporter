package main

import (
    "context"
    "errors"
    "flag"
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promhttp"
    "github.com/prometheus/common/version"
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
var interval = flag.Int("interval", 3600, "Interval (in seconds) for request balance.")
var retryInterval = flag.Int("retry-interval", 10, "Interval (in seconds) for load balance when errors.")
var retryLimit = flag.Int("retry-limit", 10, "Count of tries when error.")

var (
    retrievers   []balance_retrievers.BalanceRetriever
    balanceGauge *prometheus.GaugeVec
    mutex        sync.RWMutex
    hasError     = false
    retryCount   = 0
)

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

    flag.Parse()
}

func main() {
    log.Println("Starting Selectel balance exporter", version.Info())
    log.Println("Build context", version.BuildContext())

    config, err := readConfig()
    if err != nil {
        log.Fatalln(err)
    }

    registerRetriever(balance_retrievers.NewSelectelRetriever(config))

    if err := loadBalance(); err != nil {
        log.Fatalln(err)
    }

    go startBalanceUpdater()

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
        log.Fatalln(srv.ListenAndServe())
    }()

    log.Printf("Selectel balance exporter has been started at address %s\n", *addr)
    log.Printf("Exporter will update balance every %d seconds\n", *interval)

    c := make(chan os.Signal, 1)
    signal.Notify(c, os.Interrupt)
    signal.Notify(c, syscall.SIGTERM)

    <-c

    log.Println("Selectel balance exporter shutdown")
    ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
    defer cancel()

    if err := srv.Shutdown(ctx); err != nil {
        log.Fatalln(err)
    }

    os.Exit(0)
}

func readConfig() (balance_retrievers.SelectelConfig, error) {
    var config balance_retrievers.SelectelConfig

    if apiKey, ok := os.LookupEnv("SELECTEL_API_KEY"); ok {
        config.ApiKey = apiKey
    } else {
        return balance_retrievers.SelectelConfig{}, errors.New("environment \"SELECTEL_API_KEY\" is not set")
    }

    return config, nil
}

func registerRetriever(fetcher balance_retrievers.BalanceRetriever) {
    mutex.Lock()
    retrievers = append(retrievers, fetcher)
    mutex.Unlock()
}

func startBalanceUpdater() {
    for {
        if hasError {
            log.Printf("Request will retry after %d seconds\n", *retryInterval)
            time.Sleep(time.Second * time.Duration(*retryInterval))
        } else {
            time.Sleep(time.Second * time.Duration(*interval))
        }

        if err := loadBalance(); err != nil {
            log.Println(err.Error())
            hasError = true
            retryCount++
            if retryCount >= *retryLimit {
                log.Printf("Retry limit %d has been exceeded\n", *retryLimit)
                hasError = false
                retryCount = 0
            }
        } else {
            hasError = false
            retryCount = 0
        }
    }
}

func loadBalance() error {
    for _, f := range retrievers {
        if results, err := f.GetBalance(); err != nil {
            return err
        } else {
            for _, b := range results {
                balanceGauge.With(prometheus.Labels{"service": b.Name}).Set(b.Balance)
            }
        }
    }

    return nil
}
