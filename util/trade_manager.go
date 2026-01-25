package util

import (
	"backend/model"
	"sync"
)

type TradeManager struct {
	mu           sync.RWMutex
	activeTrades map[string]*model.Order
}

func NewTradeManager() *TradeManager {
	return &TradeManager{
		activeTrades: make(map[string]*model.Order),
	}
}

func (m *TradeManager) AddTrade(order *model.Order) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.activeTrades[order.ID] = order
}

func (m *TradeManager) RemoveTrade(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.activeTrades, id)
}

func (m *TradeManager) GetActiveList() []*model.Order {
	m.mu.RLock()
	defer m.mu.RUnlock()

	list := make([]*model.Order, 0, len(m.activeTrades))
	for _, order := range m.activeTrades {
		list = append(list, order)
	}
	return list
}

func (m *TradeManager) ReplaceAll(newTrades map[string]*model.Order) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.activeTrades = newTrades
}
