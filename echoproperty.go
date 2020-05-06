package smartmeter

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
)

type PropertyCode byte

/*
 * 参考資料
 *   ECHONET Lite規格書 『第2部 ECHONET Lite 通信ミドルウェア仕様』「第3章 電文構成（フレームフォーマット）」
 *   ECHONET Lite規格書 『第2部 ECHONET Lite 通信ミドルウェア仕様』「6.10 プロファイルオブジェクトクラスグループ規定」
 *   ECHONET Lite規格書 『APPENDIX ECHONET機器オブジェクト詳細規定 Release I』「3.3.25 低圧スマート電力量メータクラス規定」
 */

const (
	NodeProfile_VersionInformation   PropertyCode = 0x82 // Version情報
	NodeProfile_IdentificationNumber PropertyCode = 0x83 // 識別番号
	NodeProfile_FaultStatus          PropertyCode = 0x88
	NodeProfile_FaultContent         PropertyCode = 0x89
	NodeProfile_ManufacturerCode     PropertyCode = 0x8a // メーカコード
	NodeProfile_BusinessFacilityCode PropertyCode = 0x8b // 事業場コード
	NodeProfile_ProductCode          PropertyCode = 0x8c // 商品コード
	NodeProfile_ProductionNumber     PropertyCode = 0x8d // 製造番号
	NodeProfile_ProductionDate       PropertyCode = 0x8e // 製造年月日
	NodeProfile_UniqueIdentifierData PropertyCode = 0xbf // 個体識別情報
	//NodeProfile_NumberOfSelfNodeInstances PropertyCode = 0xd3 // 自ノードインスタンス数（作者の環境では1）
	NodeProfile_NumberOfSelfNodeClasses  PropertyCode = 0xd4 // 自ノードクラス数（作者の環境では2）
	NodeProfile_InstanceListNotification PropertyCode = 0xd5
	NodeProfile_SelfNodeInstanceListS    PropertyCode = 0xd6 // 自ノードインスタンスリストS
	NodeProfile_SelfNodeClassListS       PropertyCode = 0xd7 // 自ノードクラスリストS

	LvSmartElectricEnergyMeter_Coefficient                                          PropertyCode = 0xd3 // 係数（作者の環境では1）
	LvSmartElectricEnergyMeter_NormalDirectionCumulativeElectricEnergy              PropertyCode = 0xe0 // 積算電力量（正方向）
	LvSmartElectricEnergyMeter_UnitForCumulativeAmountsOfElectricEnergy             PropertyCode = 0xe1 // 積算電力量単位（作者の環境では0.1kWh）
	LvSmartElectricEnergyMeter_ReverseDirectionCumulativeElectricEnergy             PropertyCode = 0xe3 // 積算電力量（逆方向）
	LvSmartElectricEnergyMeter_InstantaneousElectricPower                           PropertyCode = 0xe7 // 瞬時電力計測値
	LvSmartElectricEnergyMeter_InstantaneousCurrent                                 PropertyCode = 0xe8 // 瞬時電流計測値
	LvSmartElectricEnergyMeter_NormalDirectionCumulativeElectricEnergyAtEvery30Min  PropertyCode = 0xea // 定時積算電力量(正方向)
	LvSmartElectricEnergyMeter_ReverseDirectionCumulativeElectricEnergyAtEvery30Min PropertyCode = 0xeb // 定時積算電力量(逆方向)
)

type EchoProperty struct {
	EPC PropertyCode // ECHONETプロパティ
	EDT []byte       // 要求電文プロパティ値データ(EDT)
}

// NewEchoProperty は EchoProperty構造体のコンストラクタ関数
func NewEchoProperty(epc PropertyCode, edt []byte) *EchoProperty {
	return &EchoProperty{EPC: epc, EDT: edt}
}

func (p *EchoProperty) Build() []byte {
	buf := new(bytes.Buffer)
	// ECHONETプロパティ
	binary.Write(buf, binary.BigEndian, p.EPC)
	// プロパティデータカウンタ
	binary.Write(buf, binary.BigEndian, uint8(len(p.EDT)))
	// プロパティ値データ
	buf.Write(p.EDT)

	return buf.Bytes()
}

