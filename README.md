# go-smartmeter

Wi-SUNモジュールを使って電力スマートメーターにアクセスするGoライブラリです。

## 特徴

- 致命的エラーとリトライ可能エラーを区別して可能ならリトライする（Wi-SUNの通信は不安定なので実用上はリトライ実装が重要）
- SKSCANで指定チャンネルだけスキャンできるようにしたので、チャンネルがわかっていれば再スキャンが高速
- ECHONET Liteの複数プロパティを1コマンドにまとめられるので、920MHz帯の節約になる
- Bルート専用モジュールとデュアルスタックモジュール両対応
- 実行ファイルが外部ライブラリ依存のない小さいバイナリになるので、Raspberry Piなど低スペック環境でも動作させやすい

## サンプルコード

次のようなコードでスマートメーターの瞬時電力計測値にアクセスできます。（Wi-SUNモジュールの入手とBルートサービスの申し込みは必須です）

```go
package main

import (
	"fmt"

	smartmeter "github.com/hnw/go-smartmeter"
)

func main() {
	dev, err := smartmeter.Open("/dev/ttyACM0",
		smartmeter.DualStackSK(), // Bルート専用モジュールを使う場合はコメントアウト
		smartmeter.ID("00000000000000000000000000000000"), // Bルート認証ID
		smartmeter.Password("AB0123456789"),               // パスワード
		smartmeter.Channel("33"))                          // チャンネル。各環境でScan()で取得した値に書き換える。

	if err != nil {
		fmt.Printf("%+v", err)
		return
	}

	err = dev.Authenticate()
	if err != nil {
		fmt.Printf("%+v\n", err)
		return
	}

	request := smartmeter.NewFrame(smartmeter.LvSmartElectricEnergyMeter, smartmeter.Get, []*smartmeter.Property{
		smartmeter.NewProperty(smartmeter.LvSmartElectricEnergyMeter_InstantaneousElectricPower, nil),
	})
	response, err = dev.QueryEchonetLite(request, smartmeter.Retry(3))
	if err != nil {
		fmt.Printf("%+v\n", err)
		return
	}

	for _, p := range response.Properties {
		fmt.Print(p.Desc())
	}
}
```

これを実行すると、次のように自宅の消費電力がわかります。

```
Instantaneous Electric Power: 389.000000 [W]
```

[examples/](examples/)以下に利用例がありますので参考にしてください。


