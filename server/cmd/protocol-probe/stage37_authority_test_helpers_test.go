package main

import "fmt"

func learnAuthoritySkill(player *playerActor, id string, level int32) {
	if level <= 0 {
		level = 1
	}
	player.mu.Lock()
	player.learnedSkills[id] = level
	player.mu.Unlock()
}

func learnAuthorityNeiGongAtLevel(player *playerActor, id string, level int32) error {
	if player == nil {
		return fmt.Errorf("nil player")
	}
	player.mu.Lock()
	defer player.mu.Unlock()
	player.progress.setFullUnlockBooks(modernNeiGongBookDefs())
	book := player.progress.book(id)
	if book == nil {
		return fmt.Errorf("current neigong %q is absent", id)
	}
	if level <= 0 || level > book.maxLevel {
		return fmt.Errorf("current neigong %q level %d outside 1..%d", id, level, book.maxLevel)
	}
	book.level = level
	book.fill = 0
	return book.applyModernEffect()
}
