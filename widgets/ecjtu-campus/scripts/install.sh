#!/usr/bin/env bash
# 将「华交校园」Plasma 6 小部件与 C++ QML 模块安装到当前用户目录（~/.local），无需 root。
# 用法：
#   ./widgets/ecjtu-campus/scripts/install.sh           # 构建并安装到 ~/.local，配置 QML_IMPORT_PATH
#   ./widgets/ecjtu-campus/scripts/install.sh --reload  # 安装并同步环境后重启 plasmashell 立即生效
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
WIDGET_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
BUILD_DIR="${WIDGET_DIR}/build"
PREFIX="${HOME}/.local"
ENV_DIR="${HOME}/.config/plasma-workspace/env"
ENV_FILE="${ENV_DIR}/ecjtu-campus.sh"
DO_RELOAD=false

usage() {
    cat <<'EOF'
用法: install.sh [选项]

选项:
  --reload    安装后将环境变量同步到 systemd --user 并重启 plasma-plasmashell.service
  -h, --help  显示此帮助
EOF
}

while [[ $# -gt 0 ]]; do
    case "$1" in
        --reload)
            DO_RELOAD=true
            shift
            ;;
        -h|--help)
            usage
            exit 0
            ;;
        *)
            echo "未知参数: $1" >&2
            usage >&2
            exit 2
            ;;
    esac
done

if ! command -v cmake >/dev/null 2>&1; then
    echo "错误: 未找到 cmake，请先安装 cmake。" >&2
    exit 1
fi

GEN_ARGS=()
if command -v ninja >/dev/null 2>&1; then
    GEN_ARGS=(-G Ninja)
fi

echo "==> [1/3] 编译 Release 版小部件与 Qt 6 GLSL 着色器..."
cmake -S "${WIDGET_DIR}" -B "${BUILD_DIR}" "${GEN_ARGS[@]}" -DCMAKE_BUILD_TYPE=Release
cmake --build "${BUILD_DIR}"

echo "==> [2/3] 安装小部件与 C++ QML 模块到 ${PREFIX}..."
cmake --install "${BUILD_DIR}" --prefix "${PREFIX}"

echo "==> [3/3] 配置 Plasma 会话环境脚本 (${ENV_FILE})..."
mkdir -p "${ENV_DIR}"
if [[ ! -f "${ENV_FILE}" ]]; then
    cat <<'EOF' > "${ENV_FILE}"
# 由 widgets/ecjtu-campus/scripts/install.sh 生成
export QML_IMPORT_PATH="$HOME/.local/lib/qml${QML_IMPORT_PATH:+:$QML_IMPORT_PATH}"
case ":$PATH:" in
    *":$HOME/.local/bin:"*) ;;
    *) export PATH="$HOME/.local/bin:$PATH" ;;
esac

# 如尚未在其他桌面环境文件中配置凭证，可取消下一行注释并填入你的教务绑定 OpenID：
# export ECJTU_WEIXIN_TOKEN='你的OpenID'
EOF
    chmod 600 "${ENV_FILE}"
    echo "已创建 ${ENV_FILE}（权限 600）。"
else
    echo "保留已有 ${ENV_FILE}（未覆盖）。"
fi

if [[ "${DO_RELOAD}" == true ]]; then
    echo "==> 同步会话环境并重启 plasma-plasmashell.service..."
    # shellcheck disable=SC1090
    source "${ENV_FILE}"
    IMPORT_VARS=(QML_IMPORT_PATH PATH)
    if [[ -n "${ECJTU_WEIXIN_TOKEN:-}" ]]; then
        IMPORT_VARS+=(ECJTU_WEIXIN_TOKEN)
    fi
    systemctl --user import-environment "${IMPORT_VARS[@]}"
    systemctl --user restart plasma-plasmashell.service
    echo "完成！现在可在「添加或管理小部件」中搜索「华交校园」。"
else
    cat <<EOF

安装完成：
  - 小部件包:   ${PREFIX}/share/plasma/plasmoids/io.github.mochenya.ecjtucampus
  - C++ 插件:   ${PREFIX}/lib/qml/io/github/mochenya/ecjtucampus/backend
  - 环境脚本:   ${ENV_FILE}

下一步让 Plasma 识别新插件（任选其一）：
  1. 注销并重新登录 KDE 会话（推荐）；
  2. 或运行以下命令立即刷新面板：
     ${SCRIPT_DIR}/install.sh --reload

注意：若你的桌面会话尚未配置 ECJTU_WEIXIN_TOKEN，请编辑 ${ENV_FILE} 取消注释并填入后生效。
EOF
fi
