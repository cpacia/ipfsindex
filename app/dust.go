package app

import "github.com/cpacia/BitcoinCash-Wallet"

// Set equal to one USD penny. Must use exchange rate provider for this.
func MinimumInputSize(w *bitcoincash.SPVWallet) (uint64, error) {
	rate, err := w.ExchangeRates().GetExchangeRate("USD")
	if err != nil {
		return 0, err
	}
	return uint64(((float64(2) / 100) / rate) * 100000000), nil
}
