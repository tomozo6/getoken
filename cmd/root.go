package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// credentials は Firebase Identity Toolkit でサインインするために
// 環境ごとに必要な値をまとめたものです。
type credentials struct {
	APIKey   string `json:"apikey"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

// config は .getoken.yaml の内容を表し、環境名と認証情報を対応付けます。
type config struct {
	DefaultEnv string                 `yaml:"default_env"`
	Envs       map[string]credentials `yaml:"envs"`
}

// signInRequest は accounts:signInWithPassword に送る JSON ペイロードです。
type signInRequest struct {
	Email             string `json:"email"`
	Password          string `json:"password"`
	ReturnSecureToken bool   `json:"returnSecureToken"`
}

// signInResponse はこの CLI が使うレスポンス項目だけを持ちます。
// 取得した idToken と API の構造化エラーメッセージを受け取ります。
type signInResponse struct {
	IDToken string `json:"idToken"`
	Error   struct {
		Message string `json:"message"`
	} `json:"error"`
}

// これらの値は Cobra がフラグ解析後に設定します。
var configPath string
var envName string

// Version は開発時の既定値です。リリースビルドでは ldflags で上書きします。
var Version = "dev"

var rootCmd = &cobra.Command{
	Use:     "getoken",
	Short:   "Fetch an ID token from Firebase Identity Toolkit",
	Version: Version,
	RunE: func(cmd *cobra.Command, args []string) error {
		// 先に認証情報を解決しておくことで、設定不備や検証エラーを
		// ネットワークアクセス前に返せるようにします。
		creds, err := loadCredentials(configPath, envName)
		if err != nil {
			return err
		}

		// コマンドの context を使い、Ctrl+C などのキャンセルが
		// HTTP リクエストにも伝わるようにします。
		token, err := fetchIDToken(cmd.Context(), creds)
		if err != nil {
			return err
		}

		// 標準出力にはトークンだけを出し、コマンド置換やパイプで
		// そのまま使いやすくします。
		fmt.Fprintln(cmd.OutOrStdout(), token)
		return nil
	},
}

func init() {
	// デフォルトではカレントディレクトリの .getoken.yaml を見て、
	// 無ければ ~/.getoken.yaml へフォールバックします。
	rootCmd.Flags().StringVarP(&configPath, "config", "c", ".getoken.yaml", "path to config yaml")
	// 利用環境は config.envs から 1 件選びます。未指定なら
	// 設定ファイルの default_env を使います。
	rootCmd.Flags().StringVarP(&envName, "env", "e", "", "environment name in config yaml")
}

// Execute は main.go から呼ばれる CLI の実行入口です。
func Execute() error {
	return rootCmd.Execute()
}

// loadCredentials は設定ファイルの解決、YAML の読み込み、環境選択、
// 必須項目の検証までをまとめて行います。
func loadCredentials(path string, selectedEnv string) (credentials, error) {
	resolvedPath, err := resolveConfigPath(path)
	if err != nil {
		return credentials{}, err
	}

	raw, err := os.ReadFile(resolvedPath)
	if err != nil {
		return credentials{}, fmt.Errorf("read credentials file: %w", err)
	}

	var cfg config
	if err := yaml.Unmarshal(raw, &cfg); err != nil {
		return credentials{}, fmt.Errorf("parse credentials file: %w", err)
	}

	if len(cfg.Envs) == 0 {
		return credentials{}, fmt.Errorf("credentials file must include at least one env under envs")
	}

	// CLI フラグが明示されていればそれを優先し、無ければ default_env を使います。
	env := selectedEnv
	if env == "" {
		env = cfg.DefaultEnv
	}
	if env == "" {
		return credentials{}, fmt.Errorf("environment was not specified and default_env is empty")
	}

	creds, ok := cfg.Envs[env]
	if !ok {
		return credentials{}, fmt.Errorf("env %q was not found in config", env)
	}

	if creds.APIKey == "" || creds.Email == "" || creds.Password == "" {
		return credentials{}, fmt.Errorf("env %q must include apikey, email, and password", env)
	}

	return creds, nil
}

// resolveConfigPath は明示指定されたパスをそのまま使います。
// デフォルト名の場合だけ、カレントディレクトリを先に見て、
// 次にホームディレクトリを探す順序を実装します。
func resolveConfigPath(path string) (string, error) {
	if path != ".getoken.yaml" {
		return path, nil
	}

	// 作業ディレクトリに設定があれば、それを最優先で使います。
	if _, err := os.Stat(path); err == nil {
		return path, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("stat config file: %w", err)
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home directory: %w", err)
	}

	homeConfigPath := filepath.Join(homeDir, ".getoken.yaml")
	if _, err := os.Stat(homeConfigPath); err == nil {
		return homeConfigPath, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("stat config file: %w", err)
	}

	return "", fmt.Errorf("config file .getoken.yaml was not found in the current directory or home directory")
}

// fetchIDToken はサインイン API を呼び出し、idToken だけを返します。
func fetchIDToken(ctx context.Context, creds credentials) (string, error) {
	payload := signInRequest{
		Email:             creds.Email,
		Password:          creds.Password,
		ReturnSecureToken: true,
	}

	// POST ボディへそのまま渡せるように、先に JSON へ変換します。
	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal request body: %w", err)
	}

	url := fmt.Sprintf(
		"https://identitytoolkit.googleapis.com/v1/accounts:signInWithPassword?key=%s",
		creds.APIKey,
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// 認証先の応答待ちが無限に伸びないよう、タイムアウトを固定します。
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response body: %w", err)
	}

	var signInResp signInResponse
	if err := json.Unmarshal(respBody, &signInResp); err != nil {
		return "", fmt.Errorf("parse response body: %w", err)
	}

	// Firebase は非 200 でも構造化エラーを返すことがあるため、
	// 取得できた場合はそのメッセージを優先して返します。
	if resp.StatusCode != http.StatusOK {
		if signInResp.Error.Message != "" {
			return "", fmt.Errorf("identity toolkit error: %s", signInResp.Error.Message)
		}
		return "", fmt.Errorf("identity toolkit returned status %s", resp.Status)
	}

	// HTTP 200 でも idToken が空なら、アプリケーションエラーとして扱います。
	if signInResp.IDToken == "" {
		return "", fmt.Errorf("idToken was empty in response")
	}

	return signInResp.IDToken, nil
}
