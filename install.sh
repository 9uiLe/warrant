#!/bin/sh
# warrant のインストールスクリプト
# 使い方:
#   curl -fsSL https://raw.githubusercontent.com/9uiLe/warrant/master/install.sh | sh
#
# 環境変数:
#   WARRANT_VERSION    インストールするバージョン（既定: 最新リリース）
#   WARRANT_INSTALL_DIR インストール先ディレクトリ（既定: $HOME/.local/bin）
#
# 注意: このスクリプトはリポジトリが public の場合にのみ動作する。
# private 中は go build 経路を使うこと。

set -eu

REPO="9uiLe/warrant"
INSTALL_DIR="${WARRANT_INSTALL_DIR:-${HOME}/.local/bin}"

# 一時ディレクトリを作成し、スクリプト終了時に掃除する
TMP_DIR=""
cleanup() {
  if [ -n "$TMP_DIR" ] && [ -d "$TMP_DIR" ]; then
    rm -rf "$TMP_DIR"
  fi
}
trap cleanup EXIT INT TERM

TMP_DIR="$(mktemp -d)"

# --- OS 検出 ---
detect_os() {
  case "$(uname -s)" in
    Darwin) printf "darwin" ;;
    Linux)  printf "linux" ;;
    *)
      printf 'エラー: サポートされていない OS: %s\n' "$(uname -s)" >&2
      printf 'Go 保有者は以下の経路でビルドしてください:\n' >&2
      printf '  git clone git@github.com:%s.git && cd warrant && CGO_ENABLED=0 go build -o warrant . && mv warrant /usr/local/bin/\n' "$REPO" >&2
      exit 1
      ;;
  esac
}

# --- ARCH 検出 ---
detect_arch() {
  case "$(uname -m)" in
    x86_64|amd64) printf "amd64" ;;
    arm64|aarch64) printf "arm64" ;;
    *)
      printf 'エラー: サポートされていないアーキテクチャ: %s\n' "$(uname -m)" >&2
      printf 'Go 保有者は以下の経路でビルドしてください:\n' >&2
      printf '  git clone git@github.com:%s.git && cd warrant && CGO_ENABLED=0 go build -o warrant . && mv warrant /usr/local/bin/\n' "$REPO" >&2
      exit 1
      ;;
  esac
}

# --- ダウンロード関数 (curl / wget フォールバック) ---
download() {
  url="$1"
  dest="$2"
  if command -v curl > /dev/null 2>&1; then
    curl -fsSL --retry 3 -o "$dest" "$url"
  elif command -v wget > /dev/null 2>&1; then
    wget -q -O "$dest" "$url"
  else
    printf 'エラー: curl または wget が必要です\n' >&2
    exit 1
  fi
}

# --- バージョン解決 ---
resolve_version() {
  # "latest" は空指定と同じく最新リリースとして扱う（GitHub Action の version: latest 契約）
  if [ -n "${WARRANT_VERSION:-}" ] && [ "$WARRANT_VERSION" != "latest" ]; then
    printf '%s' "$WARRANT_VERSION"
    return
  fi
  api_url="https://api.github.com/repos/${REPO}/releases/latest"
  tmp_json="${TMP_DIR}/latest.json"
  if ! download "$api_url" "$tmp_json" 2>/dev/null; then
    printf 'エラー: バージョンの解決に失敗しました。%s が public かどうか確認してください。\n' "$REPO" >&2
    printf 'WARRANT_VERSION に明示的なタグを指定して再試行することもできます。\n' >&2
    exit 1
  fi
  # jq 非依存で tag_name を抽出（"tag_name": "v1.2.3" 形式）
  tag="$(grep '"tag_name"' "$tmp_json" | sed 's/.*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/')"
  if [ -z "$tag" ]; then
    printf 'エラー: tag_name の解析に失敗しました\n' >&2
    exit 1
  fi
  printf '%s' "$tag"
}

# --- SHA256 ハッシュ計算コマンドを選択 ---
sha256_of() {
  file="$1"
  if command -v sha256sum > /dev/null 2>&1; then
    sha256sum "$file" | awk '{print $1}'
  elif command -v shasum > /dev/null 2>&1; then
    shasum -a 256 "$file" | awk '{print $1}'
  else
    printf 'エラー: SHA256 の計算に sha256sum または shasum が必要です\n' >&2
    exit 1
  fi
}

# ----- メイン処理 -----

OS="$(detect_os)"
ARCH="$(detect_arch)"
VERSION="$(resolve_version)"

ASSET="warrant_${OS}_${ARCH}"
BASE_URL="https://github.com/${REPO}/releases/download/${VERSION}"
ASSET_URL="${BASE_URL}/${ASSET}"
CHECKSUMS_URL="${BASE_URL}/checksums.txt"

printf 'warrant %s (%s/%s) をインストールします...\n' "$VERSION" "$OS" "$ARCH"

TMP_BINARY="${TMP_DIR}/${ASSET}"
TMP_CHECKSUMS="${TMP_DIR}/checksums.txt"

# バイナリと checksums.txt をダウンロード
if ! download "$ASSET_URL" "$TMP_BINARY"; then
  printf 'エラー: バイナリのダウンロードに失敗しました: %s\n' "$ASSET_URL" >&2
  printf 'リポジトリが public かどうか、WARRANT_VERSION が正しいタグかどうかを確認してください。\n' >&2
  printf 'private 中は go build 経路を使うこと:\n' >&2
  printf '  git clone git@github.com:%s.git && cd warrant && CGO_ENABLED=0 go build -o warrant .\n' "$REPO" >&2
  exit 1
fi

if ! download "$CHECKSUMS_URL" "$TMP_CHECKSUMS"; then
  printf 'エラー: checksums.txt のダウンロードに失敗しました\n' >&2
  exit 1
fi

# SHA256 検証（fail closed: 不一致なら即終了）
EXPECTED="$(grep "  ${ASSET}$" "$TMP_CHECKSUMS" | awk '{print $1}')"
if [ -z "$EXPECTED" ]; then
  printf 'エラー: checksums.txt に %s のエントリが見つかりません\n' "$ASSET" >&2
  exit 1
fi

ACTUAL="$(sha256_of "$TMP_BINARY")"

if [ "$ACTUAL" != "$EXPECTED" ]; then
  printf 'エラー: SHA256 の検証に失敗しました\n' >&2
  printf '  期待値: %s\n' "$EXPECTED" >&2
  printf '  実際値: %s\n' "$ACTUAL" >&2
  printf '一時ファイルを削除して終了します\n' >&2
  exit 1
fi

printf 'SHA256 検証: OK\n'

# インストール
mkdir -p "$INSTALL_DIR"
chmod +x "$TMP_BINARY"
mv "$TMP_BINARY" "${INSTALL_DIR}/warrant"

printf 'インストール完了: %s/warrant\n' "$INSTALL_DIR"

# PATH 確認
case ":${PATH}:" in
  *":${INSTALL_DIR}:"*)
    ;;
  *)
    printf '\n注意: %s が PATH に含まれていません。\n' "$INSTALL_DIR" >&2
    printf '以下をシェルの設定ファイル（~/.bashrc, ~/.zshrc 等）に追記してください:\n' >&2
    printf '  export PATH="%s:$PATH"\n' "$INSTALL_DIR" >&2
    ;;
esac

printf '\n次のステップ:\n'
printf '  warrant version   # バージョン確認\n'
printf '  warrant init      # 利用したいプロジェクトの root で実行\n'
