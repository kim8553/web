package main

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestLatestClientTaskAcceptedSevenColumnPrefixTypes(t *testing.T) {
	if int(recordTaskAccepted) >= len(qingGongRecordSchemas) {
		t.Fatalf("Task_Accepted record index %d missing", recordTaskAccepted)
	}
	got := qingGongRecordSchemas[recordTaskAccepted].colTypes
	want := []byte{3, 3, 3, 3, 7, 7, 3}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Task_Accepted prefix types=%v want=%v", got, want)
	}
	if _, err := serverRecordAddCells(playerObjectID, playerOwnerID, recordTaskAccepted, []recordCell{
		recordInt(10), recordInt(1), recordInt(0), recordString("17"), recordString("npc"), recordString("title"), recordInt(0),
	}); err == nil {
		t.Fatal("Task_Accepted column 3 accepted string; current client requires int32")
	}
	if _, err := serverRecordAddCells(playerObjectID, playerOwnerID, recordTaskAccepted, []recordCell{
		recordInt(10), recordInt(1), recordInt(0), recordInt(17), recordString("npc"), recordString("title"), recordInt(0),
	}); err != nil {
		t.Fatalf("current Task_Accepted prefix rejected: %v", err)
	}
}

func TestQuestCatalogLoadsNumericSubmitSceneIDOnlyWhenColumnExists(t *testing.T) {
	root := t.TempDir()
	taskDir := filepath.Join(root, "task", "task")
	if err := os.MkdirAll(taskDir, 0o755); err != nil {
		t.Fatal(err)
	}
	withScene := "meta\nID\tLine\tSubmitSceneID\tAcceptNpc\tSubmitNpc\tTitleId\tContextId\tCompleteDialogId\tTaskTargetId\tAcceptDialogId\n100\t7\t17\tnpc_a\tnpc_b\ttitle\tcontext\tcomplete\ttarget\tmenu\n"
	withoutScene := "meta\nID\tLine\tAcceptNpc\tSubmitNpc\tTitleId\tContextId\tCompleteDialogId\tTaskTargetId\tAcceptDialogId\n101\t8\tnpc_c\tnpc_d\ttitle2\tcontext2\tcomplete2\ttarget2\tmenu2\n"
	if err := os.WriteFile(filepath.Join(taskDir, "task_a.txt"), []byte(withScene), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(taskDir, "task_b.txt"), []byte(withoutScene), 0o644); err != nil {
		t.Fatal(err)
	}
	catalog, err := loadQuestCatalogFromTables(root)
	if err != nil {
		t.Fatal(err)
	}
	if got := catalog[100]; got == nil || got.SubmitSceneID != 17 {
		t.Fatalf("task 100 SubmitSceneID=%v", got)
	}
	if got := catalog[101]; got == nil || got.SubmitSceneID != 0 {
		t.Fatalf("task 101 missing SubmitSceneID should remain 0, got=%v", got)
	}
}
