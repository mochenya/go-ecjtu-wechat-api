#!/usr/bin/env bash
# 启动本地独立 qml6 预览窗口（使用脱敏模拟 ecjtu CLI，不修改桌面配置，不访问学校服务器）
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
WIDGET_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
BUILD_DIR="${WIDGET_DIR}/build"

if [[ ! -d "${BUILD_DIR}/qml" || ! -d "${BUILD_DIR}/package/contents/ui/shaders" ]]; then
    echo "请先构建原型：" >&2
    echo "  cmake -S \"${WIDGET_DIR}\" -B \"${BUILD_DIR}\" -G Ninja" >&2
    echo "  cmake --build \"${BUILD_DIR}\"" >&2
    exit 1
fi

export PATH="${WIDGET_DIR}/tests/fake-bin:${PATH}"
export QML_IMPORT_PATH="${BUILD_DIR}/qml${QML_IMPORT_PATH:+:${QML_IMPORT_PATH}}"
export ECJTU_WEIXIN_TOKEN="TESTPREVIEWOPENIDXXXXXXXXXXX"
export ECJTU_FAKE_MODE="${ECJTU_FAKE_MODE:-normal}"
export ECJTU_FAKE_TOMORROW="$(TZ=Asia/Shanghai date -d tomorrow +%Y-%m-%d 2>/dev/null || true)"

exec qml6 "${SCRIPT_DIR}/PreviewWindow.qml"
