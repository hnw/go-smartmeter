package smartmeter

import (
	"encoding/hex"
	"reflect"
	"testing"
)

func TestParseFrame(t *testing.T) {
	decoded, _ := hex.DecodeString("1081000102880105FF017201E80400140064")
	frame, err := ParseFrame(decoded)
	if err != nil {
		t.Errorf("Error occurred: %v", err)
	}
	if len(frame.Properties) != 1 {
		t.Errorf("OPC value differ: %v != 1", len(frame.Properties))
	}
	expectedEPC := LvSmartElectricEnergyMeter_InstantaneousCurrent
	if !reflect.DeepEqual(frame.Properties[0].EPC, expectedEPC) {
		t.Errorf("EPC value differ:%v %v != %v", frame, frame.Properties[0].EPC, expectedEPC)
	}
	expectedEDT := []byte{0x0, 0x14, 0x0, 0x64}
	if !reflect.DeepEqual(frame.Properties[0].EDT, expectedEDT) {
		t.Errorf("EDT value differ: %v != %v", frame.Properties[0].EDT, expectedEDT)
	}
}

func TestNewFrame(t *testing.T) {
	frame := NewFrame(LvSmartElectricEnergyMeter, Get, []*Property{
		NewProperty(LvSmartElectricEnergyMeter_InstantaneousElectricPower, nil),
		NewProperty(LvSmartElectricEnergyMeter_InstantaneousCurrent, nil),
	})
	frame.TID = 0
	s := frame.Build()
	expected, _ := hex.DecodeString("1081000005FF010288016202E700E800")
	if !reflect.DeepEqual(s, expected) {
		t.Errorf("echoFrame.build() error: '%s' != '%s'", hex.EncodeToString(s), hex.EncodeToString(expected))
	}
}

func TestEchoFrameCorrespondTo(t *testing.T) {
	req := NewFrame(LvSmartElectricEnergyMeter, Get, []*Property{
		NewProperty(LvSmartElectricEnergyMeter_InstantaneousCurrent, nil),
	})
	req.TID = 0xabcd

	decoded, _ := hex.DecodeString("1081ABCD02880105FF017201E80400140064")
	res, _ := ParseFrame(decoded)

	if !req.CorrespondTo(res) {
		t.Errorf("echoFrame.CorrespondTo() error: '%s' vs '%s'", hex.EncodeToString(req.Build()), hex.EncodeToString(res.Build()))
	}
	if !res.CorrespondTo(req) {
		t.Errorf("echoFrame.CorrespondTo() error: '%s' vs '%s'", hex.EncodeToString(res.Build()), hex.EncodeToString(req.Build()))
	}

	decoded2, _ := hex.DecodeString("1081ABCE02880105FF017201E80400140064")
	res2, _ := ParseFrame(decoded2)

	if req.CorrespondTo(res2) {
		t.Errorf("echoFrame.CorrespondTo() error: '%s' vs '%s'", hex.EncodeToString(req.Build()), hex.EncodeToString(res2.Build()))
	}

	decoded3, _ := hex.DecodeString("1081ABCD02880205FF017201E80400140064")
	res3, _ := ParseFrame(decoded3)

	if req.CorrespondTo(res3) {
		t.Errorf("echoFrame.CorrespondTo() error: '%s' vs '%s'", hex.EncodeToString(req.Build()), hex.EncodeToString(res3.Build()))
	}

	decoded4, _ := hex.DecodeString("1081ABCD02880205FF017201E80400140064")
	res4, _ := ParseFrame(decoded4)

	if req.CorrespondTo(res4) {
		t.Errorf("echoFrame.CorrespondTo() error: '%s' vs '%s'", hex.EncodeToString(req.Build()), hex.EncodeToString(res4.Build()))
	}

	decoded5, _ := hex.DecodeString("1081ABCD02880105FF027201E80400140064")
	res5, _ := ParseFrame(decoded5)

	if req.CorrespondTo(res5) {
		t.Errorf("echoFrame.CorrespondTo() error: '%s' vs '%s'", hex.EncodeToString(req.Build()), hex.EncodeToString(res5.Build()))
	}

	decoded6, _ := hex.DecodeString("1081ABCD02880105FF017101E80400140064")
	res6, _ := ParseFrame(decoded6)

	if req.CorrespondTo(res6) {
		t.Errorf("echoFrame.CorrespondTo() error: '%s' vs '%s'", hex.EncodeToString(req.Build()), hex.EncodeToString(res6.Build()))
	}

	decoded7, _ := hex.DecodeString("1081ABCD02880105FF017201E90400140064")
	res7, _ := ParseFrame(decoded7)

	if req.CorrespondTo(res7) {
		t.Errorf("echoFrame.CorrespondTo() error: '%s' vs '%s'", hex.EncodeToString(req.Build()), hex.EncodeToString(res7.Build()))
	}
}
