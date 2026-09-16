package exchangecommit

import (
	"errors"
	"reflect"
	"testing"
)

func TestPersistFirstSlicePersistsBeforePublishAndDoesNotAlias(t *testing.T) {
	current := []int{1, 2}
	original := append([]int(nil), current...)
	var persisted []int
	result, err := PersistFirstSlice(current, func(snapshot []int) ([]int, error) {
		snapshot[0] = 9
		return append(snapshot, 3), nil
	}, func(staged []int) error {
		if !reflect.DeepEqual(current, original) {
			t.Fatalf("current changed before persist: got=%v want=%v", current, original)
		}
		persisted = staged
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(current, original) {
		t.Fatalf("current mutated: got=%v want=%v", current, original)
	}
	want := []int{9, 2, 3}
	if !reflect.DeepEqual(result, want) || !reflect.DeepEqual(persisted, want) {
		t.Fatalf("result=%v persisted=%v want=%v", result, persisted, want)
	}
	result[0] = 77
	if persisted[0] == 77 || current[0] == 77 {
		t.Fatal("returned state aliases persisted or current state")
	}
	persisted[1] = 88
	if result[1] == 88 || current[1] == 88 {
		t.Fatal("persisted state aliases returned or current state")
	}
}

func TestPersistFirstSlicePersistenceFailureDoesNotMutateCurrent(t *testing.T) {
	current := []int{1, 2}
	original := append([]int(nil), current...)
	result, err := PersistFirstSlice(current, func(snapshot []int) ([]int, error) {
		snapshot[0] = 9
		return snapshot, nil
	}, func([]int) error { return errors.New("store down") })
	if err == nil {
		t.Fatal("persistence failure must be returned")
	}
	if result != nil {
		t.Fatalf("result=%v want=nil", result)
	}
	if !reflect.DeepEqual(current, original) {
		t.Fatalf("current mutated after persistence failure: got=%v want=%v", current, original)
	}
}

func TestPersistFirstSliceStageFailureSkipsPersistence(t *testing.T) {
	calls := 0
	_, err := PersistFirstSlice([]int{1}, func([]int) ([]int, error) {
		return nil, errors.New("stale")
	}, func([]int) error {
		calls++
		return nil
	})
	if err == nil {
		t.Fatal("stage failure must be returned")
	}
	if calls != 0 {
		t.Fatalf("persist calls=%d want=0", calls)
	}
}

func TestPersistFirstSliceRejectsNilCallbacks(t *testing.T) {
	if _, err := PersistFirstSlice[int](nil, nil, func([]int) error { return nil }); err == nil {
		t.Fatal("nil stage must fail")
	}
	if _, err := PersistFirstSlice[int](nil, func(v []int) ([]int, error) { return v, nil }, nil); err == nil {
		t.Fatal("nil persist must fail")
	}
}
