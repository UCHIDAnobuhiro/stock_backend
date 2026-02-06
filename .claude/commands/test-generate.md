指定されたファイルまたは変更差分に対してテストコードを生成してください。

引数: $ARGUMENTS

## 手順

### Step 1: テスト対象の特定

1. 引数にファイルパスが指定されている場合はそのファイルを対象とする
2. 引数がない場合は `git diff` と `git diff --cached` から変更されたGoファイル（`_test.go` を除く）を対象とする
3. 対象ファイルがない場合はその旨を伝えて終了
4. 対象ファイルの内容を読み、テスト対象の関数・メソッドを一覧化する

### Step 2: 既存テストの確認

1. 対象ファイルに対応する `_test.go` が既に存在するか確認する
2. 存在する場合は既存のテストを読み、未テストの関数を特定する
3. 既存テストのスタイル（ヘルパー関数、モック定義等）を把握し、新規テストでも踏襲する

### Step 3: テスト生成方針の提示

テストを書く前に、以下のフォーマットで生成方針をユーザーに提示し確認を得る：

```text
## テスト生成方針

対象: internal/feature/candles/usecase/candles_usecase.go
既存テスト: あり（3関数中2関数テスト済み）

生成予定:
  - TestCandlesUsecase_NewMethod          -- 正常系3ケース、異常系2ケース
  - TestCandlesUsecase_AnotherMethod      -- 正常系2ケース、異常系1ケース

モック:
  - mockCandleRepository（既存を再利用）
```

### Step 4: テストコード生成

ユーザーの承認後、以下のルールに従ってテストコードを生成する。

#### 構造・スタイルルール

- **テーブル駆動テスト**: 複数ケースは `tests := []struct { ... }{}` + `for _, tt := range tests { t.Run() }` で記述
- **並列実行**: 関数レベルとサブテストレベルの両方で `t.Parallel()` を付与
- **サブテスト**: `t.Run(tt.name, func(t *testing.T) { ... })` を使用
- **パッケージ**: テスト対象と同じパッケージに配置（`_test` サフィックスなし）

#### モック定義ルール

- 関数型フィールドを持つ構造体でモックを定義する
- 呼び出しカウント用フィールド（`XxxCalls int`）を必要に応じて追加
- 関数フィールドが未設定の場合はデフォルト値を返す
- 既存テストファイルにモックがある場合はそれを再利用する

```go
type mockXxxRepository struct {
    FindFunc  func(ctx context.Context, id string) (*entity.Xxx, error)
    FindCalls int
}

func (m *mockXxxRepository) Find(ctx context.Context, id string) (*entity.Xxx, error) {
    m.FindCalls++
    if m.FindFunc != nil {
        return m.FindFunc(ctx, id)
    }
    return nil, nil
}
```

#### テスト関数の命名規則

- 形式: `Test<構造体名>_<メソッド名>`（例: `TestAuthUsecase_Login`）
- テストケース名（`name` フィールド）は英語で記述
- 形式: `"success: <説明>"` または `"error: <説明>"`

#### アサーション

- `github.com/stretchr/testify/assert` と `require` を使用
- 致命的なエラー（セットアップ失敗等）は `require` を使用
- 値の検証は `assert` を使用
- スライスの比較には `assert.Equal` または `reflect.DeepEqual` を使用

#### ヘルパー関数

- 既存のヘルパー関数がある場合は再利用する
- 新規ヘルパーには `t.Helper()` を必ず付与
- 命名: `setupXxx`, `seedXxx`, `assertXxx`, `makeXxx` 等

#### レイヤー別テスト戦略

| レイヤー | テスト手法 | モック対象 |
|---------|-----------|-----------|
| usecase | モック駆動テスト | リポジトリインターフェース |
| handler | `httptest` + `gin.New()` | usecaseインターフェース |
| adapters | SQLiteインメモリDB | なし（実DB使用） |
| platform | 依存に応じた手法 | Redis: `redismock`等 |

### Step 5: テストケースの網羅性確認

生成したテストが以下を網羅しているか確認する：

- **正常系**: 期待通りの入力で正しい結果が返ること
- **異常系**: エラーケース（リポジトリエラー、バリデーションエラー等）
- **境界値**: 空スライス、ゼロ値、nil入力等
- **デフォルト値**: デフォルト引数が適用されるケース（該当する場合）

### Step 6: テスト実行

テストコードの生成後、対象パッケージのテストを実行して全テストがパスすることを確認する：

```bash
go test ./<対象パッケージ>/... -v -race
```

テストが失敗した場合は原因を修正し、再度実行する。

## 注意事項

- 既存テストファイルがある場合は末尾に追記する（既存コードを変更しない）
- 既存テストファイルのモックやヘルパー関数を最大限再利用する
- テスト対象のプロダクションコードは変更しない
- テストは日本語のコメントを含めない（テストケース名も英語）
- 過度なテストケースは避け、実質的なカバレッジ向上に集中する
