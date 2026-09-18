package main

import (
	"os"
	"path/filepath"
	"testing"
)

// An ordinary buy must be bound to the shop this connection actually opened
// through its NPC service. The client-provided shop ID is not authorization.
func TestOrdinaryShopBuyRequiresOpenedNPCShop(t *testing.T) {
	world := newSceneLifecycle("test", &captureMessageConnection{})
	if world.ordinaryShopBuyAuthorized("Shop_fixture") {
		t.Fatal("unopened shop was authorized")
	}

	shopFile := filepath.Join(t.TempDir(), "shop.ini")
	const catalog = "[Shop_fixture]\nPageInfo=Page1\nType=0\n0=test_item,1,1,2,0,1,0,0,0\n"
	if err := os.WriteFile(shopFile, []byte(catalog), 0o600); err != nil {
		t.Fatal(err)
	}
	original := defaultShopINIPath
	defaultShopINIPath = shopFile
	t.Cleanup(func() { defaultShopINIPath = original })
	if err := world.begin("test_scene", "test_resource"); err != nil {
		t.Fatal(err)
	}
	if world.ordinaryShopBuyAuthorized("Shop_fixture") {
		t.Fatal("shop was authorized before an NPC opened it")
	}
	world.mu.Lock()
	err := world.openSelectedServiceLocked(sceneEntity{configID: "npc_fixture"}, npcService{mark: markShop, value: "Shop_fixture"})
	world.mu.Unlock()
	if err != nil {
		t.Fatal(err)
	}
	if !world.ordinaryShopBuyAuthorized("Shop_fixture") {
		t.Fatal("the opened shop was not authorized")
	}
	if world.ordinaryShopBuyAuthorized("Shop_different") {
		t.Fatal("client-supplied different shop was authorized")
	}
	world.mu.Lock()
	err = world.openSelectedServiceLocked(sceneEntity{configID: "npc_fixture"}, npcService{mark: markShop, value: "Shop_different"})
	world.mu.Unlock()
	if err == nil || world.ordinaryShopBuyAuthorized("Shop_fixture") {
		t.Fatal("failed second shop open retained the first shop authorization")
	}
	world.mu.Lock()
	err = world.openSelectedServiceLocked(sceneEntity{configID: "npc_fixture"}, npcService{mark: markShop, value: "Shop_fixture"})
	world.mu.Unlock()
	if err != nil || !world.ordinaryShopBuyAuthorized("Shop_fixture") {
		t.Fatalf("cannot reopen legitimate shop: %v", err)
	}
	if err := world.begin("other_scene", "other_resource"); err != nil {
		t.Fatal(err)
	}
	if world.ordinaryShopBuyAuthorized("Shop_fixture") {
		t.Fatal("previous-scene shop remained authorized")
	}
	world.close()
	if world.ordinaryShopBuyAuthorized("Shop_fixture") {
		t.Fatal("closed connection retained shop authorization")
	}
}

func TestOrdinaryShopBuyCannotUseClientShopIDBeforeOpen(t *testing.T) {
	conn := &captureMessageConnection{}
	world := newSceneLifecycle("test", conn)
	player := newPlayerActor("tester", 100)
	request := clientCustomMessage{Values: []clientCustomValue{
		{Type: 2, Int32: 70}, {Type: 6, Text: "Shop_fixture"},
		{Type: 2, Int32: 0}, {Type: 2, Int32: 1}, {Type: 2, Int32: 1},
	}}
	handled, err := handleShopBuyCustom(conn, world, player, nil, nil, nil, 1, request, "test")
	if err != nil || !handled {
		t.Fatalf("unopened shop buy handled=%v err=%v", handled, err)
	}
	if world.ordinaryShopBuyAuthorized("Shop_fixture") {
		t.Fatal("unopened purchase authorized its own shop")
	}
	if len(conn.Frames()) != 0 {
		t.Fatal("unopened shop buy published frames")
	}
}
