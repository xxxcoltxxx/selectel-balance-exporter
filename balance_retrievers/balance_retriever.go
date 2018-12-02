package balance_retrievers

type BalanceRetriever interface {
    GetName() string
    GetBalance() (balances []ServiceBalance, err error)
}

type ServiceBalance struct {
    Name string
    Balance float64
}
