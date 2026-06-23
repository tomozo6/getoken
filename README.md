<p align="center">
  Firebase Identity Toolkit から `idToken` を素早く取得するための Go 製 CLI
</p>

<p align="center">
  <a href="https://github.com/tomozo6/getoken/blob/main/LICENSE"><img src="https://img.shields.io/github/license/tomozo6/getoken" alt="license"></a>
  <a href="https://go.dev/"><img src="https://img.shields.io/badge/Go-1.26%2B-00ADD8?logo=go" alt="go"></a>
  <img src="https://img.shields.io/badge/Firebase-Identity%20Toolkit-FFCA28?logo=firebase&logoColor=black" alt="firebase">
  <img src="https://img.shields.io/badge/Mode-CLI-2F855A" alt="cli">
</p>

# getoken

## 📖 getokenとは

`getoken` は Firebase Identity Toolkit の `accounts:signInWithPassword` を呼び出して、指定したユーザーの `idToken` を取得する CLI です。

- 環境ごとの `apikey`, `email`, `password` を YAML で管理
- `default_env` を持たせて、引数なし実行に対応
- カレントディレクトリまたはホームディレクトリの設定ファイルを自動探索
- 成功時は `idToken` だけを標準出力へ出すので、他コマンドにそのままつなぎやすい

## ⚡ クイックスタート

```bash
brew install tomozo6/tap/getoken
cp .getoken.yaml.example ~/.getoken.yaml
getoken
```

設定ファイルをホームディレクトリに置いておくと、どのディレクトリからでもそのまま実行できます。

## 🛠️ 使い方

### デフォルト環境で `idToken` を取得する

`default_env` が設定されていれば、引数なしでその環境を使います。

```bash
getoken
```

出力は `idToken` のみです。

```text
eyJhbGciOiJSUzI1NiIsImtpZCI6...
```

### 環境を明示して取得する

`stg` や `dev` を使いたい場合は `--env` を指定します。

```bash
getoken --env stg
getoken -e dev
```

### 別の設定ファイルを使う

既定の `.getoken.yaml` 以外を使う場合は `--config` を指定します。

```bash
getoken --config /path/to/getoken.yaml
getoken -c /path/to/getoken.yaml -e prd
```

### シェル変数に入れて後続処理に使う

標準出力がトークンだけなので、そのまま変数代入できます。

```bash
ID_TOKEN="$(getoken --env stg)"
```

## 🎛️ フラグ

| フラグ     | 短縮 | 説明                                            |
| ---------- | ---- | ----------------------------------------------- |
| `--config` | `-c` | 設定 YAML のパス。デフォルトは `.getoken.yaml`  |
| `--env`    | `-e` | 利用する環境名。未指定時は `default_env` を利用 |

## 🧾 設定ファイル

デフォルトでは `.getoken.yaml` を読み込みます。

- まずカレントディレクトリの `.getoken.yaml` を探す
- 見つからなければ `~/.getoken.yaml` を探す
- `--config` を指定した場合は、そのパスを優先する

サンプルは [.getoken.yaml.example](/Users/tomohiro_sasaki/github.com/tomozo6/getoken/.getoken.yaml.example:1) にあります。

Homebrew で入れて常用する場合は `~/.getoken.yaml` に置く運用が扱いやすいです。

```yaml
default_env: prd

envs:
  prd:
    apikey: xxxxxxxxxxxxxxxxxxxxxxx
    email: user@example.com
    password: prd-password

  stg:
    apikey: xxxxxxxxxxxxxxxxxxxxxxx
    email: user-stg@example.com
    password: stg-password

  dev:
    apikey: xxxxxxxxxxxxxxxxxxxxxxx
    email: user-dev@example.com
    password: dev-password
```

### 各項目

- `default_env`: `--env` 未指定時に使う環境名
- `envs`: 環境ごとの認証情報
- `apikey`: Firebase Web API Key
- `email`: `accounts:signInWithPassword` に渡すメールアドレス
- `password`: 対象ユーザーのパスワード

## 🧭 前提

- Firebase Identity Toolkit の `accounts:signInWithPassword` を利用できること
- 対象ユーザーがメールアドレス + パスワード認証でサインイン可能であること
- 実行環境から `https://identitytoolkit.googleapis.com/` へ到達できること

## 📤 出力と終了条件

- 成功時は `idToken` のみを標準出力へ出す
- 失敗時は標準エラー出力にエラーメッセージを出して終了する

主な失敗条件:

- 設定ファイルが見つからない
- `default_env` も `--env` も未指定
- 指定環境に `apikey`, `email`, `password` のいずれかが不足している
- Firebase から認証エラーが返る

## 🔐 セキュリティメモ

- `.getoken.yaml` には平文の認証情報が入るため、Git 管理には含めない運用を前提にしてください
- リポジトリにはサンプルとして `.getoken.yaml.example` のみを置く想定です
