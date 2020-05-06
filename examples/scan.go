// +build example
//
// Do not build by default.

// スキャンのデモ。
// 環境によっては50%くらいの確率で失敗するので、時間をおいて数回試してください。

package main

import (
	"fmt"
	"time"

	sm "github.com/hnw/go-smartmeter"
)

func main() {
	con, err := sm.Open("/dev/ttyACM0",
		//sm.Debug(),       // コマンドとレスポンスを全部確認したいときにアンコメントする
		sm.DualStackSK(), // Bルート専用モジュールを使う場合はコメントアウト
		sm.ID("00000000000000000000000000000000"), // ルートB認証ID
		sm.Password("AB0123456789"))               // パスワード

	if err != nil {
		fmt.Printf("%+v", err)
		return
	}

	err = con.Scan(sm.Timeout(100 * time.Second))
	if err != nil {
		fmt.Printf("%+v\n", err)
		return
	}
	fmt.Printf("%+v\n", con)
}
