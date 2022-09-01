package bbgo

import (
	"context"

	log "github.com/sirupsen/logrus"

	"github.com/c9s/bbgo/pkg/fixedpoint"
	"github.com/c9s/bbgo/pkg/types"
)

const enableMarketTradeStop = false

// ProtectiveStopLoss provides a way to protect your profit but also keep a room for the price volatility
// Set ActivationRatio to 1% means if the price is away from your average cost by 1%, we will activate the protective stop loss
// and the StopLossRatio is the minimal profit ratio you want to keep for your position.
// If you set StopLossRatio to 0.1% and ActivationRatio to 1%,
// when the price goes away from your average cost by 1% and then goes back to below your (average_cost * (1 - 0.1%))
// The stop will trigger.
type ProtectiveStopLoss struct {
	Symbol string `json:"symbol"`

	// ActivationRatio is the trigger condition of this ROI protection stop loss
	// When the price goes lower (for short position) with the ratio, the protection stop will be activated.
	// This number should be positive to protect the profit
	ActivationRatio fixedpoint.Value `json:"activationRatio"`

	// StopLossRatio is the ratio for stop loss. This number should be positive to protect the profit.
	// negative ratio will cause loss.
	StopLossRatio fixedpoint.Value `json:"stopLossRatio"`

	// PlaceStopOrder places the stop order on exchange and lock the balance
	PlaceStopOrder bool `json:"placeStopOrder"`

	session       *ExchangeSession
	orderExecutor *GeneralOrderExecutor
	stopLossPrice fixedpoint.Value
	stopLossOrder *types.Order
}

var one = fixedpoint.One

func (s *ProtectiveStopLoss) Subscribe(session *ExchangeSession) {
	// use 1m kline to handle roi stop
	session.Subscribe(types.KLineChannel, s.Symbol, types.SubscribeOptions{Interval: types.Interval1m})
}

func (s *ProtectiveStopLoss) shouldActivate(position *types.Position, closePrice fixedpoint.Value) bool {
	if position.IsLong() {
		r := one.Add(s.ActivationRatio)
		activationPrice := position.AverageCost.Mul(r)
		return closePrice.Compare(activationPrice) > 0
	} else if position.IsShort() {
		r := one.Sub(s.ActivationRatio)
		activationPrice := position.AverageCost.Mul(r)
		// for short position, if the close price is less than the activation price then this is a profit position.
		return closePrice.Compare(activationPrice) < 0
	}

	return false
}

func (s *ProtectiveStopLoss) placeStopOrder(ctx context.Context, position *types.Position, orderExecutor OrderExecutor) error {
	if s.stopLossOrder != nil {
		if err := orderExecutor.CancelOrders(ctx, *s.stopLossOrder); err != nil {
			log.WithError(err).Errorf("failed to cancel stop limit order: %+v", s.stopLossOrder)
		}
		s.stopLossOrder = nil
	}

	createdOrders, err := orderExecutor.SubmitOrders(ctx, types.SubmitOrder{
		Symbol:    position.Symbol,
		Side:      types.SideTypeBuy,
		Type:      types.OrderTypeStopLimit,
		Quantity:  position.GetQuantity(),
		Price:     s.stopLossPrice.Mul(one.Add(fixedpoint.NewFromFloat(0.005))), // +0.5% from the trigger price, slippage protection
		StopPrice: s.stopLossPrice,
		Market:    position.Market,
		Tag:       "protectiveStopLoss",
	})

	if len(createdOrders) > 0 {
		s.stopLossOrder = &createdOrders[0]
	}
	return err
}

func (s *ProtectiveStopLoss) shouldStop(closePrice fixedpoint.Value, position *types.Position) bool {
	if s.stopLossPrice.IsZero() {
		return false
	}

	if position.IsShort() {
		return closePrice.Compare(s.stopLossPrice) >= 0
	} else if position.IsLong() {
		return closePrice.Compare(s.stopLossPrice) <= 0
	}

	return false
}

