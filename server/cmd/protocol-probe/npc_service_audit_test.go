package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/local/9yin-go-server/internal/role"
)

func TestBuildNPCServiceAuditSeparatesConfiguredFromImplemented(t *testing.T) {
	npc := testNPCSpawn()
	npc.resolved.Extensions = map[string]string{
		"template.ShopID":      "Shop_yaopin_00100",
		"template.TransFuncID": "37",
	}
	report := buildNPCServiceAudit(role.Scene{Config: "ini\\scene\\scene_city05", Resource: "city05"}, npcCatalogStats{Resolved: 1}, []npcSpawn{npc}, nil)
	if len(report.NPCs) != 1 || len(report.NPCs[0].Services) != 2 {
		t.Fatalf("audit=%#v", report)
	}
	shop, other := report.NPCs[0].Services[0], report.NPCs[0].Services[1]
	if shop.Mark != markShop || !shop.Implemented || shop.Ready || shop.CatalogReady == nil {
		t.Fatalf("shop catalog must be distinguished from unverified purchase: %#v", shop)
	}
	if other.Implemented || other.Ready || other.CatalogReady != nil {
		t.Fatalf("unimplemented service status=%#v", other)
	}
}

func TestNPCShopAuditExactCatalogNeverAuthorizesPurchase(t *testing.T) {
	path := filepath.Join(t.TempDir(), "shop.ini")
	// This is a synthetic parser fixture, not an extracted client shop.
	fixture := "[Shop_exact_1]\nType=0\nPageInfo=Page1\n0=item_001,1,0,10,0,0,0\n"
	if err := os.WriteFile(path, []byte(fixture), 0o600); err != nil {
		t.Fatal(err)
	}
	shop := npcService{mark: markShop, value: "Shop_exact_1"}
	implemented, catalog, ready, issue := auditNPCServiceReadinessFrom(shop, path)
	if !implemented || !catalog || ready || !strings.Contains(issue, "purchase blocked") {
		t.Fatalf("exact catalog unexpectedly treated as purchasable: implemented=%t catalog=%t ready=%t issue=%q", implemented, catalog, ready, issue)
	}

	// A similar suffix is never an authorized substitute for the exact ShopID.
	shop.value = "Shop_exact"
	implemented, catalog, ready, issue = auditNPCServiceReadinessFrom(shop, path)
	if !implemented || catalog || ready || !strings.Contains(issue, "no section") {
		t.Fatalf("missing exact catalog accepted: implemented=%t catalog=%t ready=%t issue=%q", implemented, catalog, ready, issue)
	}

	implemented, catalog, ready, issue = auditNPCServiceReadinessFrom(npcService{mark: markDepot}, path)
	if !implemented || catalog || !ready || issue != "" {
		t.Fatalf("depot semantics unexpectedly changed: %t %t %t %q", implemented, catalog, ready, issue)
	}
}

func TestNPCShopCatalogReadyJSONOnlyForShop(t *testing.T) {
	available := true
	data, err := json.Marshal([]npcServiceAuditService{
		{Mark: markShop, CatalogReady: &available, Ready: false},
		{Mark: markDepot, Ready: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	var records []map[string]any
	if err := json.Unmarshal(data, &records); err != nil {
		t.Fatal(err)
	}
	if records[0]["catalog_ready"] != true || records[0]["ready"] != false {
		t.Fatalf("shop status incorrectly serialized: %s", data)
	}
	if _, exists := records[1]["catalog_ready"]; exists {
		t.Fatalf("depot incorrectly advertises shop catalog status: %s", data)
	}
}
