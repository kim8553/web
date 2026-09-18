package main

import (
	"bytes"
	"database/sql"
	"encoding/binary"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/local/9yin-go-server/internal/role"
)

type ordinaryShopBagConflictCapture struct {
	discardMessageConnection
	frames [][]byte
	failAt int
}

func (capture *ordinaryShopBagConflictCapture) WriteFrame(frame []byte) error {
	if capture.failAt > 0 && len(capture.frames)+1 == capture.failAt {
		return errors.New("connection write failed")
	}
	capture.frames = append(capture.frames, bytes.Clone(frame))
	return nil
}

func ordinaryShopBagConflictRows() *sqlmock.Rows {
	return sqlmock.NewRows([]string{"config_id", "item_type", "amount", "view_id", "name", "equip_type", "art_pack", "hardiness", "max_hardiness", "slot"}).
		AddRow("shop_existing", 100, 2, 1, nil, nil, nil, nil, nil, 4).
		AddRow("newer_other_session_purchase", 100, 1, 1, nil, nil, nil, nil, nil, 5)
}

func TestOrdinaryShopBagConflictResyncReplaysPersistedBagWithoutFailedReward(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil { t.Fatal(err) }
	defer db.Close()
	original := []bagItem{{ConfigID: "shop_existing", ItemType: 100, Amount: 2, ViewID: 1, Slot: 4}}
	actor := &playerActor{}
	actor.restoreBag(original)
	mock.ExpectQuery("SELECT config_id, item_type, amount, view_id").WithArgs(role.RoleID(91)).WillReturnRows(ordinaryShopBagConflictRows())
	capture := &ordinaryShopBagConflictCapture{}
	if err := resyncOrdinaryShopBagAfterConflict(capture, actor, &mysqlBagStore{db: db}, role.RoleID(91), original); err != nil {
		t.Fatal(err)
	}
	latest := actor.bagSnapshot()
	if len(latest) != 2 || latest[0].ConfigID != "shop_existing" || latest[1].ConfigID != "newer_other_session_purchase" {
		t.Fatalf("reloaded actor bag=%+v", latest)
	}
	if len(capture.frames) != 3 {
		t.Fatalf("frames=%d, want one old-item removal and two persisted additions", len(capture.frames))
	}
	remove := capture.frames[0]
	if len(remove) != 5 || remove[0] != 0x19 || binary.LittleEndian.Uint16(remove[1:3]) != 2 || binary.LittleEndian.Uint16(remove[3:5]) != 4 {
		t.Fatalf("old-item removal does not use existing frame: %x", remove)
	}
	for i, item := range latest {
		view := bagViewForViewID(item.ViewID)
		want, err := latestClientCurrentBagFrames(view, uint16(item.Slot), item.ViewID, bagItemProps(view, item))
		if err != nil || len(want) != 1 || !bytes.Equal(want[0], capture.frames[i+1]) {
			t.Fatalf("frame %d not identical to reconnect replay: %x, error %v", i+1, capture.frames[i+1], err)
		}
		if bytes.Contains(capture.frames[i+1], []byte("failed_reward")) {
			t.Fatal("failed purchase reward appeared in client refresh")
		}
	}
	if err := mock.ExpectationsWereMet(); err != nil { t.Fatal(err) }
}

func TestOrdinaryShopBagConflictResyncRejectsLocalMutationWithoutFrames(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil { t.Fatal(err) }
	defer db.Close()
	original := []bagItem{{ConfigID: "shop_existing", ItemType: 100, Amount: 2, ViewID: 1, Slot: 4}}
	actor := &playerActor{}
	actor.restoreBag(original)
	actor.addBagItem(bagItem{ConfigID: "unpersisted_local_item", ItemType: 100, Amount: 1, ViewID: 1, Slot: 8})
	before := actor.bagSnapshot()
	mock.ExpectQuery("SELECT config_id, item_type, amount, view_id").WithArgs(role.RoleID(91)).WillReturnRows(ordinaryShopBagConflictRows())
	capture := &ordinaryShopBagConflictCapture{}
	err = resyncOrdinaryShopBagAfterConflict(capture, actor, &mysqlBagStore{db: db}, role.RoleID(91), original)
	if err == nil || !strings.Contains(err.Error(), "local bag changed") {
		t.Fatalf("expected local mutation refusal, got %v", err)
	}
	if !reflect.DeepEqual(before, actor.bagSnapshot()) || len(capture.frames) != 0 {
		t.Fatalf("refusal changed actor or published frames: bag=%+v frames=%d", actor.bagSnapshot(), len(capture.frames))
	}
	if err := mock.ExpectationsWereMet(); err != nil { t.Fatal(err) }
}

