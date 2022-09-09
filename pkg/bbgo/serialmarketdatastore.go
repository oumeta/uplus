package bbgo

import (
	"fmt"
	"time"

	"github.com/c9s/bbgo/pkg/types"
)

type SerialMarketDataStore struct {
	*MarketDataStore
	KLines       map[types.Interval]*types.KLine
	Subscription []types.Interval
}

func NewSerialMarketDataStore(symbol string) *SerialMarketDataStore {
	return &SerialMarketDataStore{
		MarketDataStore: NewMarketDataStore(symbol),
		KLines:          make(map[types.Interval]*types.KLine),
		Subscription:    []types.Interval{},
	}
}

func (store *SerialMarketDataStore) Subscribe(interval types.Interval) {
	// dedup
	for _, i := range store.Subscription {
		if i == interval {
			return
		}
	}
	store.Subscription = append(store.Subscription, interval)
}

func (store *SerialMarketDataStore) BindStream(stream types.Stream) {
	stream.OnKLineClosed(store.handleKLineClosed)
	stream.OnKLine(store.handleKLineUpdate)

}

func (store *SerialMarketDataStore) handleKLineClosed(kline types.KLine) {
	store.AddKLine(kline)
}

func (store *SerialMarketDataStore) handleKLine(kline types.KLine) {
	store.AddKLine(kline)
}

func (store *SerialMarketDataStore) AddKLine(kline types.KLine) {
	if kline.Symbol != store.Symbol {
		return
	}
	// only consumes kline1m
	if kline.Interval != types.Interval1m {
		return
	}
	// endtime in minutes
	timestamp := kline.StartTime.Time().Add(time.Minute)
	for _, val := range store.Subscription {
		k, ok := store.KLines[val]
		//fmt.Println(k, ok)

		if !ok {
			k = &types.KLine{}
			k.Set(&kline)
			k.Interval = val
			k.Closed = false
			store.KLines[val] = k
		} else {
			k.Merge(&kline)
			k.Closed = false
		}
		//fmt.Println(timestamp.Truncate(val.Duration()))
		if timestamp.Truncate(val.Duration()) == timestamp {
			k.Closed = true

			fmt.Printf("kline:%.4f,%.4f,%.4f,start%s,end%s,close:%t \n", k.Open.Float64(), k.Low.Float64(), k.Close.Float64(), k.StartTime, k.EndTime, k.Closed)
			store.MarketDataStore.AddKLine(*k)
			delete(store.KLines, val)
		}
	}
}
