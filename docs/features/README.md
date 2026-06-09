# フィーチャードキュメント

このディレクトリは、各フィーチャー（垂直スライス）の設計・API 仕様・シーケンス図・
依存関係をまとめたドキュメントを集約しています。各フィーチャーの実装は
`internal/feature/<name>/` にあります。

## 一覧

| フィーチャー | 概要 |
| --- | --- |
| [auth](auth.md) | JWT 認証・OAuth2 ソーシャルログイン（Google / GitHub）・パスワード管理 |
| [candles](candles.md) | ローソク足データの取得・集約・Redis キャッシュ |
| [symbollist](symbollist.md) | シンボル一覧取得・ロゴ URL のバッチ取り込み |
| [watchlist](watchlist.md) | ウォッチリストの取得・追加・削除・並び替え |
| [logodetection](logodetection.md) | 画像からのロゴ検出（Cloud Vision）・企業分析（Gemini） |

## 補足

- 各ドキュメント内のソースコードへのリンクは、リポジトリルートからの相対パス
  （`../../internal/feature/<name>/...`）で記述しています。
- アーキテクチャ全体の方針は [docs/adr/](../adr/)、DB スキーマは [docs/schema/](../schema/) を参照してください。