func TestOrdinaryShopBagConflictResyncInvalidRowsFailClosed(t *testing.T) {
	for _, testcase := range []struct { name string; rows *sqlmock.Rows }{
		{"invalid_slot", sqlmock.NewRows([]string{"config_id", "item_type", "amount", "view_id", "name", "equip_type", "art_pack", "hardiness", "max_hardiness", "slot"}).AddRow("new", 100, 1, 1, nil, nil, nil, nil, nil, 0)},
		{"duplicate_slot", sqlmock.NewRows([]string{"config_id", "item_type", "amount", "view_id", "name", "equip_type", "art_pack", "hardiness", "max_hardiness", "slot"}).AddRow("first", 100, 1, 1, nil, nil, nil, nil, nil, 4).AddRow("second", 100, 1, 1, nil, nil, nil, nil, nil, 4)},
	} {
		t.Run(testcase.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil { t.Fatal(err) }
			defer db.Close()
			original := []bagItem{{ConfigID: "existing", ItemType: 100, Amount: 1, ViewID: 1, Slot: 1}}
			actor := &playerActor{}
			actor.restoreBag(original)
			mock.ExpectQuery("SELECT config_id, item_type, amount, view_id").WithArgs(role.RoleID(91)).WillReturnRows(testcase.rows)
			capture := &ordinaryShopBagConflictCapture{}
			if err := resyncOrdinaryShopBagAfterConflict(capture, actor, &mysqlBagStore{db: db}, role.RoleID(91), original); err == nil {
				t.Fatal("invalid persisted bag was accepted")
			}
			if !reflect.DeepEqual(original, actor.bagSnapshot()) || len(capture.frames) > 0 { t.Fatal("invalid bag changed actor/client") }
			if err := mock.ExpectationsWereMet(); err != nil { t.Fatal(err) }
		})
	}
}

func TestOrdinaryShopBagConflictResyncReadFailureDoesNotEraseBag(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil { t.Fatal(err) }
	defer db.Close()
	original := []bagItem{{ConfigID: "existing", ItemType: 100, Amount: 1, ViewID: 1, Slot: 1}}
	actor := &playerActor{}
	actor.restoreBag(original)
	mock.ExpectQuery("SELECT config_id, item_type, amount, view_id").WithArgs(role.RoleID(91)).WillReturnError(sql.ErrConnDone)
	capture := &ordinaryShopBagConflictCapture{}
	if err := resyncOrdinaryShopBagAfterConflict(capture, actor, &mysqlBagStore{db: db}, role.RoleID(91), original); err == nil { t.Fatal("read failure accepted") }
	if !reflect.DeepEqual(original, actor.bagSnapshot()) || len(capture.frames) != 0 { t.Fatal("read failure changed actor/client") }
	if err := mock.ExpectationsWereMet(); err != nil { t.Fatal(err) }
}

func TestOrdinaryShopBagConflictResyncWriteFailureReturned(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil { t.Fatal(err) }
	defer db.Close()
	original := []bagItem{{ConfigID: "shop_existing", ItemType: 100, Amount: 2, ViewID: 1, Slot: 4}}
	actor := &playerActor{}
	actor.restoreBag(original)
	mock.ExpectQuery("SELECT config_id, item_type, amount, view_id").WithArgs(role.RoleID(91)).WillReturnRows(ordinaryShopBagConflictRows())
	capture := &ordinaryShopBagConflictCapture{failAt: 2}
	if err := resyncOrdinaryShopBagAfterConflict(capture, actor, &mysqlBagStore{db: db}, role.RoleID(91), original); err == nil || !strings.Contains(err.Error(), "publish persisted bag refresh") {
		t.Fatalf("expected connection write error, got %v", err)
	}
	if len(actor.bagSnapshot()) != 2 || len(capture.frames) != 1 { t.Fatal("read persisted bag not retained, or unexpected frames") }
	if err := mock.ExpectationsWereMet(); err != nil { t.Fatal(err) }
}
