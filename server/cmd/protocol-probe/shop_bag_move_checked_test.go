package main

import (
	"encoding/binary"
	"reflect"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/local/9yin-go-server/internal/role"
)

// Use the existing test connection contract; capture the original server
// VIEW_REMOVE/VIEW_ADD frames without inventing a client packet format.
type bagMoveFrameCapture struct {
	discardMessageConnection
	frames [][]byte
}

func (capture *bagMoveFrameCapture) WriteFrame(frame []byte) error {
	capture.frames = append(capture.frames, append([]byte(nil), frame...))
	return nil
}

func bagMoveStartingItems() []bagItem {
	return []bagItem{
		{Slot: 5, ConfigID: "move_source", ItemType: 100, Amount: 1, ViewID: 1},
		{Slot: 9, ConfigID: "move_occupant", ItemType: 100, Amount: 2, ViewID: 2},
	}
}

func TestShopAdjacentBagMoveCrossViewSwapKeepsSourceSlot(t *testing.T) {
	player := &playerActor{}
	player.restoreBag(bagMoveStartingItems())
	link := &bagMoveFrameCapture{}
	handled, err := applyBagMove(link, player, nil, role.RoleID(71), 2, 5, 121, 9, "cross-view-test")
	if err != nil || !handled {
		t.Fatalf("cross-view MOVEITEM handled=%t err=%v", handled, err)
	}
	moved, ok := player.peekBagItem(121, 9)
	if !ok || moved.ConfigID != "move_source" || moved.Slot != 9 || moved.ViewID != 2 {
		t.Fatalf("source not moved to destination: %+v present=%t", moved, ok)
	}
	displaced, ok := player.peekBagItem(2, 5)
	if !ok || displaced.ConfigID != "move_occupant" || displaced.Slot != 5 || displaced.ViewID != 1 {
		t.Fatalf("displaced item not stored at original source slot: %+v present=%t", displaced, ok)
	}
	if len(link.frames) != 3 {
		t.Fatalf("frame count=%d, want remove and two adds", len(link.frames))
	}
	want := [][3]uint16{{0x19, 2, 5}, {0x18, 2, 5}, {0x18, 121, 9}}
	for i, frame := range link.frames {
		if len(frame) < 5 || uint16(frame[0]) != want[i][0] || binary.LittleEndian.Uint16(frame[1:3]) != want[i][1] || binary.LittleEndian.Uint16(frame[3:5]) != want[i][2] {
			t.Fatalf("frame[%d] header=%x, want command=%d view=%d slot=%d", i, frame, want[i][0], want[i][1], want[i][2])
		}
	}
}

// A concurrent NPC purchase may have changed the database since the actor
// loaded its bag. Reject the move before publishing any client frame; restore
// the complete actor bag rather than leaving only the source removed.
func TestShopAdjacentBagMoveRejectsStaleBagBeforePublishing(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	starting := bagMoveStartingItems()
	player := &playerActor{}
	player.restoreBag(starting)
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT role_id FROM roles").WithArgs(uint64(71)).WillReturnRows(
		sqlmock.NewRows([]string{"role_id"}).AddRow(uint64(71)))
	mock.ExpectQuery("SELECT slot, config_id, item_type, amount, view_id").WithArgs(uint64(71)).WillReturnRows(
		sqlmock.NewRows([]string{"slot", "config_id", "item_type", "amount", "view_id", "name", "equip_type", "art_pack", "hardiness", "max_hardiness"}).
			AddRow(5, "move_source", 100, 1, 1, nil, nil, nil, nil, nil).
			AddRow(9, "move_occupant", 100, 2, 2, nil, nil, nil, nil, nil).
			AddRow(10, "new_purchase", 100, 1, 1, nil, nil, nil, nil, nil))
	mock.ExpectRollback()
	// A nil connection makes premature frame publication fail immediately.
	handled, err := applyBagMove(nil, player, &mysqlBagStore{db: db}, role.RoleID(71), 2, 5, 121, 9, "stale-move-test")
	if err != nil || !handled {
		t.Fatalf("stale MOVEITEM handled=%t err=%v", handled, err)
	}
	if got := player.bagSnapshot(); !reflect.DeepEqual(got, starting) {
		t.Fatalf("stale MOVEITEM changed actor bag: got=%+v want=%+v", got, starting)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
