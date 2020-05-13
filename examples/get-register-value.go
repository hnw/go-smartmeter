// +build example
//
// Do not build by default.

// ID,PWなしで利用できるコマンドのデモ

package main

import (
	"fmt"

	smartmeter "github.com/hnw/go-smartmeter"
)

func main() {
	dev, err := smartmeter.Open("/dev/ttyACM0",
		//smartmeter.Verbosity(3),                           // コマンドとレスポンスを全部確認したいときにアンコメントする
		smartmeter.DualStackSK(true)) // Bルート専用モジュールを使う場合はコメントアウト

	if err != nil {
		fmt.Printf("%+v", err)
	}

	version, err := dev.GetVersion()
	if err != nil {
		fmt.Printf("%+v\n", err)
	} else {
		fmt.Printf("Version: %v\n", version)
	}

	info, err := dev.GetInfo()
	if err != nil {
		fmt.Printf("%+v\n", err)
	} else {
		fmt.Printf("Info: %v\n", info)
	}

	regNames := []string{"S02", "S03", "S07", "S0A", "S15", "S16", "S17", "SA0", "SA1", "SFB", "SFD", "SFE", "SFF"}
	for _, regName := range regNames {
		regValue, err := dev.GetRegisterValue(regName)
		if err != nil {
			fmt.Printf("%+v\n", err)
		} else {
			fmt.Printf("%s: %v\n", regName, regValue)
		}
	}
}
