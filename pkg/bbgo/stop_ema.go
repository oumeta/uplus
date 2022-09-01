package bbgo

import (
	"github.com/sirupsen/logrus"

	"github.com/c9s/bbgo/pkg/fixedpoint"
	"github.com/c9s/bbgo/pkg/indicator"
	"github.com/c9s/bbgo/pkg/types"
)

type StopEMA struct {
	types.IntervalWindow
	Range fixedpoint.Value `json:"range"`

	stopEWMA *indicator.EWMA
}

func (s *StopEMA) Bind(session *ExchangeSession, orderExecutor *GeneralOrderExecutor) {
	symbol := orderExecutor.Position().Symbol
	s.stopEWMA = session.StandardIndicatorSet(symbol).EWMA(s.IntervalWindow)
}

func (s *StopEMA) Allowed(closePrice fixedpoint.Value) bool {
	ema := fixedpoint.NewFromFloat(s.stopEWMA.Last())
	if ema.IsZero() {
		logrus.Infof("stopEMA protection: value is zero, skip")
		return false
	}

	emaStopShortPrice := ema.Mul(fixedpoint.One.Sub(s.Range))
	if closePrice.Compare(emaStopShortPrice) < 0 {
		Notify("stopEMA protection: close price %f less than stopEMA %f = EMA(%f) * (1 - RANGE %f)",
			closePrice.Float64(),
			s.IntervalWindow.String(),
			emaStopShortPrice.Float64(),
			ema.Float64(),
			s.Range.Float64())
		return false
	}

	return true
}
