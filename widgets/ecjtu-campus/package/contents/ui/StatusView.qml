import QtQuick
import QtQuick.Layouts

Item {
    id: root

    // mode: "missing_token" | "loading" | "empty" | "error"
    property string mode: "loading"
    property bool isDark: true
    property bool reducedMotion: false
    property string selectedDay: "today"
    property string errorTitle: ""
    property string errorHint: ""
    property string diagnosticDetail: ""

    signal recheckRequested()
    signal retryRequested()
    signal switchDayRequested(string day)

    property bool showDiagnostic: false

    readonly property color inkPrimary: "#FFFFFF"
    readonly property color inkSecondary: Qt.rgba(1.0, 1.0, 1.0, 0.90)
    readonly property color inkMuted: Qt.rgba(1.0, 1.0, 1.0, 0.74)

    implicitHeight: {
        if (mode === "loading") return skeletonCol.implicitHeight + 16;
        if (mode === "missing_token") return missingCol.implicitHeight + 36;
        if (mode === "empty") return emptyCol.implicitHeight + 44;
        return errorCol.implicitHeight + 32;
    }

    onModeChanged: {
        if (mode !== "error") {
            showDiagnostic = false;
        }
    }

    // 1. 加载骨架屏：3 行通透玻璃光条 + 斜向镜面扫光
    ColumnLayout {
        id: skeletonCol
        visible: root.mode === "loading"
        anchors.left: parent.left
        anchors.right: parent.right
        anchors.verticalCenter: parent.verticalCenter
        spacing: 12

        Repeater {
            model: 3
            delegate: Item {
                Layout.fillWidth: true
                implicitHeight: 50
                clip: true

                RowLayout {
                    anchors.fill: parent
                    anchors.leftMargin: 8
                    anchors.rightMargin: 8
                    spacing: 14

                    Rectangle {
                        Layout.preferredWidth: 52
                        Layout.preferredHeight: 42
                        radius: 11
                        color: Qt.rgba(1, 1, 1, 0.11)
                        border.width: 1
                        border.color: Qt.rgba(1, 1, 1, 0.24)
                    }

                    ColumnLayout {
                        Layout.fillWidth: true
                        spacing: 7

                        Rectangle {
                            Layout.fillWidth: true
                            Layout.maximumWidth: index === 1 ? 155 : 210
                            implicitHeight: 13
                            radius: 6.5
                            color: Qt.rgba(1, 1, 1, 0.16)
                        }

                        Rectangle {
                            Layout.fillWidth: true
                            Layout.maximumWidth: index === 2 ? 185 : 130
                            implicitHeight: 10
                            radius: 5
                            color: Qt.rgba(1, 1, 1, 0.10)
                        }
                    }
                }

                Rectangle {
                    id: shimmerBand
                    visible: root.mode === "loading" && !root.reducedMotion
                    width: 90
                    height: parent.height * 1.8
                    y: -parent.height * 0.4
                    rotation: 16
                    x: -110
                    gradient: Gradient {
                        orientation: Gradient.Horizontal
                        GradientStop { position: 0.0; color: "transparent" }
                        GradientStop { position: 0.5; color: Qt.rgba(1, 1, 1, 0.18) }
                        GradientStop { position: 1.0; color: "transparent" }
                    }

                    NumberAnimation on x {
                        running: root.mode === "loading" && !root.reducedMotion
                        from: -110
                        to: 390
                        duration: 1350
                        loops: Animation.Infinite
                        easing.type: Easing.InOutSine
                    }
                }
            }
        }
    }

    // 2. 未配置凭证（不提供输入框，不发请求）
    ColumnLayout {
        id: missingCol
        visible: root.mode === "missing_token"
        anchors.left: parent.left
        anchors.right: parent.right
        anchors.verticalCenter: parent.verticalCenter
        anchors.leftMargin: 16
        anchors.rightMargin: 16
        spacing: 10

        Rectangle {
            Layout.alignment: Qt.AlignHCenter
            width: 42
            height: 42
            radius: 21
            color: Qt.rgba(1, 1, 1, 0.12)
            border.width: 1
            border.color: Qt.rgba(1, 1, 1, 0.28)

            Canvas {
                id: lockCanvas
                anchors.centerIn: parent
                width: 20
                height: 20
                onPaint: {
                    var ctx = getContext("2d");
                    ctx.clearRect(0, 0, width, height);
                    ctx.strokeStyle = "#FFFFFF";
                    ctx.lineWidth = 1.6;
                    ctx.lineCap = "round";
                    ctx.lineJoin = "round";
                    ctx.beginPath();
                    ctx.arc(10, 8, 3.6, Math.PI, 0, false);
                    ctx.stroke();
                    ctx.beginPath();
                    ctx.roundedRect(4.5, 9, 11, 8, 2.2, 2.2);
                    ctx.stroke();
                }
            }
        }

        Text {
            Layout.fillWidth: true
            horizontalAlignment: Text.AlignHCenter
            text: "未检测到教务凭证"
            font.pixelSize: 15
            font.weight: Font.DemiBold
            color: root.inkPrimary
        }

        Text {
            Layout.fillWidth: true
            horizontalAlignment: Text.AlignHCenter
            text: "请为 Plasma 会话配置环境变量 ECJTU_WEIXIN_TOKEN 后重新检测。"
            font.pixelSize: 12
            lineHeight: 1.35
            color: root.inkSecondary
            wrapMode: Text.WordWrap
        }

        GlassActionPill {
            Layout.alignment: Qt.AlignHCenter
            Layout.topMargin: 4
            label: "重新检测"
            onClicked: root.recheckRequested()
        }
    }

    // 3. 正常无课日
    ColumnLayout {
        id: emptyCol
        visible: root.mode === "empty"
        anchors.left: parent.left
        anchors.right: parent.right
        anchors.verticalCenter: parent.verticalCenter
        anchors.leftMargin: 16
        anchors.rightMargin: 16
        spacing: 9

        Text {
            Layout.fillWidth: true
            horizontalAlignment: Text.AlignHCenter
            text: root.selectedDay === "tomorrow" ? "明天没有课程安排" : "今天没有课程安排"
            font.pixelSize: 15
            font.weight: Font.Medium
            color: root.inkPrimary
        }

        Text {
            Layout.fillWidth: true
            horizontalAlignment: Text.AlignHCenter
            text: root.selectedDay === "tomorrow" ? "可以安心安排自习或休息。" : "全天空闲，也可以提前看看明天的课表。"
            font.pixelSize: 12
            color: root.inkMuted
            wrapMode: Text.WordWrap
        }

        GlassActionPill {
            Layout.alignment: Qt.AlignHCenter
            Layout.topMargin: 4
            label: root.selectedDay === "tomorrow" ? "返回今天" : "查看明天课表"
            onClicked: root.switchDayRequested(root.selectedDay === "tomorrow" ? "today" : "tomorrow")
        }
    }

    // 4. 分层错误提示 + 可折叠脱敏诊断
    ColumnLayout {
        id: errorCol
        visible: root.mode === "error"
        anchors.left: parent.left
        anchors.right: parent.right
        anchors.verticalCenter: parent.verticalCenter
        anchors.leftMargin: 12
        anchors.rightMargin: 12
        spacing: 8

        Rectangle {
            Layout.alignment: Qt.AlignHCenter
            width: 36
            height: 36
            radius: 18
            color: Qt.rgba(1.0, 0.41, 0.38, 0.22)
            border.width: 1
            border.color: Qt.rgba(1.0, 0.56, 0.52, 0.52)

            Text {
                anchors.centerIn: parent
                text: "!"
                font.pixelSize: 16
                font.weight: Font.Bold
                color: "#FCA5A5"
            }
        }

        Text {
            Layout.fillWidth: true
            horizontalAlignment: Text.AlignHCenter
            text: root.errorTitle || "课表查询失败"
            font.pixelSize: 14
            font.weight: Font.DemiBold
            color: root.inkPrimary
            wrapMode: Text.WordWrap
        }

        Text {
            Layout.fillWidth: true
            horizontalAlignment: Text.AlignHCenter
            text: root.errorHint || "请稍后重试。"
            font.pixelSize: 12
            lineHeight: 1.35
            color: root.inkSecondary
            wrapMode: Text.WordWrap
        }

        RowLayout {
            Layout.alignment: Qt.AlignHCenter
            Layout.topMargin: 4
            spacing: 10

            GlassActionPill {
                label: "重试"
                onClicked: root.retryRequested()
            }

            Text {
                visible: root.diagnosticDetail.length > 0
                text: root.showDiagnostic ? "收起诊断" : "查看诊断"
                font.pixelSize: 12
                color: root.inkMuted

                MouseArea {
                    anchors.fill: parent
                    anchors.margins: -4
                    cursorShape: Qt.PointingHandCursor
                    onClicked: root.showDiagnostic = !root.showDiagnostic
                }
            }
        }

        Rectangle {
            visible: root.showDiagnostic && root.diagnosticDetail.length > 0
            Layout.fillWidth: true
            Layout.topMargin: 2
            implicitHeight: diagText.implicitHeight + 14
            radius: 10
            color: Qt.rgba(0, 0, 0, 0.28)
            border.width: 1
            border.color: Qt.rgba(1, 1, 1, 0.16)

            Text {
                id: diagText
                anchors.fill: parent
                anchors.margins: 7
                text: root.diagnosticDetail
                font.pixelSize: 11
                color: root.inkSecondary
                wrapMode: Text.WrapAnywhere
            }
        }
    }

    component GlassActionPill: Rectangle {
        id: pillBtn
        property string label: ""
        signal clicked()

        implicitWidth: pillLabel.implicitWidth + 30
        implicitHeight: 34
        radius: 17
        color: {
            if (pillBtnMouse.pressed) return Qt.rgba(1, 1, 1, 0.28);
            if (pillBtnMouse.containsMouse) return Qt.rgba(1, 1, 1, 0.22);
            return Qt.rgba(1, 1, 1, 0.14);
        }
        border.width: 1
        border.color: Qt.rgba(1, 1, 1, 0.34)

        Text {
            id: pillLabel
            anchors.centerIn: parent
            text: pillBtn.label
            font.pixelSize: 12
            font.weight: Font.DemiBold
            color: "#FFFFFF"
        }

        MouseArea {
            id: pillBtnMouse
            anchors.fill: parent
            hoverEnabled: true
            cursorShape: Qt.PointingHandCursor
            onClicked: pillBtn.clicked()
        }
    }
}
