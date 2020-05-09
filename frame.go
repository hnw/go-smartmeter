package smartmeter

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"math/rand"
	"time"
)

type ClassCode uint32
type ServiceCode byte

/*
 * 参考資料
 *   ECHONET Lite規格書 『第2部 ECHONET Lite 通信ミドルウェア仕様』「第3章 電文構成（フレームフォーマット）」
 *   ECHONET Lite規格書 『第2部 ECHONET Lite 通信ミドルウェア仕様』「6.10 プロファイルオブジェクトクラスグループ規定」
 *   ECHONET Lite規格書 『APPENDIX ECHONET機器オブジェクト詳細規定 Release I』「3.3.25 低圧スマート電力量メータクラス規定」
 */

const (
	HeaderEchonetLite                      = 0x1081   // 0x10=ECHONET Lite, 0x81=電文形式1
	Controller                 ClassCode   = 0x05ff01 // コントローラ
	NodeProfile                ClassCode   = 0x0ef001 // ノードプロファイル
	LvSmartElectricEnergyMeter ClassCode   = 0x028801 // 低圧スマート電力量メータ
	Get                        ServiceCode = 0x62
	GetRes                     ServiceCode = 0x72
)

// Frame はECHONET Liteのフレームに対応する構造体
// 複数のプロパティの操作を1フレームにまとめて送信することができる
type Frame struct {
	TID        uint16      // トランザクションID
	SEOJ       ClassCode   // 送信元ECHONET Liteオブジェクト
	DEOJ       ClassCode   // 相手先ECHONET Liteオブジェクト
	ESV        ServiceCode // ECHONET Liteサービス
	Properties []*Property // ECHONETプロパティ
}

// NewFrame は Frame構造体のコンストラクタ関数
func NewFrame(dstClassCode ClassCode, esv ServiceCode, props []*Property) *Frame {
	f := &Frame{
		SEOJ:       Controller,
		DEOJ:       dstClassCode,
		ESV:        esv,
		Properties: props,
	}
	f.RegenerateTID()
	return f
}

// ParseFrame は ECHONET Liteフレームのバイト列を受け取り、Frame構造体として返す
func ParseFrame(raw []byte) (f *Frame, err error) {
	if len(raw) < 14 {
		return nil, errors.New("Too short ECHONET Lite frame")
	}
	if binary.BigEndian.Uint16(raw[0:2]) != HeaderEchonetLite {
		return nil, fmt.Errorf("Unknown ECHONET Lite Header: %02X%02X", raw[0], raw[1])
	}
	// トランザクションID
	tid := binary.BigEndian.Uint16(raw[2:4])
	// 送信元ECHONET Liteオブジェクト
	v32 := binary.BigEndian.Uint32(raw[3:7]) // [4:7]
	v32 &= 0x00ffffff
	seoj := ClassCode(v32)
	// 相手先ECHONET Liteオブジェクト
	v32 = binary.BigEndian.Uint32(raw[6:10]) // [7:10]
	v32 &= 0x00ffffff
	deoj := ClassCode(v32)
	// ECHONET Liteサービス
	esv := ServiceCode(raw[10])
	// 処理対象プロパティカウンタ (OPC)
	nProperty := int(raw[11])

	props := make([]*Property, nProperty)
	i := 12
	for j := 0; j < nProperty; j++ {
		if len(raw) < i+2 {
			err = errors.New("Too short ECHONET Lite frame")
			return
		}
		// プロパティデータカウンタ (PDC)
		lenEDT := int(raw[i+1])
		if len(raw) < i+2+lenEDT {
			err = errors.New("Too short ECHONET Lite frame")
			return
		}
		// プロパティ値データ(EDT)
		data := raw[i+2 : i+2+lenEDT]
		props[j] = NewProperty(PropertyCode(raw[i]), data)
		i = i + 2 + lenEDT
	}

	return &Frame{TID: tid, SEOJ: seoj, DEOJ: deoj, ESV: esv, Properties: props}, nil
}

func (f *Frame) Build() []byte {
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.BigEndian, uint16(HeaderEchonetLite))
	// トランザクションID
	binary.Write(buf, binary.BigEndian, f.TID)
	// 送信元ECHONET Liteオブジェクト
	binary.Write(buf, binary.BigEndian, uint8(f.SEOJ>>16&0xff))
	binary.Write(buf, binary.BigEndian, uint16(f.SEOJ&0xffff))
	// 相手先ECHONET Liteオブジェクト
	binary.Write(buf, binary.BigEndian, uint8(f.DEOJ>>16&0xff))
	binary.Write(buf, binary.BigEndian, uint16(f.DEOJ&0xffff))
	// ECHONET Liteサービス
	binary.Write(buf, binary.BigEndian, f.ESV)
	// 処理対象プロパティカウンタ (OPC)
	nProperty := len(f.Properties)
	binary.Write(buf, binary.BigEndian, uint8(nProperty))
	for i := 0; i < nProperty; i++ {
		buf.Write(f.Properties[i].Build())
	}
	return buf.Bytes()
}

// CorrespondTo は fとtargetとがリクエスト/レスポンスとして対応しているか確認する
func (f *Frame) CorrespondTo(target *Frame) bool {
	if f.TID != target.TID {
		return false
	}
	if f.SEOJ != target.DEOJ {
		return false
	}
	if f.DEOJ != target.SEOJ {
		return false
	}
	delta := int(f.ESV) - int(target.ESV)
	if delta != -0x10 && delta != 0x10 {
		return false
	}
	if len(f.Properties) == 0 {
		return false
	}
	if len(f.Properties) != len(target.Properties) {
		// TODO: プロパティ数が多すぎるとレスポンスが分割されるので(?)一致しないことがある
		return false
	}
	/*
		opc := len(f.Properties)
		for i := 0; i < opc; i++ {
			if f.Properties[i] != target.Properties[i] {
				return false
			}
		}
	*/
	return true
}

// RegenerateTID はFrameのTIDを再生成する
func (f *Frame) RegenerateTID() {
	rand.Seed(time.Now().UnixNano()) // 時刻をseedにする（ランダム性・予測不可能性が重要ではないため）
	f.TID = uint16(rand.Int31n(0x10000))
}
