package editor

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"github.com/axsh/tokotachi/pkg/detect"
	"github.com/axsh/tokotachi/pkg/matrix"
)

// ArgsConfig defines structured argument templates for different editor launch situations.
type ArgsConfig struct {
	Default      []string `yaml:"default"`
	NewWindow    []string `yaml:"new_window,omitempty"`
	Devcontainer []string `yaml:"devcontainer,omitempty"`
}

// PlatformConfig defines platform-specific overrides.
type PlatformConfig struct {
	Cmd  *string     `yaml:"cmd,omitempty"`
	Type *string     `yaml:"type,omitempty"`
	Args *ArgsConfig `yaml:"args,omitempty"`
}

// EditorConfig defines the launch settings for a specific editor.
type EditorConfig struct {
	Cmd     string          `yaml:"cmd"`
	Type    string          `yaml:"type"`
	Args    ArgsConfig      `yaml:"args"`
	Windows *PlatformConfig `yaml:"windows,omitempty"`
	Darwin  *PlatformConfig `yaml:"darwin,omitempty"`
	Linux   *PlatformConfig `yaml:"linux,omitempty"`
}

// Config represents the schema of editor.yaml.
type Config struct {
	System struct {
		Editors map[string]EditorConfig `yaml:"editors"`
	} `yaml:"system"`
	User struct {
		Editors map[string]EditorConfig `yaml:"editors"`
	} `yaml:"user"`
}

// ResolveEditor resolves the config for the given editor name, 
// prioritizing user configuration over system configuration.
func (c *Config) ResolveEditor(name string) (EditorConfig, bool) {
	if ec, ok := c.User.Editors[name]; ok {
		return ec, true
	}
	if ec, ok := c.System.Editors[name]; ok {
		return ec, true
	}
	return EditorConfig{}, false
}

const defaultYAMLTemplate = `# tokotachi editor settings
#
# この設定ファイルはエディタの起動コマンドや引数をカスタマイズするために使用します。
# 
# セクション説明:
# - system: tt コマンドが自動管理するセクションです。アップデートにより上書きされる可能性があります。
# - user:   ユーザーが任意の設定を記述するセクションです。tt が勝手に書き換えることはありません。
#           system セクションと同名のエディタ設定が存在する場合、user セクションが優先されます。
# 
# 各エディタの設定項目:
# - cmd:               基本の起動コマンド名、または絶対パス
# - type:              起動タイプ。以下から選択します:
#                      "vscode" -> VSCode系。Dev Containerアタッチや --new-window などの引数をサポート。
#                      "local"  -> ローカルフォルダオープン。
#                      "cli"    -> Claude Codeなどの対話型CLI。
# - args:              起動状況ごとの引数の階層設定。以下のサブキーを持ちます:
#                      default:      通常のローカル起動時に渡す引数リスト
#                      new_window:   新規ウィンドウ起動時に使用する引数リスト（任意）
#                      devcontainer: Dev Containerアタッチ時に使用する引数リスト（任意）
# 
# OS別の固有オーバーライド設定 (windows / darwin / linux):
#   上記の設定項目 (cmd, type, args) は、各 OS ブロックの子要素として個別に定義することでオーバーライドが可能です。
# 
# 引数プレースホルダー (args 内で利用可能):
# - {path}:      ローカルのワークツリー絶対パス
# - {container}: 対象のコンテナ名
# - {uri}:       VSCode/Cursor 等の Dev Container リモートURI
#

system:
  editors:
    code:
      cmd: "code"
      type: "vscode"
      args:
        default: ["{path}"]
        new_window: ["--new-window", "{path}"]
        devcontainer: ["--folder-uri", "{uri}"]
      windows:
        cmd: "code"
      darwin:
        cmd: "code"
      linux:
        cmd: "code"
    cursor:
      cmd: "cursor"
      type: "vscode"
      args:
        default: ["{path}"]
        new_window: ["--new-window", "{path}"]
        devcontainer: ["--folder-uri", "{uri}"]
      windows:
        cmd: "cursor"
      darwin:
        cmd: "cursor"
      linux:
        cmd: "cursor"
    ag:
      cmd: "antigravity"
      type: "vscode"
      args:
        default: ["{path}"]
        new_window: ["--new-window", "{path}"]
        devcontainer: ["--folder-uri", "{uri}"]
      windows:
        cmd: "antigravity-ide.cmd"
      darwin:
        cmd: "antigravity"
      linux:
        cmd: "antigravity"
    claude:
      cmd: "claude"
      type: "local"
      args:
        default: ["{path}"]
      windows:
        cmd: "claude"
      darwin:
        cmd: "claude"
      linux:
        cmd: "claude"
user:
  editors: {}
`

