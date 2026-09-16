package main

import (
	"fmt"
	"log"
	"time"
)

const clientCustomActiveParry int32 = 218

type activeParryRequest struct {
	Subtype  int32
	Argument clientCustomValue
	At       time.Time
}

func handleActiveParryCustom(player *playerActor, custom clientCustomMessage, remote string) (bool, error) {
	if len(custom.Values) == 0 || custom.Values[0].Type != 2 || custom.Values[0].Int32 != 0xDA {
		return false, nil
	}
	if player == nil {
		return true, fmt.Errorf("active parry before scene player exists")
	}
	if len(custom.Values) < 3 {
		return true, fmt.Errorf("active parry requires subtype and argument")
	}
	subtype, ok := custom.Values[1].exactInt32()
	if !ok {
		return true, fmt.Errorf("active parry subtype must be an integral TVarList value, got %s", custom.Values[1])
	}
	log.Printf("%s: client set parry mode subtype=%d argument=%s (ignored; settings command)", remote, subtype, custom.Values[2])
	return true, nil
}
