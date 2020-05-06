// +build example
//
// Do not build by default.

// ID,PWなしで利用できるコマンドのデモ

package main

import (
	"fmt"

	sm "github.com/hnw/go-smartmeter"
)

func main() {
	con, err := sm.Open("/dev/ttyACM0",
		//sm.Debug(),       // コマンドとレスポンスを全部確認したいときにアンコメントする
		sm.DualStackSK()) // Bルート専用モジュールを使う場合はコメントアウト

	if err != nil {
		fmt.Printf("%+v", err)
	}

	version, err := con.GetVersion()
	if err != nil {
		fmt.Printf("%+v\n", err)
	} else {
		fmt.Printf("Version: %v\n", version)
	}

	info, err := con.GetInfo()
	if err != nil {
		fmt.Printf("%+v\n", err)
	} else {
		fmt.Printf("Info: %v\n", info)
	}

	regNames := []string{"S02", "S03", "S07", "S0A", "S15", "S16", "S17", "SA0", "SA1", "SFB", "SFD", "SFE", "SFF"}
	for _, regName := range regNames {
		regValue, err := con.GetRegisterValue(regName)
		if err != nil {
			fmt.Printf("%+v\n", err)
		} else {
			fmt.Printf("%s: %v\n", regName, regValue)
		}
	}
}
