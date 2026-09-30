package configs

import (
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v2"
)

func TestValidateShops_StorageSeizureMinValueDefault(t *testing.T) {
	b := &Balance{}
	b.validateShops()
	if int(b.StorageSeizureMinValue) != 250 {
		t.Errorf("StorageSeizureMinValue default = %d, want 250", int(b.StorageSeizureMinValue))
	}
}

func TestValidateShops_StorageSeizureMinValuePreserved(t *testing.T) {
	b := &Balance{StorageSeizureMinValue: 1000}
	b.validateShops()
	if int(b.StorageSeizureMinValue) != 1000 {
		t.Errorf("StorageSeizureMinValue = %d, want 1000 (explicit value preserved)", int(b.StorageSeizureMinValue))
	}
}

// The shelf cap (baubles slice D) is a SHOP ECONOMY knob: default 12, an
// explicit value kept.
func TestValidateShops_ShopAffixedStockCapDefault(t *testing.T) {
	b := &Balance{}
	b.validateShops()
	if int(b.ShopAffixedStockCap) != 12 {
		t.Errorf("ShopAffixedStockCap default = %d, want 12", int(b.ShopAffixedStockCap))
	}
	b = &Balance{ShopAffixedStockCap: 5}
	b.validateShops()
	if int(b.ShopAffixedStockCap) != 5 {
		t.Errorf("ShopAffixedStockCap = %d, want 5 (explicit value preserved)", int(b.ShopAffixedStockCap))
	}
}

// The shipped config names the knob, at the Go default, so the live value is
// visible in config.yaml rather than silently the default.
func TestShopAffixedStockCapShipsAtTwelve(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "_datafiles", "config.yaml"))
	if err != nil {
		t.Fatalf("read shipped config: %v", err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("decode shipped config: %v", err)
	}
	if int(cfg.Balance.ShopAffixedStockCap) != 12 {
		t.Errorf("shipped ShopAffixedStockCap = %d, want 12 (0 means the key is missing)", int(cfg.Balance.ShopAffixedStockCap))
	}
}
