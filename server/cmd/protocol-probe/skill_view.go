package main

import "fmt"

const viewportSkill uint16 = 40

type learnedSkillView struct {
	configID   string
	staticData int32
	itemType   int32
	level      int32
	maxLevel   int32
	fill       int32
	total      int32
	pauseTime  float32
}

var starterSkillViews = []learnedSkillView{{configID: "zs_default_01", staticData: 22, itemType: 1000, level: 1, maxLevel: 1}, {configID: "CS_light_rad_81", staticData: 5266, itemType: 1000, level: 1, maxLevel: 1}, {configID: "CS_jh_cqgf01", staticData: 4401, itemType: 1001, level: 1, maxLevel: 3, pauseTime: 1000}, {configID: "CS_jh_cqgf04", staticData: 4404, itemType: 1000, level: 1, maxLevel: 3}, {configID: "CS_jh_cqgf06", staticData: 4260, itemType: 1000, level: 1, maxLevel: 3}, {configID: "taolu_zhenfa_wzyx", staticData: 11356, itemType: 1000, level: 1, maxLevel: 1}, {configID: "taolu_zhenfa_cfxz", staticData: 11357, itemType: 1000, level: 1, maxLevel: 1}}

func starterSkillViewFrames() ([][]byte, error) {
	frames := [][]byte{serverCreateView(serverViewSpec{ID: viewportSkill, Capacity: uint16(len(starterSkillViews))})}
	for index, skill := range starterSkillViews {
		frame, err := serverViewAdd(viewportSkill, uint16(index+1), []serverViewProperty{viewString(7, skill.configID), viewInt(propStaticData, skill.staticData), viewInt(propItemType, skill.itemType), viewInt(6, skill.level), viewInt(propMaxLevel, skill.maxLevel), viewInt(propCurFillValue, skill.fill), viewInt(propTotalFillValue, skill.total), viewFloat(propPauseTime, skill.pauseTime), viewByte(propSkillCanUse, 1)})
		if err != nil {
			return nil, fmt.Errorf("encode SkillContainer %s: %w", skill.configID, err)
		}
		frames = append(frames, frame)
	}
	return frames, nil
}
