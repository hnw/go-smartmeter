package smartmeter

import (
	"encoding/binary"
	"fmt"
	"math"
)

// PropertyCode identifies an ECHONET Lite property code (EPC).
type PropertyCode byte

/*
 * 参考資料
 *   ECHONET Lite規格書 『第2部 ECHONET Lite 通信ミドルウェア仕様』
 *   「第3章 電文構成（フレームフォーマット）」
 *   ECHONET Lite規格書 『第2部 ECHONET Lite 通信ミドルウェア仕様』
 *   「6.10 プロファイルオブジェクトクラスグループ規定」
 *   ECHONET Lite規格書 『APPENDIX ECHONET機器オブジェクト詳細規定 Release I』
 *   「3.3.25 低圧スマート電力量メータクラス規定」
 */

// Property codes for node profile and low-voltage smart electric energy meter.
const (
	NodeProfileVersionInformation   PropertyCode = 0x82 // Version情報
	NodeProfileIdentificationNumber PropertyCode = 0x83 // 識別番号
	NodeProfileFaultStatus          PropertyCode = 0x88
	NodeProfileFaultContent         PropertyCode = 0x89
	NodeProfileManufacturerCode     PropertyCode = 0x8a // メーカコード
	NodeProfileBusinessFacilityCode PropertyCode = 0x8b // 事業場コード
	NodeProfileProductCode          PropertyCode = 0x8c // 商品コード
	NodeProfileProductionNumber     PropertyCode = 0x8d // 製造番号
	NodeProfileProductionDate       PropertyCode = 0x8e // 製造年月日
	NodeProfileUniqueIdentifierData PropertyCode = 0xbf // 個体識別情報
	// NodeProfileNumberOfSelfNodeInstances PropertyCode = 0xd3
	// 自ノードインスタンス数（作者の環境では1）
	// 自ノードクラス数（作者の環境では2）
	NodeProfileNumberOfSelfNodeClasses  PropertyCode = 0xd4
	NodeProfileInstanceListNotification PropertyCode = 0xd5
	NodeProfileSelfNodeInstanceListS    PropertyCode = 0xd6 // 自ノードインスタンスリストS
	NodeProfileSelfNodeClassListS       PropertyCode = 0xd7 // 自ノードクラスリストS

	// 係数（作者の環境では1）
	LvSmartElectricEnergyMeterCoefficient PropertyCode = 0xd3
	// 積算電力量（正方向）
	LvSmartElectricEnergyMeterNormalDirectionCumulativeElectricEnergy PropertyCode = 0xe0
	// 積算電力量単位（作者の環境では0.1kWh）
	LvSmartElectricEnergyMeterUnitForCumulativeAmountsOfElectricEnergy PropertyCode = 0xe1
	// 積算電力量（逆方向）
	LvSmartElectricEnergyMeterReverseDirectionCumulativeElectricEnergy PropertyCode = 0xe3
	// 瞬時電力計測値
	LvSmartElectricEnergyMeterInstantaneousElectricPower PropertyCode = 0xe7
	// 瞬時電流計測値
	LvSmartElectricEnergyMeterInstantaneousCurrent PropertyCode = 0xe8
	// 定時積算電力量(正方向)
	LvSmartElectricEnergyMeterNormalDirectionCumulativeElectricEnergyAtEvery30Min PropertyCode = 0xea
	// 定時積算電力量(逆方向)
	LvSmartElectricEnergyMeterReverseDirectionCumulativeElectricEnergyAtEvery30Min PropertyCode = 0xeb
)

// Property はECHONET Liteのプロパティに対応する構造体。
type Property struct {
	EPC PropertyCode // ECHONETプロパティ
	EDT []byte       // 要求電文プロパティ値データ(EDT)
}

// NewProperty は Property構造体のコンストラクタ関数。
func NewProperty(epc PropertyCode, edt []byte) *Property {
	return &Property{EPC: epc, EDT: edt}
}

// Build はプロパティをバイト列として組み立てる。
func (p *Property) Build() []byte {
	buf := make([]byte, 0, 2+len(p.EDT))
	// ECHONETプロパティ
	buf = append(buf, byte(p.EPC))
	// プロパティデータカウンタ
	buf = append(buf, byte(len(p.EDT)))
	// プロパティ値データ
	buf = append(buf, p.EDT...)
	return buf
}

var propertyDescFuncs = map[PropertyCode]func(*Property) string{}

func registerPropertyDesc(code PropertyCode, desc func(*Property) string) {
	propertyDescFuncs[code] = desc
}

