package main

import "fmt"

const (
	serverSwitchControlMessage int32 = 414
	skillActionInputSwitch     int32 = 1223
)

func skillActionSwitchFrame() ([]byte, error) {
	return serverModernCustomIntMessage(414, customString("reset"), customInt(1223), customInt(1))
}
func enableSkillActionSwitch(link sceneMessageConnection) error {
	frame, err := skillActionSwitchFrame()
	if err != nil {
		return fmt.Errorf("encode skill-action switch: %w", err)
	}
	if err := link.WriteFrame(frame); err != nil {
		return fmt.Errorf("write skill-action switch: %w", err)
	}
	return nil
}
