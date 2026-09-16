package main

import (
	"bytes"
	"encoding/binary"
	"log"
	"math"
	"strings"
	"testing"
)

func observerAppendInt32(dst []byte, value int32) []byte {
	dst = append(dst, 2)
	return binary.LittleEndian.AppendUint32(dst, uint32(value))
}

func observerAppendFloat32(dst []byte, value float32) []byte {
	dst = append(dst, 4)
	return binary.LittleEndian.AppendUint32(dst, math.Float32bits(value))
}

func observerAppendString(dst []byte, value string) []byte {
	dst = append(dst, 6)
	dst = binary.LittleEndian.AppendUint32(dst, uint32(len(value)+1))
	dst = append(dst, value...)
	return append(dst, 0)
}

func observerAppendObject(dst []byte, objectID, ownerID uint32) []byte {
	dst = append(dst, 8)
	dst = binary.LittleEndian.AppendUint32(dst, objectID)
	return binary.LittleEndian.AppendUint32(dst, ownerID)
}

func observerBuildCustom(opcode byte, values int, payload []byte) []byte {
	frame := []byte{opcode}
	frame = binary.LittleEndian.AppendUint16(frame, uint16(values))
	return append(frame, payload...)
}

func observerBuild211() []byte {
	var payload []byte
	payload = observerAppendInt32(payload, 211)
	payload = observerAppendString(payload, "CS_stage27_test")
	payload = observerAppendFloat32(payload, 1.25)
	payload = observerAppendFloat32(payload, -2.5)
	payload = observerAppendFloat32(payload, 3.75)
	payload = observerAppendFloat32(payload, 0.5)
	payload = observerAppendFloat32(payload, 10)
	payload = observerAppendFloat32(payload, 20)
	payload = observerAppendFloat32(payload, 30)
	payload = observerAppendString(payload, "custom_item_x")
	payload = observerAppendObject(payload, 0x11223344, 0x55667788)
	payload = observerAppendFloat32(payload, 40)
	payload = observerAppendFloat32(payload, 50)
	payload = observerAppendFloat32(payload, 60)
	return observerBuildCustom(0x1E, 14, payload)
}

func observerBuild212() []byte {
	var payload []byte
	payload = observerAppendInt32(payload, 212)
	payload = observerAppendFloat32(payload, 1)
	payload = observerAppendFloat32(payload, 2)
	payload = observerAppendFloat32(payload, 3)
	payload = observerAppendFloat32(payload, 0.25)
	payload = observerAppendInt32(payload, 1)
	return observerBuildCustom(0x27, 6, payload)
}

func captureObserverLog(t *testing.T, fn func()) string {
	t.Helper()
	oldWriter := log.Writer()
	oldFlags := log.Flags()
	oldPrefix := log.Prefix()
	var buffer bytes.Buffer
	log.SetOutput(&buffer)
	log.SetFlags(0)
	log.SetPrefix("")
	defer func() {
		log.SetOutput(oldWriter)
		log.SetFlags(oldFlags)
		log.SetPrefix(oldPrefix)
	}()
	fn()
	return buffer.String()
}

func TestSourceObserver211Contract(t *testing.T) {
	message, err := parseClientCustomMessage(observerBuild211())
	if err != nil {
		t.Fatal(err)
	}
	observer := &sourceObserver{enabled: true, session: 7, remote: "127.0.0.1:test"}
	got := captureObserverLog(t, func() { observer.observeCustom(message, 3) })
	for _, want := range []string{
		"OBS_CUSTOM_DECODE",
		"rx_seq=3 session=7",
		"msg_id=211",
		"index=9 wire_type=6 string=\"custom_item_x\"",
		"index=10 wire_type=8 raw8=4433221188776655 object_id=287454020 owner_id=1432778632",
		"OBS_C2S_211",
		"v9_aux_string=\"custom_item_x\"",
		"observe_only=true",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %q in log:\n%s", want, got)
		}
	}
}

func TestSourceObserver212ObserveOnly(t *testing.T) {
	message, err := parseClientCustomMessage(observerBuild212())
	if err != nil {
		t.Fatal(err)
	}
	observer := &sourceObserver{enabled: true, session: 9}
	got := captureObserverLog(t, func() { observer.observeCustom(message, 4) })
	for _, want := range []string{"OBS_C2S_212", "rx_seq=4", "flag=1", "flag_known=true", "observe_only=true"} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %q in log:\n%s", want, got)
		}
	}
	if strings.Contains(got, "SKILL_SESSION_BEGIN") {
		t.Fatalf("212 unexpectedly changed gameplay/session state:\n%s", got)
	}
}

func TestSourceObserverDisabledIsSilent(t *testing.T) {
	message, err := parseClientCustomMessage(observerBuild212())
	if err != nil {
		t.Fatal(err)
	}
	observer := &sourceObserver{}
	got := captureObserverLog(t, func() {
		observer.observeRX([]byte{1, 2, 3})
		observer.observeTX([]byte{4, 5, 6})
		observer.observeCustom(message, 1)
	})
	if got != "" {
		t.Fatalf("disabled observer emitted log: %q", got)
	}
}
