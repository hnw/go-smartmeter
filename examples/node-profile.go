// +build example
//
// Do not build by default.

// ノードプロファイルから値を取得するデモ。

package main

import (
	"fmt"

	sm "github.com/hnw/go-smartmeter"
)

func main() {
	con, err := sm.Open("/dev/ttyACM0",
		//sm.Debug(),       // コマンドとレスポンスを全部確認したいときにアンコメントする
		sm.DualStackSK(), // Bルート専用モジュールを使う場合はコメントアウト
		sm.ID("00000000000000000000000000000000"), // Bルート認証ID
		sm.Password("AB0123456789"),               // パスワード
		sm.Channel("33"))                          // チャンネル。各環境でScan()で取得した値に書き換える。

	if err != nil {
		fmt.Printf("%+v", err)
		return
	}

	if con.IPAddr == "" {
		ipAddr, err := con.GetNeibourIP()
		if err == nil {
			con.IPAddr = ipAddr
		}
	}

	request := sm.NewEchoFrame(sm.NodeProfile, sm.Get, []*sm.EchoProperty{
		sm.NewEchoProperty(sm.NodeProfile_VersionInformation, nil),
		sm.NewEchoProperty(sm.NodeProfile_ManufacturerCode, nil),
		sm.NewEchoProperty(sm.NodeProfile_SelfNodeInstanceListS, nil),
	})
	response, err := con.QueryEchoRequest(request, sm.Retry(3))
	if err != nil {
		fmt.Printf("Error: %+v\n", err)

		// 値が取得できなかったので、認証してから再度値を取る
		err = con.Authenticate()
		if err != nil {
			fmt.Printf("%+v\n", err)
			return
		}
		response, err = con.QueryEchoRequest(request, sm.Retry(3))
	}

	for _, p := range response.Properties {
		fmt.Print(p.Desc())
	}
}
