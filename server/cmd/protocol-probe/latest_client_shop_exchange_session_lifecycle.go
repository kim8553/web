package main

import worldcore "github.com/local/9yin-go-server/internal/world"

// currentShopExchangeSessionAllows is only a read-only preflight. The scene
// mutex protects both the recorded selection and the live NPC/service lookup.
// Client-selected ShopID alone is never evidence that the shop was opened.
func (s *sceneLifecycle) currentShopExchangeSessionAllows(shopID string) bool {
	if s == nil {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed || shopID == "" {
		return false
	}
	session := s.activeShop
	if session.NPCObjectID == 0 {
		return false
	}
	npc, ok := s.entities[worldcore.EntityID(session.NPCObjectID)]
	if !ok {
		return false
	}
	service, advertised := findNPCService(npc.services, markShop)
	return session.allows(shopID, s.epoch, npc.id, npc.ownerID, npc.configID,
		advertised && service.value == shopID)
}
