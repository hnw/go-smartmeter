# go-smartmeter

Wi-SUNモジュールを使って電力スマートメーターにアクセスするGoライブラリです。

## 特徴

- 致命的エラーとリトライ可能エラーを区別して可能ならリトライする（Wi-SUNの通信は不安定なので実用上はリトライ実装が重要）
- SKSCANで指定チャンネルだけスキャンできるようにしたので、チャンネルがわかっていれば再スキャンが高速
- ECHONET Liteの複数プロパティを1コマンドにまとめられるので、920MHz帯の節約になる
- Bルート専用モジュールとデュアルスタックモジュール両対応
- Go製で実行ファイルが依存のない小さいバイナリになるので、Raspberry Piなど低スペック環境でも動作させやすい

## サンプルコード

次のようなコードでスマートメーターの瞬時電力計測値にアクセスできます。（Wi-SUNモジュールの入手とBルートサービスの申し込みは必須です）

```go
package main

import (
	"fmt"

	sm "github.com/hnw/go-smartmeter"
)

func main() {
	con, err := sm.Open("/dev/ttyACM0",
		sm.DualStackSK(), // Bルート専用モジュールを使う場合はコメントアウト
		sm.ID("00000000000000000000000000000000"), // Bルート認証ID
		sm.Password("AB0123456789"),               // パスワード
		sm.Channel("33"))                          // チャンネル。各環境でScan()で取得した値に書き換える。

	if err != nil {
		fmt.Printf("%+v", err)
		return
	}

	err = con.Authenticate()
	if err != nil {
		fmt.Printf("%+v\n", err)
		return
	}

	request := sm.NewEchoFrame(sm.LvSmartElectricEnergyMeter, sm.Get, []*sm.EchoProperty{
		sm.NewEchoProperty(sm.LvSmartElectricEnergyMeter_InstantaneousElectricPower, nil),
	})
	response, err = con.QueryEchoRequest(request, sm.Retry(3))
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


