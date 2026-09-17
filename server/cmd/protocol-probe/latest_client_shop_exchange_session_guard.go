package main

// currentShopExchangeSession records only server-observed NPC shop-service
// selection. It is NOT a grant/debit authorization and is deliberately not a
// client-defined session token or an assumed upstream protocol structure.
type currentShopExchangeSession struct {
	ShopID      string
	NPCObjectID uint32
	NPCOwnerID  uint32
	NPCConfigID string
	SceneEpoch  uint64
}

// allows proves that an exchange refers to the last shop successfully opened
// via an NPC service, in the same scene generation, and that the NPC still
// advertises that exact shop. The caller must resolve NPC identity/services
// under sceneLifecycle.mu. No unproven distance or expiry threshold is added.
func (session currentShopExchangeSession) allows(shopID string, epoch uint64, npcID, ownerID uint32, configID string, advertised bool) bool {
	return shopID != "" && advertised &&
		session.ShopID == shopID &&
		session.NPCObjectID != 0 && session.NPCObjectID == npcID &&
		session.NPCOwnerID == ownerID &&
		session.NPCConfigID != "" && session.NPCConfigID == configID &&
		session.SceneEpoch == epoch
}
