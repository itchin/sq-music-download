package util

import (
    "fmt"
    "github.com/dustin/go-humanize"
    "github.com/shopspring/decimal"
    "strconv"
    "strings"
)

type WriteCounter struct {
    Total uint64
    Size uint64
    ratio decimal.Decimal
}

func (wc *WriteCounter) Init() {
    wc.ratio = decimal.NewFromFloat(float64(100)).Div(decimal.NewFromFloat(float64(wc.Size)))
    //fmt.Println("比例", wc.ratio)
}

func (wc *WriteCounter) Write(p []byte) (int, error) {
    n := len(p)
    wc.Total += uint64(n)
    wc.PrintProgress()
    return n, nil
}

func (wc *WriteCounter) PrintProgress() {
    fmt.Printf("\r%s", strings.Repeat(" ", 35))
    s := fmt.Sprintf("%s", decimal.NewFromFloat(float64(wc.Total)).Mul(wc.ratio))
    f, _ := strconv.ParseFloat(s, 32)
    fmt.Printf("\rDownloading... %s %s%s", humanize.Bytes(wc.Total), fmt.Sprintf("%0.2f", f), "% complete")
}
