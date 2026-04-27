# Docker
APP_ENV=docker

# ログレベル（任意。DEBUG を指定した場合のみ slog のレベルを下げる。未設定時は INFO 相当）
# LOG_LEVEL=DEBUG

# DB
DB_HOST=db
DB_PORT=5432
DB_USER=appuser
DB_PASSWORD=apppass
DB_NAME=app
RUN_MIGRATIONS=true

# GORM ログレベル（任意。info / warn / error のいずれか。未設定・不明値は Silent）
# DB_LOG_LEVEL=warn

# CORS（許可するオリジン、カンマ区切りで複数指定可。未設定時は http://localhost:3000）
CORS_ALLOWED_ORIGINS=http://localhost:3000

# JWT
JWT_SECRET=your_jwt_secret_here

# Cookie Secure フラグ（本番環境では true に変更すること）
# true: HTTPS のみで Cookie を送信（本番必須）
# false: HTTP でも Cookie を送信（ローカル開発用）
COOKIE_SECURE=false

# Password Pepper（パスワードハッシュ用ペッパー）
PASSWORD_PEPPER=your_password_pepper_here

# twelvedata
TWELVE_DATA_API_KEY=your_twelvedata_api_key_here
TWELVE_DATA_BASE_URL=https://api.twelvedata.com

# Ingest バッチのタイムアウト時間（任意。正の整数。未設定時は 3 時間）
# INGEST_TIMEOUT_HOURS=3

# Ingest バッチの許容失敗率（任意。0.0〜1.0 の浮動小数。未設定時は 0.2）
# 銘柄単位の失敗率がこの値を超えた場合、ingest プロセスは exit 1 で終了する。
# INGEST_MAX_FAILURE_RATE=0.2

# Redis
REDIS_HOST=redis
REDIS_PORT=6379
REDIS_PASSWORD=

# Google Cloud (ロゴ検出・企業分析機能)
GOOGLE_GENAI_USE_VERTEXAI=true
GOOGLE_CLOUD_PROJECT=your_gcp_project_id
GOOGLE_CLOUD_LOCATION=asia-northeast1