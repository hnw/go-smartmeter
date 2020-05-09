// +build example
//
// Do not build by default.

// スキャンのデモ。
// 環境によっては50%くらいの確率で失敗するので、時間をおいて数回試してください。

package main

import (
	"fmt"
	"time"

	smartmeter "github.com/hnw/go-smartmeter"
)

func main() {
	dev, err := smartmeter.Open("/dev/ttyACM0",
		//smartmeter.Debug(true),                            // コマンドとレスポンスを全部確認したいときにアンコメントする
		smartmeter.DualStackSK(true),                      // Bルート専用モジュールを使う場合はコメントアウト
		smartmeter.ID("00000000000000000000000000000000"), // ルートB認証ID
		smartmeter.Password("AB0123456789"))               // パスワード

	if err != nil {
		fmt.Printf("%+v", err)
		return
	}

	err = dev.Scan(smartmeter.Timeout(100 * time.Second))
	if err != nil {
		fmt.Printf("%+v\n", err)
		return
	}
	fmt.Printf("%+v\n", dev)
}
