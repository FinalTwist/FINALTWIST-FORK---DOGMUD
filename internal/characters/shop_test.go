package characters

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestShop_StockItem(t *testing.T) {
	tests := []struct {
		name           string
		initialShop    Shop
		stockItemId    int
		expectedShop   Shop
		expectedResult bool
	}{
		{
			name:        "Stock existing item increases quantity",
			initialShop: Shop{{ItemId: 1, Quantity: 2, QuantityMax: 5}},
			stockItemId: 1,
			expectedShop: Shop{
				{ItemId: 1, Quantity: 3, QuantityMax: 5},
			},
			expectedResult: true,
		},
		{
			name:        "Stock new item adds to shop",
			initialShop: Shop{},
			stockItemId: 2,
			expectedShop: Shop{
				{ItemId: 2, Quantity: 1, QuantityMax: StockTemporary},
			},
			expectedResult: true,
		},
		{
			name: "Stock one existing, one new",
			initialShop: Shop{
				{ItemId: 1, Quantity: 1, QuantityMax: 3},
			},
			stockItemId: 2,
			expectedShop: Shop{
				{ItemId: 1, Quantity: 1, QuantityMax: 3},
				{ItemId: 2, Quantity: 1, QuantityMax: StockTemporary},
			},
			expectedResult: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			shop := tt.initialShop
			result := shop.StockItem(tt.stockItemId)
			assert.Equal(t, tt.expectedResult, result)
			assert.Equal(t, len(tt.expectedShop), len(shop))
			for i := range tt.expectedShop {
				assert.Equal(t, tt.expectedShop[i].ItemId, shop[i].ItemId)
				assert.Equal(t, tt.expectedShop[i].Quantity, shop[i].Quantity)
				assert.Equal(t, tt.expectedShop[i].QuantityMax, shop[i].QuantityMax)
			}
		})
	}
}
func TestShop_Destock(t *testing.T) {
	tests := []struct {
		name           string
		initialShop    Shop
		destockItem    ShopItem
		expectedShop   Shop
		expectedResult bool
	}{
		{
			name: "Destock existing item with quantity > 1",
			initialShop: Shop{
				{ItemId: 1, Quantity: 2, QuantityMax: 5},
			},
			destockItem: ShopItem{ItemId: 1},
			expectedShop: Shop{
				{ItemId: 1, Quantity: 1, QuantityMax: 5},
			},
			expectedResult: true,
		},
		{
			name: "Destock existing item with quantity == 1 and temporary",
			initialShop: Shop{
				{ItemId: 2, Quantity: 1, QuantityMax: StockTemporary},
			},
			destockItem:    ShopItem{ItemId: 2},
			expectedShop:   Shop{},
			expectedResult: true,
		},
		{
			name: "Destock existing item with quantity == 1 and non-temporary",
			initialShop: Shop{
				{ItemId: 3, Quantity: 1, QuantityMax: 5},
			},
			destockItem: ShopItem{ItemId: 3},
			expectedShop: Shop{
				{ItemId: 3, Quantity: 0, QuantityMax: 5},
			},
			expectedResult: true,
		},
		{
			name: "Destock item with unlimited quantity",
			initialShop: Shop{
				{ItemId: 4, Quantity: 0, QuantityMax: StockUnlimited},
			},
			destockItem: ShopItem{ItemId: 4},
			expectedShop: Shop{
				{ItemId: 4, Quantity: 0, QuantityMax: StockUnlimited},
			},
			expectedResult: true,
		},
		{
			name: "Destock non-existing item returns false",
			initialShop: Shop{
				{ItemId: 5, Quantity: 2, QuantityMax: 5},
			},
			destockItem: ShopItem{ItemId: 6},
			expectedShop: Shop{
				{ItemId: 5, Quantity: 2, QuantityMax: 5},
			},
			expectedResult: false,
		},
		{
			name: "Destock with MobId and ConditionId match",
			initialShop: Shop{
				{ItemId: 7, MobId: 1, ConditionId: 2, Quantity: 2, QuantityMax: 5},
			},
			destockItem: ShopItem{ItemId: 7, MobId: 1, ConditionId: 2},
			expectedShop: Shop{
				{ItemId: 7, MobId: 1, ConditionId: 2, Quantity: 1, QuantityMax: 5},
			},
			expectedResult: true,
		},
		{
			name: "Destock with MobId mismatch returns false",
			initialShop: Shop{
				{ItemId: 8, MobId: 1, ConditionId: 2, Quantity: 2, QuantityMax: 5},
			},
			destockItem: ShopItem{ItemId: 8, MobId: 2, ConditionId: 2},
			expectedShop: Shop{
				{ItemId: 8, MobId: 1, ConditionId: 2, Quantity: 2, QuantityMax: 5},
			},
			expectedResult: false,
		},
		{
			name: "Destock with ConditionId mismatch returns false",
			initialShop: Shop{
				{ItemId: 9, MobId: 1, ConditionId: 2, Quantity: 2, QuantityMax: 5},
			},
			destockItem: ShopItem{ItemId: 9, MobId: 1, ConditionId: 3},
			expectedShop: Shop{
				{ItemId: 9, MobId: 1, ConditionId: 2, Quantity: 2, QuantityMax: 5},
			},
			expectedResult: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			shop := tt.initialShop
			result := shop.Destock(tt.destockItem)
			assert.Equal(t, tt.expectedResult, result)
			assert.Equal(t, len(tt.expectedShop), len(shop))
			for i := range tt.expectedShop {
				assert.Equal(t, tt.expectedShop[i].ItemId, shop[i].ItemId)
				assert.Equal(t, tt.expectedShop[i].MobId, shop[i].MobId)
				assert.Equal(t, tt.expectedShop[i].ConditionId, shop[i].ConditionId)
				assert.Equal(t, tt.expectedShop[i].Quantity, shop[i].Quantity)
				assert.Equal(t, tt.expectedShop[i].QuantityMax, shop[i].QuantityMax)
			}
		})
	}
}
func TestShop_GetInstock(t *testing.T) {
	tests := []struct {
		name         string
		shop         Shop
		expectedShop Shop
	}{
		{
			name: "All items in stock",
			shop: Shop{
				{ItemId: 1, Quantity: 2, QuantityMax: 5},
				{ItemId: 2, Quantity: 1, QuantityMax: 3},
			},
			expectedShop: Shop{
				{ItemId: 1, Quantity: 2, QuantityMax: 5},
				{ItemId: 2, Quantity: 1, QuantityMax: 3},
			},
		},
		{
			name: "Some items out of stock",
			shop: Shop{
				{ItemId: 1, Quantity: 0, QuantityMax: 5},
				{ItemId: 2, Quantity: 3, QuantityMax: 3},
			},
			expectedShop: Shop{
				{ItemId: 2, Quantity: 3, QuantityMax: 3},
			},
		},
		{
			name: "Unlimited stock item always in stock",
			shop: Shop{
				{ItemId: 1, Quantity: 0, QuantityMax: StockUnlimited},
				{ItemId: 2, Quantity: 0, QuantityMax: 5},
			},
			expectedShop: Shop{
				{ItemId: 1, Quantity: 0, QuantityMax: StockUnlimited},
			},
		},
		{
			name: "No items in stock",
			shop: Shop{
				{ItemId: 1, Quantity: 0, QuantityMax: 5},
				{ItemId: 2, Quantity: 0, QuantityMax: 3},
			},
			expectedShop: Shop{},
		},
		{
			name: "Mixed: in stock, out of stock, unlimited",
			shop: Shop{
				{ItemId: 1, Quantity: 0, QuantityMax: 5},
				{ItemId: 2, Quantity: 2, QuantityMax: 3},
				{ItemId: 3, Quantity: 0, QuantityMax: StockUnlimited},
			},
			expectedShop: Shop{
				{ItemId: 2, Quantity: 2, QuantityMax: 3},
				{ItemId: 3, Quantity: 0, QuantityMax: StockUnlimited},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.shop.GetInstock()
			assert.Equal(t, len(tt.expectedShop), len(got))
			for i := range tt.expectedShop {
				assert.Equal(t, tt.expectedShop[i].ItemId, got[i].ItemId)
				assert.Equal(t, tt.expectedShop[i].Quantity, got[i].Quantity)
				assert.Equal(t, tt.expectedShop[i].QuantityMax, got[i].QuantityMax)
			}
		})
	}
}
func TestShopItem_Available(t *testing.T) {
	tests := []struct {
		name     string
		item     ShopItem
		expected bool
	}{
		{
			name:     "Quantity greater than zero",
			item:     ShopItem{Quantity: 2, QuantityMax: 5},
			expected: true,
		},
		{
			name:     "Quantity zero, not unlimited",
			item:     ShopItem{Quantity: 0, QuantityMax: 5},
			expected: false,
		},
		{
			name:     "Quantity zero, unlimited stock",
			item:     ShopItem{Quantity: 0, QuantityMax: StockUnlimited},
			expected: true,
		},
		{
			name:     "Quantity negative, not unlimited",
			item:     ShopItem{Quantity: -1, QuantityMax: 5},
			expected: false,
		},
		{
			name:     "Quantity negative, unlimited stock",
			item:     ShopItem{Quantity: -1, QuantityMax: StockUnlimited},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.item.Available()
			assert.Equal(t, tt.expected, result)
		})
	}
}