func (s *ProtectiveStopLoss) Bind(session *ExchangeSession, orderExecutor *GeneralOrderExecutor) {
	s.session = session
	s.orderExecutor = orderExecutor

	orderExecutor.TradeCollector().OnPositionUpdate(func(position *types.Position) {
		if position.IsClosed() {
			s.stopLossOrder = nil
			s.stopLossPrice = fixedpoint.Zero
		}
	})

	session.UserDataStream.OnOrderUpdate(func(order types.Order) {
		if s.stopLossOrder == nil {
			return
		}

		if order.OrderID == s.stopLossOrder.OrderID {
			switch order.Status {
			case types.OrderStatusFilled, types.OrderStatusCanceled:
				s.stopLossOrder = nil
				s.stopLossPrice = fixedpoint.Zero
			}
		}
	})

	position := orderExecutor.Position()
	session.MarketDataStream.OnKLineClosed(func(kline types.KLine) {
		if kline.Symbol != position.Symbol || kline.Interval != types.Interval1m {
			return
		}

		isPositionOpened := !position.IsClosed() && !position.IsDust(kline.Close)
		if isPositionOpened {
			s.handleChange(context.Background(), position, kline.Close, s.orderExecutor)
		} else {
			s.stopLossPrice = fixedpoint.Zero
		}
	})

	if !IsBackTesting && enableMarketTradeStop {
		session.MarketDataStream.OnMarketTrade(func(trade types.Trade) {
			if trade.Symbol != position.Symbol {
				return
			}

			if s.stopLossPrice.IsZero() || s.PlaceStopOrder {
				return
			}

			s.checkStopPrice(trade.Price, position)
		})
	}
}

func (s *ProtectiveStopLoss) handleChange(ctx context.Context, position *types.Position, closePrice fixedpoint.Value, orderExecutor *GeneralOrderExecutor) {
	if s.stopLossOrder != nil {
		// use RESTful to query the order status
		// orderQuery := orderExecutor.Session().Exchange.(types.ExchangeOrderQueryService)
		// order, err := orderQuery.QueryOrder(ctx, types.OrderQuery{
		// 	Symbol:  s.stopLossOrder.Symbol,
		// 	OrderID: strconv.FormatUint(s.stopLossOrder.OrderID, 10),
		// })
		// if err != nil {
		// 	log.WithError(err).Errorf("query order failed")
		// }
	}

	if s.stopLossPrice.IsZero() {
		if s.shouldActivate(position, closePrice) {
			// calculate stop loss price
			if position.IsShort() {
				s.stopLossPrice = position.AverageCost.Mul(one.Sub(s.StopLossRatio))
			} else if position.IsLong() {
				s.stopLossPrice = position.AverageCost.Mul(one.Add(s.StopLossRatio))
			}

			log.Infof("[ProtectiveStopLoss] %s protection stop loss activated, current price = %f, average cost = %f, stop loss price = %f",
				position.Symbol, closePrice.Float64(), position.AverageCost.Float64(), s.stopLossPrice.Float64())

			if s.PlaceStopOrder {
				if err := s.placeStopOrder(ctx, position, orderExecutor); err != nil {
					log.WithError(err).Errorf("failed to place stop limit order")
				}
				return
			}
		} else {
			// not activated, skip setup stop order
			return
		}
	}

	// check stop price
	s.checkStopPrice(closePrice, position)
}

func (s *ProtectiveStopLoss) checkStopPrice(closePrice fixedpoint.Value, position *types.Position) {
	if s.stopLossPrice.IsZero() {
		return
	}

	if s.shouldStop(closePrice, position) {
		log.Infof("[ProtectiveStopLoss] protection stop order is triggered at price %f, position = %+v", closePrice.Float64(), position)
		if err := s.orderExecutor.ClosePosition(context.Background(), one, "protectiveStopLoss"); err != nil {
			log.WithError(err).Errorf("failed to close position")
		}
	}
}