func init() {
	registerPropertyDesc(NodeProfileVersionInformation, propertyDescVersionInformation)
	registerPropertyDesc(NodeProfileManufacturerCode, propertyDescManufacturerCode)
	registerPropertyDesc(NodeProfileSelfNodeInstanceListS, propertyDescSelfNodeInstanceListS)
	registerPropertyDesc(LvSmartElectricEnergyMeterCoefficient, propertyDescCoefficient)
	registerPropertyDesc(
		LvSmartElectricEnergyMeterUnitForCumulativeAmountsOfElectricEnergy,
		propertyDescUnit,
	)
	registerPropertyDesc(
		LvSmartElectricEnergyMeterNormalDirectionCumulativeElectricEnergy,
		propertyDescCumulativeEnergy("normal"),
	)
	registerPropertyDesc(
		LvSmartElectricEnergyMeterReverseDirectionCumulativeElectricEnergy,
		propertyDescCumulativeEnergy("reverse"),
	)
	registerPropertyDesc(
		LvSmartElectricEnergyMeterInstantaneousElectricPower,
		propertyDescInstantaneousPower,
	)
	registerPropertyDesc(
		LvSmartElectricEnergyMeterInstantaneousCurrent,
		propertyDescInstantaneousCurrent,
	)
	registerPropertyDesc(
		LvSmartElectricEnergyMeterNormalDirectionCumulativeElectricEnergyAtEvery30Min,
		propertyDescCumulativeEnergyEvery30Min("normal"),
	)
	registerPropertyDesc(
		LvSmartElectricEnergyMeterReverseDirectionCumulativeElectricEnergyAtEvery30Min,
		propertyDescCumulativeEnergyEvery30Min("reverse"),
	)
}

// Desc はプロパティの内容を文字列として返す。
func (p *Property) Desc() string {
	if descFn, ok := propertyDescFuncs[p.EPC]; ok {
		return descFn(p)
	}
	return fmt.Sprintf("EPC=0x%02x: %v\n", p.EPC, p.EDT)
}

func propertyDescVersionInformation(p *Property) string {
	return fmt.Sprintf("Version information: %d.%d\n", p.EDT[0], p.EDT[1])
}

func propertyDescManufacturerCode(p *Property) string {
	code := uint24FromBytes(p.EDT)
	return fmt.Sprintf("Manufacturer code: 0x%06X\n", code)
}

func propertyDescSelfNodeInstanceListS(p *Property) string {
	result := "Self node instance list: [ "
	for i := 0; i < int(p.EDT[0]); i++ {
		start := i*3 + 1
		result += fmt.Sprintf("0x%06x ", uint24FromBytes(p.EDT[start:start+3]))
	}
	result += "]\n"
	return result
}

func propertyDescCoefficient(p *Property) string {
	return fmt.Sprintf("Coefficient: %d\n", binary.BigEndian.Uint32(p.EDT))
}

func propertyDescUnit(p *Property) string {
	unit := 1.0
	if p.EDT[0] >= 0x1 && p.EDT[0] <= 0x4 {
		unit *= math.Pow(10, -float64(p.EDT[0]))
	} else if p.EDT[0] >= 0xa && p.EDT[0] <= 0xd {
		unit *= math.Pow(10, float64(p.EDT[0]-0x9))
	}
	return fmt.Sprintf(
		"Unit for cumulative amounts of electric energy: %f [kWh]\n",
		unit,
	)
}

func propertyDescCumulativeEnergy(direction string) func(*Property) string {
	return func(p *Property) string {
		value := float64(signedInt32FromBytes(p.EDT)) / 10.0
		return fmt.Sprintf(
			"Cumulative Electric Energy (%s direction): %f [kWh]\n",
			direction,
			value,
		)
	}
}

func propertyDescInstantaneousPower(p *Property) string {
	value := float64(signedInt32FromBytes(p.EDT))
	return fmt.Sprintf("Instantaneous Electric Power: %f [W]\n", value)
}

func propertyDescInstantaneousCurrent(p *Property) string {
	rPhase := float64(signedInt16FromBytes(p.EDT[:2])) / 10.0
	tPhase := float64(signedInt16FromBytes(p.EDT[2:])) / 10.0
	return fmt.Sprintf(
		"Instantaneous Current (R-phase): %f [A]\nInstantaneous Current (T-phase): %f [A]\n",
		rPhase,
		tPhase,
	)
}

func propertyDescCumulativeEnergyEvery30Min(direction string) func(*Property) string {
	return func(p *Property) string {
		year := binary.BigEndian.Uint16(p.EDT[:2])
		month := p.EDT[2]
		day := p.EDT[3]
		hour := p.EDT[4]
		minute := p.EDT[5]
		second := p.EDT[6]
		value := float64(binary.BigEndian.Uint32(p.EDT[7:])) / 10.0
		return fmt.Sprintf(
			"Cumulative Electric Energy (%04d-%02d-%02d %02d:%02d:%02d, %s direction): %f [kWh]\n",
			year,
			month,
			day,
			hour,
			minute,
			second,
			direction,
			value,
		)
	}
}

const (
	signed16Mod = int64(1) << 16
	signed32Mod = int64(1) << 32
)

func signedInt16FromBytes(b []byte) int64 {
	v := binary.BigEndian.Uint16(b)
	if v&0x8000 == 0 {
		return int64(v)
	}
	return int64(v) - signed16Mod
}

func signedInt32FromBytes(b []byte) int64 {
	v := binary.BigEndian.Uint32(b)
	if v&0x80000000 == 0 {
		return int64(v)
	}
	return int64(v) - signed32Mod
}

func uint24FromBytes(b []byte) uint32 {
	return uint32(b[0])<<16 | uint32(b[1])<<8 | uint32(b[2])
}
