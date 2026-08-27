# gep

`gep`（ゲップ、**G**et **E**DINET **P**DF）は、文書管理番号を指定して
[EDINET API Version 2](https://disclosure2dl.edinet-fsa.go.jp/guide/static/disclosure/WZEK0110.html)
から開示書類を取得する小さなコマンドラインツールです。

分析中に「この数値を原本のPDFですぐ確認したい」という場面を想定しています。
既定では `文書管理番号.pdf` という名前でカレントディレクトリに保存します。

## インストール

Go 1.22以降がインストールされている場合は、次のコマンドでインストールできます。

```sh
go install github.com/k2chiya/gep@latest
```

インストール後に `gep` が見つからない場合は、Goのバイナリ保存先を `PATH` に追加してください。

```sh
export PATH="$(go env GOPATH)/bin:$PATH"
```

## APIキーの設定

EDINET API Version 2の利用にはAPIキーが必要です。取得したキーを環境変数
`EDINET_API_KEY` に設定します。

```sh
export EDINET_API_KEY='発行されたAPIキー'
```

毎回設定したくない場合は、この行を `~/.zshrc` などのシェル設定ファイルに追加します。
APIキーをソースコードやGit管理対象のファイルへ書かないでください。

## 使い方

文書管理番号だけを指定するとPDFを取得します。

```sh
gep S100XXXX
```

成功すると `S100XXXX.pdf` を保存し、その絶対パスを標準出力に表示します。

```sh
open "$(gep S100XXXX)"       # macOS：取得後すぐ開く
xdg-open "$(gep S100XXXX)"  # Linux：取得後すぐ開く
```

保存先ディレクトリも指定できます。

```sh
gep --output-dir ./reports S100XXXX
```

### PDF以外の取得

EDINET APIの `type` を指定すると、XBRLや添付書類などをZIPで取得できます。

| type | 内容 | 保存名 |
|---:|---|---|
| 1 | 提出本文書および監査報告書（XBRLなど） | `文書管理番号.zip` |
| 2 | PDF（既定値） | `文書管理番号.pdf` |
| 3 | 代替書面・添付文書 | `文書管理番号.zip` |
| 4 | 英文ファイル | `文書管理番号.zip` |
| 5 | CSV | `文書管理番号.zip` |

```sh
gep --type 1 S100XXXX
```

タイムアウトの既定値は60秒です。Goのduration表記で変更できます。

```sh
gep --timeout 2m S100XXXX
```

## シェルスクリプトでの利用

成功時は保存先だけを標準出力へ、エラーは標準エラーへ出力します。失敗時の終了コードは1です。
途中で取得に失敗した場合、不完全な出力ファイルは残しません。

```sh
if pdf=$(gep "$doc_id"); then
    printf 'saved: %s\n' "$pdf"
else
    printf 'download failed: %s\n' "$doc_id" >&2
fi
```

複数の文書管理番号は、例えば次のように処理できます。

```sh
while IFS= read -r doc_id; do
    gep --output-dir ./reports "$doc_id" || exit 1
done < doc_ids.txt
```

## 開発

```sh
go test ./...
go build .
```

## 注意事項

- EDINETの利用条件およびアクセス頻度に配慮して使用してください。
- 本ツールは金融庁・EDINETの公式ツールではありません。
- EDINET APIの仕様変更により動作しなくなる可能性があります。

## ライセンス

[MIT License](LICENSE)
