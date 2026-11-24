import QtQuick
import QtQuick.Effects
import QtQuick.Layouts

import "glass" as Glass
import "scheduleUtils.js" as ScheduleUtils

Item {
    id: root

    // 后端对象契约（可绑定 C++ CourseBackend，也兼容离线预览测试对象）
    required property var backend

    // 外观与双模式适配
    // drawOuterFrame = true：由自研 Clear Liquid Glass 主外壳呈现
    // insetPad：透明 Panel Popup / 预览留给柔和外投影的边距（桌面紧贴态可为 0）
    property bool drawOuterFrame: true
    property int insetPad: 0
    property Item backdropItem: null
    property bool isDark: true
    property bool reducedMotion: false
    property bool reducedTransparency: false

    // 内部解析状态（由 QML JS JSON.parse 解析 C++ 透传的 rawJson）
    property var parsedSchedule: null
    property string jsonParseError: ""
    property int scheduleVersion: 0

    // 对标 rdev/liquid-glass-react 的纯净白字阶 + 统一 GPU 柔影，在亮壁纸/深色终端/白窗下都清晰通透
    readonly property color inkPrimary: "#FFFFFF"
    readonly property color inkSecondary: Qt.rgba(1.0, 1.0, 1.0, 0.90)
    readonly property color inkMuted: Qt.rgba(1.0, 1.0, 1.0, 0.74)

    // C++ CourseBackend::State 枚举值映射：0 Idle, 1 MissingToken, 2 Loading, 3 Ready, 4 Error
    readonly property int backendState: backend ? backend.state : 0
    readonly property bool isMissingToken: backendState === 1
    readonly property bool isLoading: backendState === 2
    readonly property bool isBackendError: backendState === 4
    readonly property bool hasJsonError: backendState === 3 && jsonParseError.length > 0
    readonly property bool isReadyCourses: backendState === 3
        && !hasJsonError
        && parsedSchedule !== null
        && parsedSchedule.courses.length > 0
    readonly property bool isEmptyDay: backendState === 3
        && !hasJsonError
        && parsedSchedule !== null
        && parsedSchedule.courses.length === 0

    readonly property string headerDateText: {
        if (parsedSchedule && parsedSchedule.date) {
            return ScheduleUtils.formatDisplayDate(parsedSchedule.date);
        }
        return backend && backend.activeDate ? ScheduleUtils.formatDisplayDate(backend.activeDate) : "";
    }

    readonly property string headerWeekdayText: {
        if (parsedSchedule && parsedSchedule.weekday > 0) {
            return ScheduleUtils.formatWeekday(parsedSchedule.weekday);
        }
        return "";
    }

    readonly property string headerWeekBadge: {
        if (parsedSchedule && parsedSchedule.week > 0) {
            return "第 " + parsedSchedule.week + " 周";
        }
        return "";
    }

    readonly property string headerCourseCount: {
        if (isReadyCourses) {
            return parsedSchedule.courses.length + " 门课";
        }
        return "";
    }

    // 严格同心圆角几何：外圆角 28 - 内边距 14 = 内部主轨道圆角 14
    readonly property int cardWidth: 364
    readonly property int framePadding: 14

    implicitWidth: cardWidth + insetPad * 2
    implicitHeight: {
        var topChrome = isMissingToken ? (32 + framePadding * 2) : (130 + framePadding * 2);
        var bodyH = 160;
        if (isReadyCourses) {
            bodyH = Math.min(430, Math.max(124, courseList.contentHeight + 8));
        } else {
            bodyH = Math.max(154, statusView.implicitHeight + 12);
        }
        var cardH = Math.max(236, Math.min(576, topChrome + bodyH));
        return cardH + insetPad * 2;
    }

    function scrollCoursesTo(offsetY) {
        if (isReadyCourses) {
            courseList.contentY = offsetY;
        }
    }

    function syncParsedSchedule() {
        if (!backend || backend.state !== 3) {
            jsonParseError = "";
            if (!backend || backend.state === 1 || backend.state === 4) {
                parsedSchedule = null;
            }
            scheduleVersion += 1;
            return;
        }
        var res = ScheduleUtils.parseScheduleJson(backend.rawJson);
        if (res.ok) {
            parsedSchedule = res.schedule;
            jsonParseError = "";
        } else {
            parsedSchedule = null;
            jsonParseError = res.error || "invalid json";
        }
        scheduleVersion += 1;
    }

    Connections {
        target: root.backend
        function onRawJsonChanged() { root.syncParsedSchedule(); }
        function onStateChanged() { root.syncParsedSchedule(); }
    }

    Component.onCompleted: syncParsedSchedule()

    // Clear Liquid Glass 主外壳（桌面真折射 / 弹窗 KWin 活色模糊 + 刀锋级 1.5px 镜面边）
    Glass.LiquidGlassSurface {
        id: glassBg
        anchors.fill: parent
        visible: root.drawOuterFrame
        backdropItem: root.backdropItem
        insetPad: root.insetPad
        isDark: root.isDark
        reducedMotion: root.reducedMotion
        reducedTransparency: root.reducedTransparency
        cornerRadius: 28
    }

    // 内容安全容器：单 Pass GPU 柔和文字阴影（复刻 rdev/liquid-glass-react 的 0 2px 12px rgba(0,0,0,0.4) 悬浮质感）
    Item {
        id: innerFrame
        anchors.fill: parent
        anchors.margins: root.insetPad + root.framePadding
        layer.enabled: !root.reducedTransparency && GraphicsInfo.api !== GraphicsInfo.Software
        layer.effect: MultiEffect {
            shadowEnabled: true
            shadowColor: Qt.rgba(0.0, 0.0, 0.0, 0.52)
            shadowBlur: 0.40
            shadowHorizontalOffset: 0
            shadowVerticalOffset: 1.5
            autoPaddingEnabled: false
        }

        // 第 1 行：极简蚀刻风品牌微标 ([GH] ECJTU) + 晶体周次胶囊 + 晶体圆钮刷新
        RowLayout {
            id: brandRow
            anchors.left: parent.left
            anchors.right: parent.right
            anchors.top: parent.top
            anchors.leftMargin: 4
            anchors.rightMargin: 2
            spacing: 6

            Glass.TintedIcon {
                source: Qt.resolvedUrl("../icons/github-mark.svg")
                iconSize: 14
                color: root.inkSecondary
                opacity: 0.88
            }

            Text {
                text: "ECJTU"
                font.pixelSize: 11
                font.weight: Font.DemiBold
                font.letterSpacing: 1.0
                color: root.inkMuted
            }

            Item { Layout.fillWidth: true }

            // 通透晶体周次微胶囊（告别实白贴片）
            Rectangle {
                visible: !root.isMissingToken && root.headerWeekBadge.length > 0
                implicitWidth: weekText.implicitWidth + 16
                implicitHeight: 22
                radius: 11
                color: Qt.rgba(1, 1, 1, 0.13)
                border.width: 1
                border.color: Qt.rgba(1, 1, 1, 0.30)

                Text {
                    id: weekText
                    anchors.centerIn: parent
                    text: root.headerWeekBadge
                    font.pixelSize: 11
                    font.weight: Font.DemiBold
                    font.features: ({ "tnum": 1 })
                    color: root.inkSecondary
                }
            }

            // 通透液态圆钮刷新
            Rectangle {
                id: refreshBtn
                visible: !root.isMissingToken
                Layout.preferredWidth: 26
                Layout.preferredHeight: 26
                Layout.minimumWidth: 26
                Layout.minimumHeight: 26
                radius: 13
                scale: refreshMouse.pressed ? 0.90 : 1.0
                color: refreshMouse.containsMouse
                    ? Qt.rgba(1, 1, 1, 0.22)
                    : Qt.rgba(1, 1, 1, 0.10)
                border.width: 1
                border.color: refreshMouse.containsMouse
                    ? Qt.rgba(1, 1, 1, 0.44)
                    : Qt.rgba(1, 1, 1, 0.24)

                Behavior on scale {
                    enabled: !root.reducedMotion
                    NumberAnimation { duration: 110; easing.type: Easing.OutQuad }
                }

                Canvas {
                    id: refreshGlyph
                    anchors.centerIn: parent
                    width: 14
                    height: 14
                    opacity: refreshMouse.containsMouse || root.isLoading ? 1.0 : 0.82

                    RotationAnimation on rotation {
                        running: root.isLoading && !root.reducedMotion
                        from: 0
                        to: 360
                        duration: 850
                        loops: Animation.Infinite
                    }

                    onPaint: {
                        var ctx = getContext("2d");
                        ctx.clearRect(0, 0, width, height);
                        ctx.strokeStyle = "#FFFFFF";
                        ctx.lineWidth = 1.5;
                        ctx.lineCap = "round";
                        ctx.beginPath();
                        ctx.arc(7, 7, 4.5, -Math.PI * 0.65, Math.PI * 0.92, false);
                        ctx.stroke();
                        ctx.beginPath();
                        ctx.moveTo(9.4, 2.2);
                        ctx.lineTo(10.6, 4.6);
                        ctx.lineTo(8.0, 4.9);
                        ctx.stroke();
                    }
                }

                MouseArea {
                    id: refreshMouse
                    anchors.fill: parent
                    hoverEnabled: true
                    cursorShape: Qt.PointingHandCursor
                    enabled: !root.isLoading
                    onClicked: {
                        if (root.backend) {
                            root.backend.retry();
                        }
                    }
                }
            }
        }

        // 第 2 行：舒展大号日期抬头 + 星期 + 右侧当日课程负荷微注
        RowLayout {
            id: dateHeaderRow
            visible: !root.isMissingToken
            anchors.left: parent.left
            anchors.right: parent.right
            anchors.top: brandRow.bottom
            anchors.topMargin: 5
            anchors.leftMargin: 4
            anchors.rightMargin: 4
            spacing: 9

            Text {
                text: root.headerDateText
                font.pixelSize: 24
                font.weight: Font.DemiBold
                font.letterSpacing: -0.6
                font.features: ({ "tnum": 1 })
                color: root.inkPrimary
            }

            Text {
                visible: root.headerWeekdayText.length > 0
                text: root.headerWeekdayText
                font.pixelSize: 13
                font.weight: Font.Medium
                color: root.inkSecondary
                Layout.alignment: Qt.AlignBaseline
            }

            Item { Layout.fillWidth: true }

            Text {
                visible: root.headerCourseCount.length > 0
                text: root.headerCourseCount
                font.pixelSize: 12
                font.weight: Font.Medium
                font.features: ({ "tnum": 1 })
                color: root.inkMuted
                Layout.alignment: Qt.AlignBaseline
            }
        }

        // L3 光学主角：全宽通透液态果冻分段滑块（点击 + 横向拖拽 + 真实壁纸折射）
        Glass.LiquidSegmentedControl {
            id: daySwitch
            visible: !root.isMissingToken
            anchors.left: parent.left
            anchors.right: parent.right
            anchors.top: dateHeaderRow.bottom
            anchors.topMargin: 10
            height: 38
            currentKey: root.backend ? root.backend.selectedDay : "today"
            isDark: root.isDark
            reducedMotion: root.reducedMotion
            reducedTransparency: root.reducedTransparency
            peekSourceItem: root.backdropItem
            peekActive: root.backdropItem !== null
            snapshotTrigger: root.scheduleVersion
            onActivated: function (day) {
                if (root.backend) {
                    root.backend.selectDay(day);
                }
            }
        }

        // 内容舞台
        Item {
            id: scheduleStage
            anchors.left: parent.left
            anchors.right: parent.right
            anchors.top: root.isMissingToken ? brandRow.bottom : daySwitch.bottom
            anchors.topMargin: 10
            anchors.bottom: parent.bottom
            clip: true

            ListView {
                id: courseList
                anchors.fill: parent
                visible: root.isReadyCourses
                clip: true
                boundsBehavior: Flickable.StopAtBounds
                spacing: 0
                topMargin: 0
                bottomMargin: 6
                model: root.isReadyCourses ? root.parsedSchedule.courses : []

                delegate: CourseRow {
                    required property var modelData
                    required property int index

                    readonly property int prevSlot: {
                        if (index <= 0 || !root.parsedSchedule) return 0;
                        var prev = root.parsedSchedule.courses[index - 1];
                        return prev ? prev.sectionSlot : 0;
                    }
                    readonly property int nextSlot: {
                        if (!root.parsedSchedule || index >= root.parsedSchedule.courses.length - 1) return 0;
                        var next = root.parsedSchedule.courses[index + 1];
                        return next ? next.sectionSlot : 0;
                    }
                    readonly property bool startsNewSection: modelData.sectionSlot > 0
                        && (index === 0 || prevSlot !== modelData.sectionSlot)

                    width: courseList.width
                    course: modelData
                    isDark: root.isDark
                    reducedMotion: root.reducedMotion
                    sectionHeaderText: startsNewSection ? ScheduleUtils.sectionSlotLabel(modelData.sectionSlot) : ""
                    showDivider: index < courseList.count - 1 && nextSlot === modelData.sectionSlot
                }
            }

            StatusView {
                id: statusView
                visible: !root.isReadyCourses
                anchors.fill: parent
                isDark: root.isDark
                reducedMotion: root.reducedMotion
                selectedDay: root.backend ? root.backend.selectedDay : "today"
                mode: {
                    if (root.isMissingToken) return "missing_token";
                    if (root.isLoading) return "loading";
                    if (root.isEmptyDay) return "empty";
                    if (root.isBackendError || root.hasJsonError) return "error";
                    return "loading";
                }
                errorTitle: root.hasJsonError
                    ? "课表数据格式异常"
                    : (root.backend ? root.backend.errorTitle : "")
                errorHint: root.hasJsonError
                    ? "CLI 返回内容不是有效的课表 JSON，请检查 ecjtu 版本或稍后重试。"
                    : (root.backend ? root.backend.errorHint : "")
                diagnosticDetail: root.hasJsonError
                    ? root.jsonParseError
                    : (root.backend ? root.backend.diagnosticDetail : "")

                onRecheckRequested: {
                    if (root.backend) root.backend.recheckToken();
                }
                onRetryRequested: {
                    if (root.backend) root.backend.retry();
                }
                onSwitchDayRequested: function (day) {
                    if (root.backend) root.backend.selectDay(day);
                }
            }
        }
    }
}
