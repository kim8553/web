package main

import (
	"fmt"

	"github.com/local/9yin-go-server/internal/shopbuypublish"
)

// buildShopPurchaseClientFrames freezes publication to the staged durable/live
// result instead of re-reading a possibly newer live currency value after the
// player mutex has been released. It reuses the existing current currency and
// latest-client bag encoders; it does not define or infer a shop-buy selector.
func buildShopPurchaseClientFrames(staged stagedShopPurchase) ([][]byte, error) {
	if staged.RewardSlot <= 0 || staged.RewardSlot > 0xffff {
		return nil, fmt.Errorf("shop buy publish: reward slot %d outside uint16 range", staged.RewardSlot)
	}
	if staged.Reward.Slot != staged.RewardSlot {
		return nil, fmt.Errorf("shop buy publish: reward slot mismatch item=%d staged=%d", staged.Reward.Slot, staged.RewardSlot)
	}
	if staged.RewardView == 0 || bagViewForViewID(staged.Reward.ViewID) != staged.RewardView {
		return nil, fmt.Errorf("shop buy publish: reward view mismatch view=%d item_view_id=%d", staged.RewardView, staged.Reward.ViewID)
	}
	if staged.Reward.Amount <= 0 {
		return nil, fmt.Errorf("shop buy publish: non-positive reward amount %d", staged.Reward.Amount)
	}

	// Use a private zero-value actor only as an input to the already-recovered
	// current currency encoder. No live player state is read or modified here.
	shadow := &playerActor{
		silver:       staged.CurrencyAfter.Silver,
		gold:         staged.CurrencyAfter.Gold,
		silverCard:   staged.CurrencyAfter.SilverCard,
		silverTicket: staged.CurrencyAfter.SilverTicket,
	}
	currencyFrame, err := shadow.currenciesUpdate()
	if err != nil {
		return nil, fmt.Errorf("shop buy publish: encode committed currency: %w", err)
	}
	bagFrames, err := latestClientCurrentBagFrames(
		staged.RewardView,
		uint16(staged.RewardSlot),
		staged.Reward.ViewID,
		bagItemProps(staged.RewardView, staged.Reward),
	)
	if err != nil {
		return nil, fmt.Errorf("shop buy publish: encode committed reward: %w", err)
	}
	frames := make([][]byte, 0, 1+len(bagFrames))
	frames = append(frames, currencyFrame)
	frames = append(frames, bagFrames...)
	return frames, nil
}

// publishShopPurchaseCommittedState is dormant until exact-current ordinary
// shop-buy wire authority is proven. It must only be called after the Stage19
// durable/live commit succeeds. Publication failure does not roll back DB or
// memory; shopbuypublish returns ErrResyncRequired so the future caller can
// resync/reconnect the client.
func publishShopPurchaseCommittedState(link sceneMessageConnection, staged stagedShopPurchase) error {
	if link == nil {
		return fmt.Errorf("shop buy publish: connection unavailable")
	}
	frames, err := buildShopPurchaseClientFrames(staged)
	if err != nil {
		return err
	}
	_, err = shopbuypublish.Publish(frames, link.WriteFrame)
	if err != nil {
		return fmt.Errorf("shop buy publish: %w", err)
	}
	return nil
}
