package httpratelimit

import "github.com/go-redis/redismock/v9"

// ExpectAllow は redismock に対して Allow() 呼び出し時の期待値を設定するテスト用ヘルパーです。
// 外部テストパッケージが Redis を立てずに rate limiter の挙動を検証する用途に使います。
// (currentCount, totalCount) は Allow 内部の Lua スクリプトの返り値形式に従います。
func ExpectAllow(mock redismock.ClientMock, key string, currentCount, totalCount int64) {
	mock.ExpectEvalSha(scriptHash(), []string{key}, "_", "_", "_", "_", "_").
		SetVal([]interface{}{currentCount, totalCount})
}
