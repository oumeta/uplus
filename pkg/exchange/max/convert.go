package max

import (
	"fmt"
	"strings"
	"time"

	"github.com/c9s/bbgo/pkg/exchange/max/maxapi"
	"github.com/c9s/bbgo/pkg/fixedpoint"
	"github.com/c9s/bbgo/pkg/types"
)

func toGlobalCurrency(currency string) string {
	return strings.ToUpper(currency)
}

func toLocalCurrency(currency string) string {
	return strings.ToLower(currency)
}

func toLocalSymbol(symbol string) string {
	return strings.ToLower(symbol)
}

func toGlobalSymbol(symbol string) string {
	return strings.ToUpper(symbol)
}

func toLocalSideType(side types.SideType) string {
	return strings.ToLower(string(side))
}

func toGlobalSideType(v string) types.SideType {
	switch strings.ToLower(v) {
	case "bid", "buy":
		return types.SideTypeBuy

	case "ask", "sell":
		return types.SideTypeSell

	case "self-trade":
		return types.SideTypeSelf

	}

	return types.SideType(v)
}

func toGlobalRewards(maxRewards []max.Reward) ([]types.Reward, error) {
	// convert to global reward
	var rewards []types.Reward
	for _, r := range maxRewards {
		// ignore "accepted"
		if r.State != "done" {
			continue
		}

		reward, err := r.Reward()
		if err != nil {
			return nil, err
		}

		rewards = append(rewards, *reward)
	}

	return rewards, nil
}

func toGlobalOrderStatus(orderState max.OrderState, executedVolume, remainingVolume fixedpoint.Value) types.OrderStatus {
	switch orderState {

	case max.OrderStateCancel:
		return types.OrderStatusCanceled

	case max.OrderStateFinalizing, max.OrderStateDone:
		if executedVolume.IsZero() {
			return types.OrderStatusCanceled
		} else if remainingVolume.IsZero() {
			return types.OrderStatusFilled
		}

		return types.OrderStatusFilled

	case max.OrderStateWait:
		if executedVolume.Sign() > 0 && remainingVolume.Sign() > 0 {
			return types.OrderStatusPartiallyFilled
		}

		return types.OrderStatusNew

	case max.OrderStateConvert:
		if executedVolume.Sign() > 0 && remainingVolume.Sign() > 0 {
			return types.OrderStatusPartiallyFilled
		}

		return types.OrderStatusNew

	case max.OrderStateFailed:
		return types.OrderStatusRejected

	}

	log.Errorf("unknown order status: %v", orderState)
	return types.OrderStatus(orderState)
}

func toGlobalOrderType(orderType max.OrderType) types.OrderType {
	switch orderType {
	case max.OrderTypeLimit:
		return types.OrderTypeLimit

	case max.OrderTypeMarket:
		return types.OrderTypeMarket

	case max.OrderTypeStopLimit:
		return types.OrderTypeStopLimit

	case max.OrderTypeStopMarket:
		return types.OrderTypeStopMarket

	case max.OrderTypeIOCLimit:
		return types.OrderTypeLimit

	case max.OrderTypePostOnly:
		return types.OrderTypeLimitMaker

	}

	log.Errorf("order convert error, unknown order type: %v", orderType)
	return types.OrderType(orderType)
}

func toLocalOrderType(orderType types.OrderType) (max.OrderType, error) {
	switch orderType {

	case types.OrderTypeStopLimit:
		return max.OrderTypeStopLimit, nil

	case types.OrderTypeStopMarket:
		return max.OrderTypeStopMarket, nil

	case types.OrderTypeLimitMaker:
		return max.OrderTypePostOnly, nil

	case types.OrderTypeLimit:
		return max.OrderTypeLimit, nil

	case types.OrderTypeMarket:
		return max.OrderTypeMarket, nil
	}

	return "", fmt.Errorf("order type %s not supported", orderType)
}

func toGlobalOrders(maxOrders []max.Order) (orders []types.Order, err error) {
	for _, localOrder := range maxOrders {
		o, err := toGlobalOrder(localOrder)
		if err != nil {
			log.WithError(err).Error("order convert error")
		}

		orders = append(orders, *o)
	}

	return orders, err
}

