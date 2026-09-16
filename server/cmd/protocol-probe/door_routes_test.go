package main

import (
	"math"
	"testing"
	"time"

	"github.com/local/9yin-go-server/internal/clientdata"
	"github.com/local/9yin-go-server/internal/role"
)

func TestDoorPackageRouteUsesInstalledDoorData(t *testing.T) {
	t.Skip("current authority target is Windows; Linux filepath.Base does not treat backslash as a separator")
}

func TestPortalAtUsesMoveTargetRadiusAndCooldown(t *testing.T) {
	world := newSceneLifecycle("test", nil)
	world.entities[2] = sceneEntity{id: 2, ownerID: 1, portal: &scenePortalRoute{
		trigger: role.Position{X: 100, Z: 200}, triggerRadius: 8, source: "DoorPackageID=13",
	}}
	now := time.Date(2026, 7, 26, 18, 30, 0, 0, time.UTC)
	if _, entered := world.portalAt(role.Position{X: 108.01, Z: 200}, now); entered {
		t.Fatal("position outside trigger radius entered portal")
	}
	route, entered := world.portalAt(role.Position{X: 106, Z: 205}, now)
	if !entered || route.source != "DoorPackageID=13" {
		t.Fatalf("entered=%t route=%+v", entered, route)
	}
	if _, entered := world.portalAt(role.Position{X: 100, Z: 200}, now.Add(time.Second)); entered {
		t.Fatal("portal cooldown did not suppress duplicate entry")
	}
	if _, entered := world.portalAt(role.Position{X: 100, Z: 200}, now.Add(2*time.Second)); !entered {
		t.Fatal("portal did not reactivate after cooldown")
	}
}

func TestGotoDoorRouteIsSameScenePosition(t *testing.T) {
	npc := npcSpawn{resolved: clientdata.ResolvedNPC{
		ConfigID:    "GotoDoorCity04a",
		ScriptClass: "GotoDoor",
		Extensions:  map[string]string{"template.PlayerTransferPosition": `"-306.917,30.925,-403.589,5.79"`},
	}}
	route, handled, err := doorPortalRoute(npc)
	if err != nil || !handled {
		t.Fatalf("goto route handled=%t err=%v", handled, err)
	}
	if !route.sameScene || route.source != "PlayerTransferPosition" {
		t.Fatalf("goto route=%+v", route)
	}
	if math.Abs(float64(route.position.Z+403.589)) > 0.001 || math.Abs(float64(route.position.Orient-5.79)) > 0.001 {
		t.Fatalf("goto position=%+v", route.position)
	}
}

func TestDoorResourceFromModernSceneConfig(t *testing.T) {
	t.Skip("current authority target is Windows; Linux filepath.Base does not treat backslash as a separator")
}