// ptr is a helper that returns a pointer to the given string.
func ptr(s string) *string {
	return &s
}

// defaultConfig returns the default configuration constructed as Go structs.
func defaultConfig() *Config {
	cfg := &Config{}
	cfg.System.Editors = make(map[string]EditorConfig)

	// code
	cfg.System.Editors["code"] = EditorConfig{
		Cmd:  "code",
		Type: "vscode",
		Args: ArgsConfig{
			Default:      []string{"{path}"},
			NewWindow:    []string{"--new-window", "{path}"},
			Devcontainer: []string{"--folder-uri", "{uri}"},
		},
		Windows: &PlatformConfig{Cmd: ptr("code")},
		Darwin:  &PlatformConfig{Cmd: ptr("code")},
		Linux:   &PlatformConfig{Cmd: ptr("code")},
	}

	// cursor
	cfg.System.Editors["cursor"] = EditorConfig{
		Cmd:  "cursor",
		Type: "vscode",
		Args: ArgsConfig{
			Default:      []string{"{path}"},
			NewWindow:    []string{"--new-window", "{path}"},
			Devcontainer: []string{"--folder-uri", "{uri}"},
		},
		Windows: &PlatformConfig{Cmd: ptr("cursor")},
		Darwin:  &PlatformConfig{Cmd: ptr("cursor")},
		Linux:   &PlatformConfig{Cmd: ptr("cursor")},
	}

	// ag
	cfg.System.Editors["ag"] = EditorConfig{
		Cmd:  "antigravity",
		Type: "vscode",
		Args: ArgsConfig{
			Default:      []string{"{path}"},
			NewWindow:    []string{"--new-window", "{path}"},
			Devcontainer: []string{"--folder-uri", "{uri}"},
		},
		Windows: &PlatformConfig{Cmd: ptr("antigravity-ide.cmd")},
		Darwin:  &PlatformConfig{Cmd: ptr("antigravity")},
		Linux:   &PlatformConfig{Cmd: ptr("antigravity")},
	}

	// claude
	cfg.System.Editors["claude"] = EditorConfig{
		Cmd:  "claude",
		Type: "local",
		Args: ArgsConfig{
			Default: []string{"{path}"},
		},
		Windows: &PlatformConfig{Cmd: ptr("claude")},
		Darwin:  &PlatformConfig{Cmd: ptr("claude")},
		Linux:   &PlatformConfig{Cmd: ptr("claude")},
	}

	cfg.User.Editors = make(map[string]EditorConfig)
	return cfg
}

// LoadConfig loads the configuration from editor.yaml in the user's home directory.
// If the file does not exist, it will be automatically created with default settings.
// On load errors, it returns the built-in fallback configuration and the error.
func LoadConfig() (*Config, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return defaultConfig(), fmt.Errorf("failed to get user home dir: %w", err)
	}

	configDir := filepath.Join(home, ".kotoshiro", "tokotachi")
	configPath := filepath.Join(configDir, "editor.yaml")

	// Auto-generate if not exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		if err := os.MkdirAll(configDir, 0755); err != nil {
			return defaultConfig(), fmt.Errorf("failed to create config directory %s: %w", configDir, err)
		}
		if err := ioutil.WriteFile(configPath, []byte(defaultYAMLTemplate), 0644); err != nil {
			return defaultConfig(), fmt.Errorf("failed to write default editor.yaml: %w", err)
		}
	}

	data, err := ioutil.ReadFile(configPath)
	if err != nil {
		return defaultConfig(), fmt.Errorf("failed to read editor.yaml: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return defaultConfig(), fmt.Errorf("failed to parse editor.yaml: %w", err)
	}

	if cfg.System.Editors == nil {
		cfg.System.Editors = make(map[string]EditorConfig)
	}
	if cfg.User.Editors == nil {
		cfg.User.Editors = make(map[string]EditorConfig)
	}

	return &cfg, nil
}

// Initialize matrix type resolver.
func init() {
	matrix.EditorTypeResolver = func(ed detect.Editor) string {
		cfg, err := LoadConfig()
		if err != nil {
			cfg = defaultConfig()
		}
		if ec, ok := cfg.ResolveEditor(string(ed)); ok {
			return ec.Type
		}
		return ""
	}
}