func toGlobalOrder(maxOrder max.Order) (*types.Order, error) {
	executedVolume := maxOrder.ExecutedVolume
	remainingVolume := maxOrder.RemainingVolume
	isMargin := maxOrder.WalletType == max.WalletTypeMargin

	return &types.Order{
		SubmitOrder: types.SubmitOrder{
			ClientOrderID: maxOrder.ClientOID,
			Symbol:        toGlobalSymbol(maxOrder.Market),
			Side:          toGlobalSideType(maxOrder.Side),
			Type:          toGlobalOrderType(maxOrder.OrderType),
			Quantity:      maxOrder.Volume,
			Price:         maxOrder.Price,
			TimeInForce:   types.TimeInForceGTC, // MAX only supports GTC
			GroupID:       maxOrder.GroupID,
		},
		Exchange:         types.ExchangeMax,
		IsWorking:        maxOrder.State == max.OrderStateWait,
		OrderID:          maxOrder.ID,
		Status:           toGlobalOrderStatus(maxOrder.State, executedVolume, remainingVolume),
		ExecutedQuantity: executedVolume,
		CreationTime:     types.Time(maxOrder.CreatedAt.Time()),
		UpdateTime:       types.Time(maxOrder.CreatedAt.Time()),
		IsMargin:         isMargin,
		IsIsolated:       false, // isolated margin is not supported
	}, nil
}

func toGlobalTrade(t max.Trade) (*types.Trade, error) {
	isMargin := t.WalletType == max.WalletTypeMargin
	side := toGlobalSideType(t.Side)
	return &types.Trade{
		ID:            t.ID,
		OrderID:       t.OrderID,
		Price:         t.Price,
		Symbol:        toGlobalSymbol(t.Market),
		Exchange:      types.ExchangeMax,
		Quantity:      t.Volume,
		Side:          side,
		IsBuyer:       t.IsBuyer(),
		IsMaker:       t.IsMaker(),
		Fee:           t.Fee,
		FeeCurrency:   toGlobalCurrency(t.FeeCurrency),
		QuoteQuantity: t.Funds,
		Time:          types.Time(t.CreatedAt),
		IsMargin:      isMargin,
		IsIsolated:    false,
		IsFutures:     false,
	}, nil
}

func toGlobalDepositStatus(a string) types.DepositStatus {
	switch a {
	case "submitting", "submitted", "checking":
		return types.DepositPending

	case "accepted":
		return types.DepositSuccess

	case "rejected":
		return types.DepositRejected

	case "canceled":
		return types.DepositCancelled

	case "suspect", "refunded":

	}

	return types.DepositStatus(a)
}

func convertWebSocketTrade(t max.TradeUpdate) (*types.Trade, error) {
	// skip trade ID that is the same. however this should not happen
	var side = toGlobalSideType(t.Side)

	// trade time
	mts := time.Unix(0, t.Timestamp*int64(time.Millisecond))

	price, err := fixedpoint.NewFromString(t.Price)
	if err != nil {
		return nil, err
	}

	quantity, err := fixedpoint.NewFromString(t.Volume)
	if err != nil {
		return nil, err
	}

	quoteQuantity := price.Mul(quantity)

	fee, err := fixedpoint.NewFromString(t.Fee)
	if err != nil {
		return nil, err
	}

	return &types.Trade{
		ID:            t.ID,
		OrderID:       t.OrderID,
		Symbol:        toGlobalSymbol(t.Market),
		Exchange:      types.ExchangeMax,
		Price:         price,
		Quantity:      quantity,
		Side:          side,
		IsBuyer:       side == types.SideTypeBuy,
		IsMaker:       t.Maker,
		Fee:           fee,
		FeeCurrency:   toGlobalCurrency(t.FeeCurrency),
		QuoteQuantity: quoteQuantity,
		Time:          types.Time(mts),
	}, nil
}

func convertWebSocketOrderUpdate(u max.OrderUpdate) (*types.Order, error) {
	timeInForce := types.TimeInForceGTC
	if u.OrderType == max.OrderTypeIOCLimit {
		timeInForce = types.TimeInForceIOC
	}

	return &types.Order{
		SubmitOrder: types.SubmitOrder{
			ClientOrderID: u.ClientOID,
			Symbol:        toGlobalSymbol(u.Market),
			Side:          toGlobalSideType(u.Side),
			Type:          toGlobalOrderType(u.OrderType),
			Quantity:      u.Volume,
			Price:         u.Price,
			StopPrice:     u.StopPrice,
			TimeInForce:   timeInForce, // MAX only supports GTC
			GroupID:       u.GroupID,
		},
		Exchange:         types.ExchangeMax,
		OrderID:          u.ID,
		Status:           toGlobalOrderStatus(u.State, u.ExecutedVolume, u.RemainingVolume),
		ExecutedQuantity: u.ExecutedVolume,
		CreationTime:     types.Time(time.Unix(0, u.CreatedAtMs*int64(time.Millisecond))),
		UpdateTime:       types.Time(time.Unix(0, u.CreatedAtMs*int64(time.Millisecond))),
	}, nil
}