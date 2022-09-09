package indicator

import (
	"fmt"
	"math"
	"time"

	"github.com/c9s/bbgo/pkg/datatype/floats"
	"github.com/c9s/bbgo/pkg/types"
)

// Refer: Commodity Channel Index
// Refer URL: http://www.andrewshamlet.net/2017/07/08/python-tutorial-cci
// with modification of ddof=0 to let standard deviation to be divided by N instead of N-1
//
//go:generate callbackgen -type CCI
type CCI struct {
	types.SeriesBase
	types.IntervalWindow
	Input        floats.Slice
	TypicalPrice floats.Slice
	MA           floats.Slice
	Values       floats.Slice
	EndTime      time.Time

	UpdateCallbacks []func(value float64)

	Sqrt bool
}

func (inc *CCI) Update(value float64) {
	if len(inc.TypicalPrice) == 0 {
		inc.SeriesBase.Series = inc
		inc.TypicalPrice.Push(value)
		inc.Input.Push(value)
		return
	} else if len(inc.TypicalPrice) > MaxNumOfEWMA {
		inc.TypicalPrice = inc.TypicalPrice[MaxNumOfEWMATruncateSize-1:]
		inc.Input = inc.Input[MaxNumOfEWMATruncateSize-1:]
	}

	inc.Input.Push(value)
	tp := inc.TypicalPrice.Last() - inc.Input.Index(inc.Window) + value
	inc.TypicalPrice.Push(tp)
	if len(inc.Input) < inc.Window {
		return
	}
	ma := tp / float64(inc.Window)
	inc.MA.Push(ma)
	if len(inc.MA) > MaxNumOfEWMA {
		inc.MA = inc.MA[MaxNumOfEWMATruncateSize-1:]
	}
	md := 0.

	if inc.Sqrt {
		for i := 0; i < inc.Window; i++ {
			diff := inc.Input.Index(i) - ma
			md += diff * diff
		}
		md = math.Sqrt(md / float64(inc.Window))
		fmt.Println("sqrt")
	} else {
		for i := 0; i < inc.Window; i++ {
			diff := inc.Input.Index(i) - ma
			md += math.Abs(diff)
		}
		md = (md / float64(inc.Window))
		//fmt.Println("simple")

	}
	//fmt.Println("value,ma,md", value, ma, md)
	cci := (value - ma) / (0.015 * md)

	inc.Values.Push(cci)
	if len(inc.Values) > MaxNumOfEWMA {
		inc.Values = inc.Values[MaxNumOfEWMATruncateSize-1:]
	}
}

func (inc *CCI) Last() float64 {
	if len(inc.Values) == 0 {
		return 0
	}
	return inc.Values[len(inc.Values)-1]
}

func (inc *CCI) Index(i int) float64 {
	if i >= len(inc.Values) {
		return 0
	}
	return inc.Values[len(inc.Values)-1-i]
}

func (inc *CCI) Length() int {
	return len(inc.Values)
}

var _ types.SeriesExtend = &CCI{}

var sliceKline = []types.KLine{}

func (inc *CCI) PushK(k types.KLine) {

	if time.Now().Before(k.EndTime.Time()) || k.EndTime.Before(inc.EndTime) || inc.EndTime.Equal(k.EndTime.Time()) {
		//fmt.Println(time.Now().Before(k.EndTime.Time()), k.EndTime.Before(inc.EndTime), inc.EndTime.Equal(k.EndTime.Time()))
		return
	}
	//fmt.Println(inc.EndTime, k.EndTime.Time(), inc.EndTime.Equal(k.EndTime.Time()))
	//kline := k
	//fmt.Printf("pushk o:%.4f,l:%.4f,c:%.4f,start: %s,closed:%t \n", kline.Open.Float64(), kline.Low.Float64(), kline.Close.Float64(), kline.StartTime, k.Closed)
	//
	//sliceKline = append(sliceKline, k)
	inc.Update(k.High.Add(k.Low).Add(k.Close).Div(three).Float64())
	//if len(sliceKline) > 11 {
	//	for i := len(sliceKline) - 1; i > len(sliceKline)-10; i-- {
	//		kline := sliceKline[i]
	//		fmt.Printf("\no:%.4f,l:%.4f,c:%.4f,start: %s,end:%s closed:%t \n", kline.Open.Float64(), kline.Low.Float64(), kline.Close.Float64(), kline.StartTime, kline.EndTime, kline.Closed)
	//	}
	//}

	inc.EndTime = k.EndTime.Time()
	inc.EmitUpdate(inc.Last())
}

// cci 实时计算
func (inc *CCI) RealPushK(k types.KLine) (v float64) {
	value := k.High.Add(k.Low).Add(k.Close).Div(three).Float64()
	//tp := inc.TypicalPrice.Last() + value
	tp := inc.TypicalPrice.Last() - inc.Input.Index(inc.Window) + value

	ma := tp / float64(inc.Window)

	md := math.Abs(value - ma)

	for i := 0; i < inc.Window-1; i++ {
		diff := inc.Input.Index(i) - ma
		md += math.Abs(diff)
	}
	md = (md / float64(inc.Window))
	//fmt.Println("simple")

	//fmt.Println("value,ma,md", value, ma, md)
	cci := (value - ma) / (0.015 * md)
	return cci
	//
	//
	//
	//if inc.EndTime != zeroTime && k.EndTime.Before(inc.EndTime) {
	//	return
	//}
	//fmt.Println("k.Closed:", k.Closed, inc.Values.Length())
	//if !k.Closed {
	//	inc.Input.Pop(int64(inc.Input.Length() - 1))
	//	inc.Values.Pop(int64(inc.Values.Length() - 1))
	//	inc.MA.Pop(int64(inc.MA.Length() - 1))
	//	inc.TypicalPrice.Pop(int64(inc.TypicalPrice.Length() - 1))
	//	fmt.Println("repush cci")
	//	//inc.Values[len(inc.Values)-1]
	//	//spew.Dump(inc.Values)
	//	inc.Update(k.High.Add(k.Low).Add(k.Close).Div(three).Float64())
	//	//fmt.Println("-2,-1,0,len:", inc.MA.Index(len(inc.Values)-3), inc.MA.Index(len(inc.Values)-2), inc.MA.Last(), inc.MA.Length())
	//	//fmt.Println("-2,-1,0,len:", inc.Values.Index(len(inc.Values)-3), inc.Values.Index(len(inc.Values)-2), inc.Values.Last(), inc.Values.Length(), k.High.Add(k.Low).Add(k.Close).Div(three).Float64())
	//
	//	return
	//}

}

func (inc *CCI) CalculateAndUpdate(allKLines []types.KLine) {
	if inc.TypicalPrice.Length() == 0 {
		for _, k := range allKLines {
			inc.PushK(k)
			inc.EmitUpdate(inc.Last())
		}
	} else {
		k := allKLines[len(allKLines)-1]
		inc.PushK(k)
		inc.EmitUpdate(inc.Last())
	}
}

func (inc *CCI) handleKLineWindowUpdate(interval types.Interval, window types.KLineWindow) {
	if inc.Interval != interval {
		return
	}

	inc.CalculateAndUpdate(window)
}

func (inc *CCI) Bind(updater KLineWindowUpdater) {
	updater.OnKLineWindowUpdate(inc.handleKLineWindowUpdate)
}
