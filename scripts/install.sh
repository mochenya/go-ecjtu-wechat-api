#!/bin/sh
# 安装或更新当前用户的 ecjtu。
set -eu

repo=https://github.com/mochenya/go-ecjtu-wechat-api
install_dir=${ECJTU_INSTALL_DIR:-"$HOME/.local/bin"}

fail() {
    printf '错误：%s\n' "$*" >&2
    exit 1
}

for command in curl tar mktemp awk; do
    command -v "$command" >/dev/null 2>&1 || fail "缺少 $command，请先安装。"
done
if command -v sha256sum >/dev/null 2>&1; then
    hash_command=sha256sum
elif command -v shasum >/dev/null 2>&1; then
    hash_command=shasum
else
    fail '缺少 sha256sum 或 shasum，无法校验下载文件。'
fi

case $(uname -s) in
    Linux) os=linux ;;
    Darwin) os=darwin ;;
    *) fail '仅支持 Linux 和 macOS。' ;;
esac
case $(uname -m) in
    x86_64|amd64) arch=amd64 ;;
    aarch64|arm64) arch=arm64 ;;
    *) fail '不支持当前处理器架构。' ;;
esac

latest=$(curl -fsSL -o /dev/null -w '%{url_effective}' "$repo/releases/latest") || fail '无法查询最新 Release。'
version=${latest##*/}
case "$latest" in
    "$repo"/releases/tag/*) ;;
    *) fail '无法识别最新 Release 地址。' ;;
esac
printf '%s\n' "$version" | awk '/^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$/ { ok=1 } END { exit !ok }' || fail '最新 Release 版本格式不受支持。'

[ ! -d "$install_dir/ecjtu" ] || fail '安装目标是目录，无法替换。'
if [ -e "$install_dir/ecjtu" ]; then
    installed_version=
    if [ -x "$install_dir/ecjtu" ]; then
        installed_version=$("$install_dir/ecjtu" --version 2>/dev/null | awk 'NR == 1 && $1 == "ecjtu" { print $2; exit }')
    fi
    if [ "$installed_version" = "$version" ]; then
        printf '已是最新版本 %s：%s/ecjtu\n' "$version" "$install_dir"
        exit 0
    fi
    printf '正在更新 ecjtu（当前版本：%s，最新版本：%s）。\n' "${installed_version:-未知}" "$version"
else
    printf '正在安装 ecjtu %s。\n' "$version"
fi

name="ecjtu_${version}_${os}_${arch}"
archive="$name.tar.gz"
temp_dir=$(mktemp -d) || fail '无法创建临时目录。'
temp_binary=
cleanup() {
    [ -z "$temp_binary" ] || rm -f "$temp_binary"
    rm -rf "$temp_dir"
}
trap cleanup 0
trap 'exit 1' HUP INT TERM

printf '正在下载 %s（%s/%s）…\n' "$version" "$os" "$arch"
curl -fsSL "$repo/releases/download/$version/$archive" -o "$temp_dir/$archive" || fail '下载程序失败。'
curl -fsSL "$repo/releases/download/$version/checksums.txt" -o "$temp_dir/checksums.txt" || fail '下载校验文件失败。'
expected=$(awk -v name="./$archive" '$2 == name { print $1; count++ } END { if (count != 1) exit 1 }' "$temp_dir/checksums.txt") || fail '校验文件中没有唯一匹配的压缩包。'
case "$hash_command" in
    sha256sum) actual=$(sha256sum "$temp_dir/$archive") ;;
    shasum) actual=$(shasum -a 256 "$temp_dir/$archive") ;;
esac
actual=${actual%% *}
[ "$actual" = "$expected" ] || fail 'SHA-256 校验失败，未安装。'

tar -xzf "$temp_dir/$archive" -C "$temp_dir" "$name/ecjtu" || fail '解压程序失败。'
mkdir -p "$install_dir" || fail '无法创建安装目录。'
temp_binary=$(mktemp "$install_dir/.ecjtu.XXXXXX") || fail '无法在安装目录创建临时文件。'
cat "$temp_dir/$name/ecjtu" > "$temp_binary"
chmod 755 "$temp_binary"
mv -f "$temp_binary" "$install_dir/ecjtu"
temp_binary=
printf '已安装 %s 至 %s/ecjtu\n' "$version" "$install_dir"
case ":$PATH:" in
    *":$install_dir:"*) ;;
    *) printf '请将 %s 加入 PATH，然后重新打开终端。\n' "$install_dir" ;;
esac
