// +build example
//
// Do not build by default.

// ノードプロファイルから値を取得するデモ。

package main

import (
	"fmt"

	smartmeter "github.com/hnw/go-smartmeter"
)

func main() {
	dev, err := smartmeter.Open("/dev/ttyACM0",
		//smartmeter.Verbosity(3),                           // コマンドとレスポンスを全部確認したいときにアンコメントする
		smartmeter.DualStackSK(true),                      // Bルート専用モジュールを使う場合はコメントアウト
		smartmeter.ID("00000000000000000000000000000000"), // Bルート認証ID
		smartmeter.Password("AB0123456789"),               // パスワード
		smartmeter.Channel("33"))                          // チャンネル。各環境でScan()で取得した値に書き換える。

	if err != nil {
		fmt.Printf("%+v", err)
		return
	}

	if dev.IPAddr == "" {
		ipAddr, err := dev.GetNeibourIP()
		if err == nil {
			dev.IPAddr = ipAddr
		}
	}

	request := smartmeter.NewFrame(smartmeter.NodeProfile, smartmeter.Get, []*smartmeter.Property{
		smartmeter.NewProperty(smartmeter.NodeProfile_VersionInformation, nil),
		smartmeter.NewProperty(smartmeter.NodeProfile_ManufacturerCode, nil),
		smartmeter.NewProperty(smartmeter.NodeProfile_SelfNodeInstanceListS, nil),
	})
	response, err := dev.QueryEchonetLite(request, smartmeter.Retry(3))
	if err != nil {
		fmt.Printf("Error: %+v\n", err)

		// 値が取得できなかったので、認証してから再度値を取る
		err = dev.Authenticate()
		if err != nil {
			fmt.Printf("%+v\n", err)
			return
		}
		response, err = dev.QueryEchonetLite(request, smartmeter.Retry(3))
	}

	for _, p := range response.Properties {
		fmt.Print(p.Desc())
	}
}
