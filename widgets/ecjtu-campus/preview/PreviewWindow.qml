import QtQuick
import QtQuick.Layouts
import QtQuick.Window

import io.github.mochenya.ecjtucampus.backend 1.0 as CampusBackend
import "../build/package/contents/ui" as CampusUI
import "../build/package/contents/ui/glass" as Glass

Window {
    id: win

    width: 980
    height: 740
    visible: true
    title: "华交校园 · Clear Crystal Liquid Glass 预览"
    color: "#090D16"

    property bool darkWallpaper: false
    property bool desktopRefractMode: true
    property bool reducedTransparency: false
    property bool reducedMotion: false
    property bool popupOpen: true

    // "live_cpp" 使用真实 C++ CourseBackend + fake ecjtu 进程；其余用于快速视觉验收
    property string demoScenario: "live_cpp"

    CampusBackend.CourseBackend {
        id: realBackend
    }

    QtObject {
        id: mockBackend
        property bool hasToken: win.demoScenario !== "missing_token"
        property int state: {
            if (win.demoScenario === "missing_token") return 1;
            if (win.demoScenario === "loading") return 2;
            if (win.demoScenario === "exit3" || win.demoScenario === "exit4") return 4;
            return 3;
        }
        property string selectedDay: "today"
        property string activeDate: selectedDay === "tomorrow" ? "2026-09-15" : "2026-09-14"
        property string rawJson: {
            if (win.demoScenario === "empty") {
                return JSON.stringify({ date: activeDate, weekday: 6, week: 13, courses: [] });
            }
            if (win.demoScenario === "badjson") {
                return "{broken-json";
            }
            if (selectedDay === "tomorrow") {
                return JSON.stringify({
                    date: "2026-09-15",
                    weekday: 2,
                    week: 13,
                    courses: [
                        { name: "数据结构与算法", status: "上课", timeText: "1-16 1,2", weeks: [{start:1, end:16}], periods: [1,2], location: "教31栋208", teacher: "教师D" },
                        { name: "计算机网络", status: "实验", timeText: "3-10 5,6", weeks: [{start:3, end:10}], periods: [5,6], location: "网络工程实验中心(教26栋302)", teacher: "教师E" }
                    ]
                });
            }
            return JSON.stringify({
                date: "2026-09-14",
                weekday: 1,
                week: 13,
                courses: [
                    { name: "大学物理Ⅱ", status: "上课", timeText: "1-16 1,2", weeks: [{start:1, end:16}], periods: [1,2], location: "31-206", teacher: "教师A" },
                    { name: "软件工程 (A)", status: "实验", timeText: "3-15 3,4", weeks: [{start:3, end:15}], periods: [3,4], location: "软件设计实验室(教26栋403; 教26栋404)", teacher: "教师B" },
                    { name: "概率论与数理统计", status: "上课", timeText: "1-16 5,6", weeks: [{start:1, end:16}], periods: [5,6], location: "31-205", teacher: "教师C" },
                    { name: "线性代数阶段测验", status: "考试", timeText: "13 7,8", weeks: [{start:13, end:13}], periods: [7,8], location: "教31栋阶梯教室101", teacher: "教师D" },
                    { name: "铁路运输导论", status: "上课", timeText: "1-12 11,12", weeks: [{start:1, end:12}], periods: [11,12], location: "教21栋305", teacher: "" }
                ]
            });
        }
        property string errorCode: win.demoScenario === "exit3" ? "not_bound" : "upstream"
        property string errorTitle: win.demoScenario === "exit3" ? "教务凭证未绑定或已失效" : "教务系统暂时无法访问"
        property string errorHint: win.demoScenario === "exit3"
            ? "请前往华东交通大学教务微信公众号重新绑定后再试。"
            : "连接超时或学校服务器返回异常，请稍后重试。"
        property string diagnosticDetail: win.demoScenario === "exit3"
            ? "phase=error kind=notbound err=\"course.Day: notbound token=TEST***\""
            : "phase=error kind=transport err=\"course.Day: context deadline exceeded\""

        function openAndRefreshToday() { selectedDay = "today"; }
        function selectDay(day) { selectedDay = day === "tomorrow" ? "tomorrow" : "today"; }
        function retry() {}
        function recheckToken() { win.demoScenario = "live_cpp"; realBackend.openAndRefreshToday(); }
    }

    Component.onCompleted: {
        realBackend.openAndRefreshToday();
    }

    // 高色彩对比 + 清晰几何元素的丰富桌面壁纸，直观检验 Clear Crystal 边缘 Snell 弯曲、光学折叠与 RGB 色散
    Item {
        id: wallpaperLayer
        anchors.fill: parent

        Rectangle {
            anchors.fill: parent
            gradient: Gradient {
                GradientStop { position: 0.0; color: win.darkWallpaper ? "#081022" : "#B8E1FF" }
                GradientStop { position: 0.42; color: win.darkWallpaper ? "#131F44" : "#E6D9FF" }
                GradientStop { position: 0.78; color: win.darkWallpaper ? "#1B183B" : "#FFE2CC" }
                GradientStop { position: 1.0; color: win.darkWallpaper ? "#0C2134" : "#BCEEE0" }
            }
        }

        // 绚丽色块 + 几何斜带 + 模拟桌面图标（穿过玻璃边缘时立刻呈现清晰的水滴折射弯曲！）
        Canvas {
            id: scenicCanvas
            anchors.fill: parent
            onPaint: {
                var ctx = getContext("2d");
                ctx.clearRect(0, 0, width, height);

                function drawOrb(cx, cy, r, c0, c1) {
                    var g = ctx.createRadialGradient(cx, cy, 0, cx, cy, r);
                    g.addColorStop(0.0, c0);
                    g.addColorStop(0.55, c1);
                    g.addColorStop(1.0, "rgba(0,0,0,0)");
                    ctx.fillStyle = g;
                    ctx.beginPath();
                    ctx.arc(cx, cy, r, 0, Math.PI * 2);
                    ctx.fill();
                }

                if (win.darkWallpaper) {
                    drawOrb(150, 140, 240, "rgba(38, 166, 255, 0.78)", "rgba(30, 110, 230, 0.22)");
                    drawOrb(500, 260, 260, "rgba(218, 68, 255, 0.68)", "rgba(130, 42, 210, 0.18)");
                    drawOrb(180, 560, 240, "rgba(20, 230, 192, 0.65)", "rgba(14, 150, 160, 0.16)");
                    drawOrb(490, 600, 220, "rgba(255, 138, 62, 0.56)", "rgba(215, 76, 48, 0.14)");
                } else {
                    drawOrb(145, 145, 250, "rgba(56, 189, 248, 0.78)", "rgba(125, 211, 252, 0.25)");
                    drawOrb(495, 255, 260, "rgba(192, 132, 252, 0.72)", "rgba(216, 180, 254, 0.22)");
                    drawOrb(185, 570, 240, "rgba(52, 211, 153, 0.68)", "rgba(110, 231, 183, 0.20)");
                    drawOrb(490, 595, 220, "rgba(251, 146, 60, 0.66)", "rgba(253, 186, 116, 0.20)");
                }

                // 穿过卡片左右与上下边缘的清晰斜纹，方便肉眼检验 26px Bevel 的 Snell 弯曲与色散
                ctx.save();
                ctx.lineWidth = 3;
                ctx.strokeStyle = win.darkWallpaper ? "rgba(255,255,255,0.22)" : "rgba(15, 45, 95, 0.18)";
                for (var x = -height; x < width + height; x += 48) {
                    ctx.beginPath();
                    ctx.moveTo(x, 0);
                    ctx.lineTo(x + height * 0.52, height);
                    ctx.stroke();
                }

                // 模拟高对比桌面图标块紧贴卡片边缘（直接展示边缘厚玻璃折射与光学折叠）
                var tiles = [
                    { x: 108, y: 120, w: 54, h: 54, c: "#0EA5E9" },
                    { x: 472, y: 168, w: 56, h: 56, c: "#F43F5E" },
                    { x: 112, y: 372, w: 54, h: 54, c: "#10B981" },
                    { x: 470, y: 452, w: 58, h: 58, c: "#F59E0B" },
                    { x: 280, y: 66,  w: 64, h: 36, c: "#8B5CF6" }
                ];
                for (var i = 0; i < tiles.length; ++i) {
                    var t = tiles[i];
                    ctx.fillStyle = t.c;
                    ctx.beginPath();
                    ctx.roundedRect(t.x, t.y, t.w, t.h, 14, 14);
                    ctx.fill();
                    ctx.strokeStyle = "rgba(255,255,255,0.55)";
                    ctx.lineWidth = 1.5;
                    ctx.stroke();
                }
                ctx.restore();
            }
            Connections {
                target: win
                function onDarkWallpaperChanged() { scenicCanvas.requestPaint(); }
            }
        }
    }

    // 模拟 KWin 对 Panel Popup 底层提供的活色奶油模糊（3-Pass Kawase + 高饱和）
    ShaderEffectSource {
        id: popupMockCrop
        visible: false
        sourceItem: !win.desktopRefractMode ? wallpaperLayer : null
        sourceRect: Qt.rect(widgetHost.x + 14, widgetHost.y + 14, widgetHost.width - 28, widgetHost.height - 28)
        textureSize: Qt.size(Math.max(4, Math.round((widgetHost.width - 28) * 0.38)),
                             Math.max(4, Math.round((widgetHost.height - 28) * 0.38)))
        live: !win.desktopRefractMode
        smooth: true
    }

    ShaderEffect {
        id: popupMockBlur1
        visible: false
        width: Math.max(4, Math.round((widgetHost.width - 28) * 0.38))
        height: Math.max(4, Math.round((widgetHost.height - 28) * 0.38))
        property variant source: popupMockCrop
        property vector2d texelSize: Qt.vector2d(1.0 / width, 1.0 / height)
        property real offset: 1.15
        fragmentShader: Qt.resolvedUrl("../build/package/contents/ui/shaders/kawase_blur.frag.qsb")
    }

    ShaderEffectSource {
        id: popupMockBlur1Src
        visible: false
        sourceItem: !win.desktopRefractMode ? popupMockBlur1 : null
        live: !win.desktopRefractMode
        smooth: true
    }

    ShaderEffect {
        id: popupMockBlur2
        visible: false
        width: Math.max(4, Math.round((widgetHost.width - 28) * 0.38))
        height: Math.max(4, Math.round((widgetHost.height - 28) * 0.38))
        property variant source: popupMockBlur1Src
        property vector2d texelSize: Qt.vector2d(1.0 / width, 1.0 / height)
        property real offset: 2.25
        fragmentShader: Qt.resolvedUrl("../build/package/contents/ui/shaders/kawase_blur.frag.qsb")
    }

    ShaderEffectSource {
        id: popupMockBlur2Src
        visible: false
        sourceItem: !win.desktopRefractMode ? popupMockBlur2 : null
        live: !win.desktopRefractMode
        smooth: true
    }

    // 仅在弹窗模拟态下放在透明 CampusPanel 背后，模拟 KWin enableBlurBehind 提供的 Squircle 奶油模糊层
    Rectangle {
        visible: win.popupOpen && !win.desktopRefractMode && !win.reducedTransparency
        x: widgetHost.x + 14
        y: widgetHost.y + 14
        width: widgetHost.width - 28
        height: widgetHost.height - 28
        radius: 28
        color: "transparent"
        clip: true

        ShaderEffect {
            anchors.fill: parent
            property variant source: popupMockBlur2Src
            property vector2d texelSize: Qt.vector2d(1.0 / Math.max(4, width), 1.0 / Math.max(4, height))
            property real offset: 3.45
            fragmentShader: Qt.resolvedUrl("../build/package/contents/ui/shaders/kawase_blur.frag.qsb")
        }
    }

    // 模拟 Plasma 顶部面板
    Rectangle {
        id: mockTopPanel
        anchors.left: parent.left
        anchors.right: parent.right
        anchors.top: parent.top
        height: 36
        color: win.darkWallpaper ? Qt.rgba(0.05, 0.08, 0.14, 0.62) : Qt.rgba(1, 1, 1, 0.52)

        Rectangle {
            anchors.left: parent.left
            anchors.right: parent.right
            anchors.bottom: parent.bottom
            height: 1
            color: win.darkWallpaper ? Qt.rgba(1, 1, 1, 0.14) : Qt.rgba(1, 1, 1, 0.72)
        }

        Text {
            anchors.left: parent.left
            anchors.leftMargin: 16
            anchors.verticalCenter: parent.verticalCenter
            text: "KDE Plasma 6 面板预览"
            font.pixelSize: 12
            color: win.darkWallpaper ? Qt.rgba(1, 1, 1, 0.72) : Qt.rgba(0.07, 0.12, 0.22, 0.72)
        }

        Rectangle {
            anchors.centerIn: parent
            implicitWidth: compactPreviewRow.implicitWidth + 16
            height: 26
            radius: 13
            color: compactPreviewMouse.containsMouse || win.popupOpen
                ? (win.darkWallpaper ? Qt.rgba(1, 1, 1, 0.16) : Qt.rgba(1, 1, 1, 0.56))
                : "transparent"
            border.width: compactPreviewMouse.containsMouse || win.popupOpen ? 1 : 0
            border.color: win.darkWallpaper ? Qt.rgba(1, 1, 1, 0.30) : Qt.rgba(1, 1, 1, 0.90)

            RowLayout {
                id: compactPreviewRow
                anchors.centerIn: parent
                spacing: 5

                Glass.TintedIcon {
                    source: Qt.resolvedUrl("../build/package/contents/icons/github-mark.svg")
                    iconSize: 15
                    color: win.darkWallpaper ? "#F8FAFC" : "#0A101D"
                }

                Text {
                    text: "ECJTU"
                    font.pixelSize: 12
                    font.weight: Font.DemiBold
                    font.letterSpacing: 0.5
                    color: win.darkWallpaper ? "#F8FAFC" : "#0A101D"
                }
            }

            MouseArea {
                id: compactPreviewMouse
                anchors.fill: parent
                hoverEnabled: true
                cursorShape: Qt.PointingHandCursor
                onClicked: {
                    win.popupOpen = !win.popupOpen;
                    if (win.popupOpen && win.demoScenario === "live_cpp") {
                        realBackend.openAndRefreshToday();
                    }
                }
            }
        }
    }

    // 主舞台：渲染真实 CampusPanel + Clear Crystal Liquid Glass
    Item {
        id: widgetHost
        x: 118
        y: 66
        width: campusPanel.implicitWidth
        height: campusPanel.implicitHeight
        visible: win.popupOpen
        opacity: win.popupOpen ? 1.0 : 0.0
        scale: win.popupOpen ? 1.0 : 0.97

        Behavior on opacity { enabled: !win.reducedMotion; NumberAnimation { duration: 170; easing.type: Easing.OutCubic } }
        Behavior on scale { enabled: !win.reducedMotion; NumberAnimation { duration: 170; easing.type: Easing.OutCubic } }
        Behavior on height { enabled: !win.reducedMotion; NumberAnimation { duration: 190; easing.type: Easing.OutQuad } }

        CampusUI.CampusPanel {
            id: campusPanel
            anchors.fill: parent
            backend: win.demoScenario === "live_cpp" ? realBackend : mockBackend
            drawOuterFrame: true
            insetPad: 14
            isDark: win.darkWallpaper
            reducedMotion: win.reducedMotion
            reducedTransparency: win.reducedTransparency
            backdropItem: win.desktopRefractMode ? wallpaperLayer : null
        }
    }

    // 右侧调试控制台
    Rectangle {
        anchors.right: parent.right
        anchors.rightMargin: 24
        anchors.top: mockTopPanel.bottom
        anchors.topMargin: 28
        width: 286
        implicitHeight: ctrlCol.implicitHeight + 28
        radius: 20
        color: win.darkWallpaper ? Qt.rgba(0.05, 0.08, 0.14, 0.76) : Qt.rgba(1, 1, 1, 0.72)
        border.width: 1
        border.color: win.darkWallpaper ? Qt.rgba(1, 1, 1, 0.16) : Qt.rgba(1, 1, 1, 0.85)

        ColumnLayout {
            id: ctrlCol
            anchors.left: parent.left
            anchors.right: parent.right
            anchors.top: parent.top
            anchors.margins: 14
            spacing: 10

            Text {
                text: "Clear Crystal 光学调试台"
                font.pixelSize: 14
                font.weight: Font.DemiBold
                color: win.darkWallpaper ? "#F8FAFC" : "#0A101D"
            }

            Text {
                text: "渲染模式与背景"
                font.pixelSize: 12
                color: win.darkWallpaper ? "#94A3B8" : "#4A586D"
            }

            GridLayout {
                Layout.fillWidth: true
                columns: 2
                columnSpacing: 8
                rowSpacing: 8

                ToggleChip {
                    label: win.desktopRefractMode ? "桌面真实折射" : "面板透明弹窗"
                    active: win.desktopRefractMode
                    dark: win.darkWallpaper
                    onClicked: win.desktopRefractMode = !win.desktopRefractMode
                }
                ToggleChip {
                    label: win.darkWallpaper ? "深色壁纸" : "浅色壁纸"
                    active: !win.darkWallpaper
                    dark: win.darkWallpaper
                    onClicked: win.darkWallpaper = !win.darkWallpaper
                }
                ToggleChip {
                    label: "降低透明度"
                    active: win.reducedTransparency
                    dark: win.darkWallpaper
                    onClicked: win.reducedTransparency = !win.reducedTransparency
                }
                ToggleChip {
                    label: "减少动画"
                    active: win.reducedMotion
                    dark: win.darkWallpaper
                    onClicked: win.reducedMotion = !win.reducedMotion
                }
            }

            Text {
                Layout.topMargin: 4
                text: "业务状态切换"
                font.pixelSize: 12
                color: win.darkWallpaper ? "#94A3B8" : "#4A586D"
            }

            GridLayout {
                Layout.fillWidth: true
                columns: 2
                columnSpacing: 8
                rowSpacing: 8

                Repeater {
                    model: [
                        { key: "live_cpp", text: "实测 C++ 进程" },
                        { key: "empty", text: "正常无课日" },
                        { key: "loading", text: "加载骨架屏" },
                        { key: "missing_token", text: "未配置凭证" },
                        { key: "exit3", text: "未绑定 (码3)" },
                        { key: "exit4", text: "上游异常 (码4)" },
                        { key: "badjson", text: "JSON 损坏" }
                    ]
                    delegate: ToggleChip {
                        required property var modelData
                        label: modelData.text
                        active: win.demoScenario === modelData.key
                        dark: win.darkWallpaper
                        onClicked: {
                            win.demoScenario = modelData.key;
                            if (modelData.key === "live_cpp") {
                                realBackend.openAndRefreshToday();
                            }
                        }
                    }
                }
            }

            Text {
                Layout.fillWidth: true
                Layout.topMargin: 4
                text: "可横向拖拽「今天 / 明天」水滴滑块，或在满课列表向上滚动体验实时 Snell 折射与柯西色散。"
                font.pixelSize: 11
                lineHeight: 1.35
                color: win.darkWallpaper ? "#94A3B8" : "#4A586D"
                wrapMode: Text.WordWrap
            }
        }
    }

    component ToggleChip: Rectangle {
        id: chip
        property string label: ""
        property bool active: false
        property bool dark: true
        signal clicked()

        Layout.fillWidth: true
        implicitHeight: 30
        radius: 10
        color: active
            ? (dark ? Qt.rgba(0.22, 0.74, 0.97, 0.24) : Qt.rgba(0.05, 0.65, 0.91, 0.16))
            : (dark ? Qt.rgba(1, 1, 1, 0.08) : Qt.rgba(0.06, 0.11, 0.22, 0.06))
        border.width: 1
        border.color: active
            ? (dark ? Qt.rgba(0.49, 0.83, 0.99, 0.55) : Qt.rgba(0.01, 0.52, 0.78, 0.40))
            : (dark ? Qt.rgba(1, 1, 1, 0.14) : Qt.rgba(0.06, 0.11, 0.22, 0.10))

        Text {
            anchors.centerIn: parent
            text: chip.label
            font.pixelSize: 12
            font.weight: chip.active ? Font.DemiBold : Font.Normal
            color: chip.dark ? "#F8FAFC" : "#0A101D"
        }

        MouseArea {
            anchors.fill: parent
            cursorShape: Qt.PointingHandCursor
            onClicked: chip.clicked()
        }
    }
}
