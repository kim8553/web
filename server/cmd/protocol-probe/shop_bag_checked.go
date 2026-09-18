package main

import (
	"github.com/local/9yin-go-server/internal/role"
	"github.com/local/9yin-go-server/internal/shopbuyatomic"
)

// persistBagMutationChecked does not replace the original bag system. It
// adds a pre-change snapshot comparison for migrated MySQL roles; legacy
// JSON storage retains its existing single-file save behavior.
func persistBagMutationChecked(store bagStoreIface, roleID role.RoleID, before, after []bagItem) error {
	if store == nil {
		return nil
	}
	if mysqlStore, ok := store.(*mysqlBagStore); ok && mysqlStore != nil && mysqlStore.db != nil {
		return shopbuyatomic.SaveBagChecked(mysqlStore.db, uint64(roleID), ordinaryShopBagRows(after), ordinaryShopBagRows(before))
	}
	return store.Save(roleID, after)
}
