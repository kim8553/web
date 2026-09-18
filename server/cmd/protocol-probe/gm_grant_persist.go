package main

import (
	"errors"
	"fmt"

	"github.com/local/9yin-go-server/internal/role"
)

// grantBagItemDurably retains the existing item grant and bag serializer but
// never leaves an unsuccessful save looking like a completed GM grant.
// The caller must not send inventory frames or a success message on error.
func grantBagItemDurably(player *playerActor, store bagStoreIface, roleID role.RoleID, item bagItem) error {
	if player == nil || store == nil || roleID == 0 {
		return errors.New("GM grant: missing player, bag store, or role")
	}
	before := player.bagSnapshot()
	player.addBagItem(item)
	if err := store.Save(roleID, player.bagSnapshot()); err != nil {
		player.restoreBag(before)
		return fmt.Errorf("persist GM grant %s: %w", item.ConfigID, err)
	}
	return nil
}
