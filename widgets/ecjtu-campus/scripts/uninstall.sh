#!/usr/bin/env bash
# 卸载用户级安装的「华交校园」Plasma 6 小部件与 C++ QML 模块。
# 用法：
#   ./widgets/ecjtu-campus/scripts/uninstall.sh            # 卸载小部件与插件（保留环境脚本以防误删凭证）
#   ./widgets/ecjtu-campus/scripts/uninstall.sh --purge-env # 连同 ~/.config/plasma-workspace/env/ecjtu-campus.sh 一并删除
set -euo pipefail

PREFIX="${HOME}/.local"
PLASMOID_DIR="${PREFIX}/share/plasma/plasmoids/io.github.mochenya.ecjtucampus"
QML_PLUGIN_DIR="${PREFIX}/lib/qml/io/github/mochenya/ecjtucampus"
ENV_FILE="${HOME}/.config/plasma-workspace/env/ecjtu-campus.sh"
PURGE_ENV=false

while [[ $# -gt 0 ]]; do
    case "$1" in
        --purge-env)
            PURGE_ENV=true
            shift
            ;;
        -h|--help)
            echo "用法: uninstall.sh [--purge-env]"
            exit 0
            ;;
        *)
            echo "未知参数: $1" >&2
            exit 2
            ;;
    esac
done

rm -rf "${PLASMOID_DIR}" "${QML_PLUGIN_DIR}"
echo "已移除小部件包与 C++ QML 插件。"

if [[ "${PURGE_ENV}" == true ]]; then
    rm -f "${ENV_FILE}"
    echo "已移除环境脚本 ${ENV_FILE}。"
elif [[ -f "${ENV_FILE}" ]]; then
    echo "已保留 ${ENV_FILE}（如需一并删除请加 --purge-env）。"
fi

echo "请先从桌面/面板移除旧实例，或重新登录 / 重启 plasma-plasmashell.service。"