func (p *EchoProperty) Desc() (result string) {
	switch p.EPC {
	case NodeProfile_VersionInformation:
		// Version情報
		result = fmt.Sprintf("Version information: %d.%d\n", p.EDT[0], p.EDT[1])
	case NodeProfile_ManufacturerCode:
		// メーカコード
		result = fmt.Sprintf("Manufacturer code: 0x%06X\n", binary.BigEndian.Uint32(append([]byte{0}, p.EDT...)))
	case NodeProfile_SelfNodeInstanceListS:
		// 自ノードインスタンスリストS
		result = "Self node instance list: [ "
		for i := 0; i < int(p.EDT[0]); i++ {
			result += fmt.Sprintf("0x%06x ", binary.BigEndian.Uint32(append([]byte{0}, p.EDT[i*3+1:i*3+4]...)))
		}
		result += "]\n"

	case LvSmartElectricEnergyMeter_Coefficient:
		// 係数
		result = fmt.Sprintf("Coefficient: %d\n", binary.BigEndian.Uint32(p.EDT))
	case LvSmartElectricEnergyMeter_UnitForCumulativeAmountsOfElectricEnergy:
		// 積算電力量単位
		unit := 1.0
		if p.EDT[0] >= 0x1 && p.EDT[0] <= 0x4 {
			unit *= math.Pow(10, -float64(p.EDT[0]))
		} else if p.EDT[0] >= 0xa && p.EDT[0] <= 0xd {
			unit *= math.Pow(10, float64(p.EDT[0]-0x9))
		}
		result = fmt.Sprintf("Unit for cumulative amounts of electric energy: %f [kWh]\n", unit)
	case LvSmartElectricEnergyMeter_NormalDirectionCumulativeElectricEnergy, LvSmartElectricEnergyMeter_ReverseDirectionCumulativeElectricEnergy:
		// 積算電力量
		direction := "normal"
		if p.EPC == LvSmartElectricEnergyMeter_ReverseDirectionCumulativeElectricEnergy {
			direction = "reverse"
		}
		result = fmt.Sprintf("Cumulative Electric Energy (%s direction): %f [kWh]\n",
			direction,
			float64(int32(binary.BigEndian.Uint32(p.EDT)))/10.0)
	case LvSmartElectricEnergyMeter_InstantaneousElectricPower:
		// 瞬時電力計測値
		result = fmt.Sprintf("Instantaneous Electric Power: %f [W]\n", float64(int32(binary.BigEndian.Uint32(p.EDT))))
	case LvSmartElectricEnergyMeter_InstantaneousCurrent:
		// 瞬時電流計測値
		result = fmt.Sprintf("Instantaneous Current (R-phase): %f [A]\n", float64(int16(binary.BigEndian.Uint16(p.EDT[:2])))/10.0)
		result += fmt.Sprintf("Instantaneous Current (T-phase): %f [A]\n", float64(int16(binary.BigEndian.Uint16(p.EDT[2:])))/10.0)
	case LvSmartElectricEnergyMeter_NormalDirectionCumulativeElectricEnergyAtEvery30Min, LvSmartElectricEnergyMeter_ReverseDirectionCumulativeElectricEnergyAtEvery30Min:
		// 定時積算電力量
		direction := "normal"
		if p.EPC == LvSmartElectricEnergyMeter_ReverseDirectionCumulativeElectricEnergyAtEvery30Min {
			direction = "reverse"
		}
		result = fmt.Sprintf("Cumulative Electric Energy (%04d-%02d-%02d %02d:%02d:%02d, %s direction): %f [kWh]\n",
			binary.BigEndian.Uint16(p.EDT[:2]), p.EDT[2], p.EDT[3],
			p.EDT[4], p.EDT[5], p.EDT[6],
			direction,
			float64(binary.BigEndian.Uint32(p.EDT[7:]))/10.0,
		)
	default:
		result = fmt.Sprintf("EPC=0x%02x: %v\n", p.EPC, p.EDT)
	}
	return
}
