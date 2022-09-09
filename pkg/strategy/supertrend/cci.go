package supertrend

import (
	"github.com/c9s/bbgo/pkg/datatype/floats"
	"github.com/c9s/bbgo/pkg/fixedpoint"
	"github.com/c9s/bbgo/pkg/indicator"
	"github.com/c9s/bbgo/pkg/types"
	"github.com/iamjinlei/go-tart"
	"time"
)

// Developed by Donald Lambert and featured in Commodities
// magazine in 1980, the Commodity Channel Index (CCI) is
// a versatile indicator that can be used to identify a new
// trend or warn of extreme conditions. Lambert originally
// developed CCI to identify cyclical turns in commodities,
// but the indicator can be successfully applied to indices,
// ETFs, stocks and other securities. In general, CCI measures
// the current price level relative to an average price level
// over a given period of time. CCI is relatively high when
// prices are far above their average, but is relatively low
// when prices are far below their average. In this manner,
// CCI can be used to identify overbought and oversold levels.
//
//	https://school.stockcharts.com/doku.php?id=technical_indicators:commodity_channel_index_cci
//	https://www.investopedia.com/terms/c/commoditychannelindex.asp
//	https://www.fidelity.com/learning-center/trading-investing/technical-analysis/technical-indicator-guide/cci
type Cci struct {
	n            int64
	initPeriod   int64
	avg          *tart.Sma
	dev          *tart.Dev
	sz           int64
	Interval     types.Interval `json:"interval"`
	TypicalPrice floats.Slice
	EndTime      time.Time
}

var three = fixedpoint.NewFromInt(3)

var zeroTime = time.Time{}

func NewCci(n int64) *Cci {
	avg := tart.NewSma(n)
	dev := tart.NewDev(n)
	a := avg.InitPeriod()
	b := dev.InitPeriod()
	if a < b {
		a = b
	}
	return &Cci{
		n:          n,
		initPeriod: a,
		avg:        avg,
		dev:        dev,
		sz:         0,
	}
}

func (d *Cci) Update(h, l, c float64) float64 {

	if len(d.TypicalPrice) == 0 {

		d.TypicalPrice.Push(h)

		return 0
	} else if len(d.TypicalPrice) > indicator.MaxNumOfEWMA {
		d.TypicalPrice = d.TypicalPrice[indicator.MaxNumOfEWMATruncateSize-1:]

	}

	d.sz++

	m := (h + l + c) / 3.0
	avg := d.avg.Update(m)
	dev := d.dev.Update(m)

	if almostZero(dev) {
		return 0
	}

	return (m - avg) / (0.015 * dev)
}

func (d *Cci) InitPeriod() int64 {
	return d.initPeriod
}

func (d *Cci) Valid() bool {
	return d.sz > d.initPeriod
}
func (inc *Cci) PushK(k types.KLine) {
	inc.Update(k.High.Float64(), k.Low.Float64(), k.Close.Float64())
}

func (inc *Cci) RePushK(k types.KLine) {
	if inc.EndTime != zeroTime && !k.EndTime.After(inc.EndTime) {
		//inc.Input.Pop(int64(inc.Input.Length() - 1))
		//inc.Values.Pop(int64(inc.Values.Length() - 1))
		//inc.MA.Pop(int64(inc.MA.Length() - 1))
		inc.TypicalPrice.Pop(int64(inc.TypicalPrice.Length() - 1))

		//inc.Values[len(inc.Values)-1]
		//spew.Dump(inc.Values)
		inc.Update(k.High.Float64(), k.Low.Float64(), k.Close.Float64())
		//fmt.Println("-2,-1,0,len:", inc.MA.Index(len(inc.Values)-3), inc.MA.Index(len(inc.Values)-2), inc.MA.Last(), inc.MA.Length())
		//fmt.Println("-2,-1,0,len:", inc.Values.Index(len(inc.Values)-3), inc.Values.Index(len(inc.Values)-2), inc.Values.Last(), inc.Values.Length(), k.High.Add(k.Low).Add(k.Close).Div(three).Float64())

		return
	}

	inc.Update(k.High.Float64(), k.Low.Float64(), k.Close.Float64())

	inc.EndTime = k.EndTime.Time()
	//inc.EmitUpdate(inc.Last())
}

func (inc *Cci) CalculateAndUpdate(allKLines []types.KLine) {
	if inc.TypicalPrice == nil {
		for _, k := range allKLines {
			inc.PushK(k)
			//inc.EmitUpdate(inc.Last())
		}
	} else {
		// last k
		k := allKLines[len(allKLines)-1]
		inc.PushK(k)
		//inc.EmitUpdate(inc.Last())
	}
}

func (inc *Cci) handleKLineWindowUpdate(interval types.Interval, window types.KLineWindow) {
	if inc.Interval != interval {
		return
	}

	inc.CalculateAndUpdate(window)
}

func (inc *Cci) Bind(updater indicator.KLineWindowUpdater) {
	updater.OnKLineWindowUpdate(inc.handleKLineWindowUpdate)
}

// Developed by Donald Lambert and featured in Commodities
// magazine in 1980, the Commodity Channel Index (CCI) is
// a versatile indicator that can be used to identify a new
// trend or warn of extreme conditions. Lambert originally
// developed CCI to identify cyclical turns in commodities,
// but the indicator can be successfully applied to indices,
// ETFs, stocks and other securities. In general, CCI measures
// the current price level relative to an average price level
// over a given period of time. CCI is relatively high when
// prices are far above their average, but is relatively low
// when prices are far below their average. In this manner,
// CCI can be used to identify overbought and oversold levels.
//
//	https://school.stockcharts.com/doku.php?id=technical_indicators:commodity_channel_index_cci
//	https://www.investopedia.com/terms/c/commoditychannelindex.asp
//	https://www.fidelity.com/learning-center/trading-investing/technical-analysis/technical-indicator-guide/cci
func CciArr(h, l, c []float64, n int64) []float64 {
	out := make([]float64, len(h))

	d := NewCci(n)
	for i := 0; i < len(h); i++ {
		out[i] = d.Update(h[i], l[i], c[i])
	}

	return out
}
