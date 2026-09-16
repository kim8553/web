package main

import (
	"bytes"
	"encoding/binary"
	"testing"
)

func TestStarterSkillViewUsesNativeSkillContainer(t *testing.T) {
	frames, err := starterSkillViewFrames()
	if err != nil {
		t.Fatal(err)
	}
	if got, want := len(frames), len(starterSkillViews)+1; got != want {
		t.Fatalf("View=40 frames=%d, want %d", got, want)
	}
	create := frames[0]
	if create[0] != 0x15 ||
		binary.LittleEndian.Uint16(create[1:]) != viewportSkill ||
		binary.LittleEndian.Uint16(create[3:]) != uint16(len(starterSkillViews)) {
		t.Fatalf("CreateView(40)=%x", create)
	}

	for index, skill := range starterSkillViews {
		frame := frames[index+1]
		if frame[0] != 0x18 ||
			binary.LittleEndian.Uint16(frame[1:]) != viewportSkill ||
			binary.LittleEndian.Uint16(frame[3:]) != uint16(index+1) {
			t.Fatalf("%s ViewAdd header=%x", skill.configID, frame)
		}
		for _, expected := range [][]byte{
			[]byte(skill.configID + "\x00"),
			binary.LittleEndian.AppendUint16(nil, 204), // StaticData
			binary.LittleEndian.AppendUint16(nil, 205), // ItemType
			{6, 0}, // Level
			binary.LittleEndian.AppendUint16(nil, 203),            // MaxLevel
			binary.LittleEndian.AppendUint16(nil, 176),            // CurFillValue
			binary.LittleEndian.AppendUint16(nil, 177),            // TotalFillValue
			binary.LittleEndian.AppendUint16(nil, 212),            // PauseTime
			append(binary.LittleEndian.AppendUint16(nil, 225), 1), // CanUse
		} {
			if !bytes.Contains(frame, expected) {
				t.Fatalf("%s ViewAdd lacks %x: %x", skill.configID, expected, frame)
			}
		}
	}
}

func TestYiHuaWuQueSkillMetadataMatchesInstalledClient(t *testing.T) {
	want := map[string]int32{"CS_yhwq_hsqs01": 12636, "CS_yhwq_hsqs02": 12637, "CS_yhwq_hsqs03": 12638, "CS_yhwq_hsqs04": 12639, "CS_yhwq_hsqs05": 12640, "CS_yhwq_hsqs06": 12641, "CS_yhwq_hsqs07": 12642}
	if len(yiHuaWuQueCombatSkills) != len(want) {
		t.Fatalf("YiHua family=%d want=%d", len(yiHuaWuQueCombatSkills), len(want))
	}
	for id, static := range want {
		d, ok := yiHuaWuQueCombatSkills[id]
		if !ok || d.staticData != static || d.level != 1 {
			t.Fatalf("%s=%+v ok=%t", id, d, ok)
		}
	}
	if len(starterSkillViews) != 7 {
		t.Fatalf("current starter skills=%d want=7", len(starterSkillViews))
	}
}

func TestHeartBuddhaPalmSkillMetadataMatchesInstalledClient(t *testing.T) {
	want := map[string]int32{"CS_jh_xfz01": 13189, "CS_jh_xfz02": 13190, "CS_jh_xfz02_hide": 13204, "CS_jh_xfz03": 13205, "CS_jh_xfz04": 13206, "CS_jh_xfz05": 13207, "CS_jh_xfz06": 13208, "CS_jh_xfz07": 13203}
	if len(heartBuddhaPalmCombatSkills) != len(want) {
		t.Fatalf("HeartBuddha family=%d want=%d", len(heartBuddhaPalmCombatSkills), len(want))
	}
	for id, static := range want {
		d, ok := heartBuddhaPalmCombatSkills[id]
		if !ok || d.staticData != static || d.level != 1 {
			t.Fatalf("%s=%+v ok=%t", id, d, ok)
		}
	}
}

func TestTaiJiQuanGuPuSkillMetadataMatchesInstalledClient(t *testing.T) {
	want := map[string]int32{"CS_wd_tjq01": 4057, "CS_wd_tjq02": 4058, "CS_wd_tjq03": 4238, "CS_wd_tjq04": 4045, "CS_wd_tjq05": 5451, "CS_wd_tjq06": 5452, "CS_wd_tjq07": 5453, "CS_wd_tjq08": 2107}
	if len(taiJiQuanGuPuCombatSkills) != len(want) {
		t.Fatalf("TaiJi family=%d want=%d", len(taiJiQuanGuPuCombatSkills), len(want))
	}
	for id, static := range want {
		d, ok := taiJiQuanGuPuCombatSkills[id]
		if !ok || d.staticData != static || d.level != 1 {
			t.Fatalf("%s=%+v ok=%t", id, d, ok)
		}
	}
	for _, starter := range starterSkillViews {
		if _, ok := want[starter.configID]; ok {
			t.Fatalf("current starter list unexpectedly contains TaiJi skill %+v", starter)
		}
	}
}
