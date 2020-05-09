// +build example
//
// Do not build by default.

// 低圧スマート電力量メータから値を取得するデモ。

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

	request := smartmeter.NewFrame(smartmeter.LvSmartElectricEnergyMeter, smartmeter.Get, []*smartmeter.Property{
		smartmeter.NewProperty(smartmeter.LvSmartElectricEnergyMeter_InstantaneousElectricPower, nil),
	})
	// 瞬時電力計測値を表示し続ける。作者の環境では2〜6秒に1回のペースで値が取得できます。
	for {
		request.RegenerateTID()
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
			fmt.Printf("%s: %s", time.Now().Format("2006-01-02 15:04:05"), p.Desc())
		}
		time.Sleep(1 * time.Second)
	}
}
