package main

import "testing"

func TestCurrentShopExchangeSessionGuard(t *testing.T) {
	valid := currentShopExchangeSession{
		ShopID: "Shop_dy_20216", NPCObjectID: 345, NPCOwnerID: 1,
		NPCConfigID: "npc_shop_test", SceneEpoch: 7,
	}
	for _, tc := range []struct {
		name string
		session currentShopExchangeSession
		shop string
		epoch uint64
		id uint32
		owner uint32
		config string
		advertised bool
		want bool
	}{
		{"selected-service",valid,"Shop_dy_20216",7,345,1,"npc_shop_test",true,true},
		{"no-open-service",currentShopExchangeSession{},"Shop_dy_20216",7,345,1,"npc_shop_test",true,false},
		{"client-forged-shop",valid,"another_shop",7,345,1,"npc_shop_test",true,false},
		{"scene-transition",valid,"Shop_dy_20216",8,345,1,"npc_shop_test",true,false},
		{"npc-removed",valid,"Shop_dy_20216",7,0,0,"",false,false},
		{"npc-id-reused",valid,"Shop_dy_20216",7,345,1,"different_npc",true,false},
		{"npc-owner-mismatch",valid,"Shop_dy_20216",7,345,2,"npc_shop_test",true,false},
		{"service-changed",valid,"Shop_dy_20216",7,345,1,"npc_shop_test",false,false},
		{"empty-shop",valid,"",7,345,1,"npc_shop_test",true,false},
	} {
		t.Run(tc.name,func(t *testing.T) {
			got := tc.session.allows(tc.shop,tc.epoch,tc.id,tc.owner,tc.config,tc.advertised)
			if got != tc.want { t.Fatalf("session.allows()=%v want=%v",got,tc.want) }
		})
	}
}
