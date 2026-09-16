package main

import (
	"encoding/binary"
	"github.com/local/9yin-go-server/internal/transport"
	"log"
	"math"
	"os"
	"strings"
	"sync/atomic"
)

type sourceObserver struct {
	enabled  bool
	session  uint64
	remote   string
	eventSeq atomic.Uint64
	rxSeq    atomic.Uint64
	txSeq    atomic.Uint64
}

func newSourceObserver(session uint64, remote string) *sourceObserver {
	value := strings.TrimSpace(strings.ToLower(os.Getenv("JIUYIN_SOURCE_OBSERVER")))
	enabled := value == "1" || value == "true" || value == "yes" || value == "on"
	return &sourceObserver{enabled: enabled, session: session, remote: remote}
}
func (o *sourceObserver) nextEvent() uint64 {
	if o == nil {
		return 0
	}
	return o.eventSeq.Add(1)
}
func (o *sourceObserver) observeRX(frame []byte) uint64 {
	if o == nil || !o.enabled {
		return 0
	}
	rxSeq := o.rxSeq.Add(1)
	eventSeq := o.nextEvent()
	log.Printf("OBS_RX_RAW event_seq=%d rx_seq=%d session=%d remote=%q len=%d hex=%X", eventSeq, rxSeq, o.session, o.remote, len(frame), frame)
	return rxSeq
}
func (o *sourceObserver) observeTX(frame []byte) {
	if o == nil || !o.enabled {
		return
	}
	txSeq := o.txSeq.Add(1)
	eventSeq := o.nextEvent()
	log.Printf("OBS_TX_RAW event_seq=%d tx_seq=%d session=%d remote=%q len=%d hex=%X", eventSeq, txSeq, o.session, o.remote, len(frame), frame)
}
func (o *sourceObserver) observeCustom(message clientCustomMessage, rxSeq uint64) {
	if o == nil || !o.enabled {
		return
	}
	msgID := int32(-1)
	if len(message.Values) > 0 && message.Values[0].Type == 2 {
		msgID = message.Values[0].Int32
	}
	log.Printf("OBS_CUSTOM_DECODE event_seq=%d rx_seq=%d session=%d outer_opcode=0x%02X msg_id=%d value_count=%d", o.nextEvent(), rxSeq, o.session, message.Opcode, msgID, len(message.Values))
	for index, value := range message.Values {
		eventSeq := o.nextEvent()
		switch value.Type {
		case 2:
			log.Printf("OBS_CUSTOM_VALUE event_seq=%d rx_seq=%d session=%d msg_id=%d index=%d wire_type=2 int32=%d raw_u32=0x%08X", eventSeq, rxSeq, o.session, msgID, index, value.Int32, uint32(value.Int32))
		case 3:
			log.Printf("OBS_CUSTOM_VALUE event_seq=%d rx_seq=%d session=%d msg_id=%d index=%d wire_type=3 int64=%d raw_u64=0x%016X", eventSeq, rxSeq, o.session, msgID, index, value.Int64, uint64(value.Int64))
		case 4:
			bits := math.Float32bits(value.Float32)
			log.Printf("OBS_CUSTOM_VALUE event_seq=%d rx_seq=%d session=%d msg_id=%d index=%d wire_type=4 float32=%g raw_u32=0x%08X", eventSeq, rxSeq, o.session, msgID, index, value.Float32, bits)
		case 5:
			bits := math.Float64bits(value.Float64)
			log.Printf("OBS_CUSTOM_VALUE event_seq=%d rx_seq=%d session=%d msg_id=%d index=%d wire_type=5 float64=%g raw_u64=0x%016X", eventSeq, rxSeq, o.session, msgID, index, value.Float64, bits)
		case 6:
			log.Printf("OBS_CUSTOM_VALUE event_seq=%d rx_seq=%d session=%d msg_id=%d index=%d wire_type=6 string=%q", eventSeq, rxSeq, o.session, msgID, index, value.Text)
		case 7:
			log.Printf("OBS_CUSTOM_VALUE event_seq=%d rx_seq=%d session=%d msg_id=%d index=%d wire_type=7 widestr=%q", eventSeq, rxSeq, o.session, msgID, index, value.Text)
		case 8:
			objectID := binary.LittleEndian.Uint32(value.Raw[:4])
			ownerID := binary.LittleEndian.Uint32(value.Raw[4:])
			log.Printf("OBS_CUSTOM_VALUE event_seq=%d rx_seq=%d session=%d msg_id=%d index=%d wire_type=8 raw8=%X object_id=%d owner_id=%d", eventSeq, rxSeq, o.session, msgID, index, value.Raw, objectID, ownerID)
		default:
			log.Printf("OBS_CUSTOM_VALUE event_seq=%d rx_seq=%d session=%d msg_id=%d index=%d wire_type=%d unsupported=true", eventSeq, rxSeq, o.session, msgID, index, value.Type)
		}
	}
	if msgID == 211 {
		skillID := ""
		if len(message.Values) > 1 && message.Values[1].Type == 6 {
			skillID = message.Values[1].Text
		}
		v9 := ""
		if len(message.Values) > 9 && message.Values[9].Type == 6 {
			v9 = message.Values[9].Text
		}
		log.Printf("OBS_C2S_211 event_seq=%d rx_seq=%d session=%d skill=%q value_count=%d v9_aux_string=%q observe_only=true", o.nextEvent(), rxSeq, o.session, skillID, len(message.Values), v9)
	} else if msgID == 212 {
		flag := int32(0)
		flagKnown := false
		if len(message.Values) > 5 && message.Values[5].Type == 2 {
			flag = message.Values[5].Int32
			flagKnown = true
		}
		log.Printf("OBS_C2S_212 event_seq=%d rx_seq=%d session=%d value_count=%d flag=%d flag_known=%t observe_only=true", o.nextEvent(), rxSeq, o.session, len(message.Values), flag, flagKnown)
	}
}

type observedSceneConnection struct {
	inner    *transport.Connection
	observer *sourceObserver
}

func (c *observedSceneConnection) ReadFrame() ([]byte, error) {
	return c.inner.ReadFrame()
}
func (c *observedSceneConnection) WriteFrame(frame []byte) error {
	c.observer.observeTX(frame)
	return c.inner.WriteFrame(frame)
}
func (c *observedSceneConnection) Close() error {
	return c.inner.Close()
}
