import QtQuick
import QtQuick.Layouts

Item {
    id: root

    property var course: ({})
    property bool isDark: true
    property bool showDivider: true
    property string sectionHeaderText: ""
    property bool reducedMotion: false

    property bool expanded: false
    property real pointerX: width * 0.5
    property bool rowHovered: false

    readonly property color inkPrimary: "#FFFFFF"
    readonly property color inkSecondary: Qt.rgba(1.0, 1.0, 1.0, 0.90)
    readonly property color inkMuted: Qt.rgba(1.0, 1.0, 1.0, 0.74)

    readonly property bool isLab: course.status === "实验"
    readonly property bool isExam: course.status === "考试"
    readonly property bool showStatusBadge: isLab || isExam
    readonly property bool hasDetails: (course.weeksLabel && course.weeksLabel.length > 0)
        || (course.location && course.location.length > 0)
        || (course.periodFull && course.periodFull.length > 0)

    implicitHeight: (sectionHeaderText.length > 0 ? sectionHeader.height : 0) + contentColumn.implicitHeight

    // 轻蚀刻时段眉标（上午 / 下午 / 晚间，严格来自节次区间，绝不虚构钟点）
    Item {
        id: sectionHeader
        visible: root.sectionHeaderText.length > 0
        anchors.left: parent.left
        anchors.right: parent.right
        anchors.top: parent.top
        height: root.sectionHeaderText.length > 0 ? 22 : 0

        RowLayout {
            anchors.left: parent.left
            anchors.right: parent.right
            anchors.verticalCenter: parent.verticalCenter
            anchors.leftMargin: 6
            anchors.rightMargin: 6
            spacing: 8

            Text {
                text: root.sectionHeaderText
                font.pixelSize: 11
                font.weight: Font.DemiBold
                font.letterSpacing: 0.8
                color: root.inkMuted
            }

            Rectangle {
                Layout.fillWidth: true
                height: 1
                color: Qt.rgba(1, 1, 1, 0.14)
            }
        }
    }

    // 悬停 / 展开时的轻盈晶体行透镜
    Rectangle {
        id: hoverBackdrop
        anchors.fill: mainSurface
        radius: 14
        opacity: root.rowHovered || root.expanded ? 1.0 : 0.0
        color: Qt.rgba(1, 1, 1, root.expanded ? 0.14 : 0.09)
        border.width: 1
        border.color: Qt.rgba(1, 1, 1, root.expanded ? 0.30 : 0.22)
        clip: true

        Behavior on opacity {
            enabled: !root.reducedMotion
            NumberAnimation { duration: 145; easing.type: Easing.OutQuad }
        }

        // 光标随动镜面聚光带
        Rectangle {
            visible: root.rowHovered && !root.reducedMotion
            width: 156
            height: 156
            radius: 78
            x: root.pointerX - width * 0.5
            y: (parent.height - height) * 0.5
            gradient: Gradient {
                GradientStop { position: 0.0; color: Qt.rgba(1, 1, 1, 0.14) }
                GradientStop { position: 1.0; color: "transparent" }
            }
        }
    }

    Item {
        id: mainSurface
        anchors.left: parent.left
        anchors.right: parent.right
        anchors.top: sectionHeader.bottom
        height: contentColumn.implicitHeight

        ColumnLayout {
            id: contentColumn
            anchors.left: parent.left
            anchors.right: parent.right
            spacing: 0

            RowLayout {
                Layout.fillWidth: true
                Layout.leftMargin: 6
                Layout.rightMargin: 8
                Layout.topMargin: 8
                Layout.bottomMargin: root.expanded ? 6 : 8
                spacing: 11

                // 左侧通透晶体节次小块（复刻参考图 JD 头像的轻盈通透玻璃质感，告别实白方块）
                Rectangle {
                    id: periodTile
                    Layout.preferredWidth: 52
                    Layout.minimumWidth: 52
                    Layout.maximumWidth: 52
                    Layout.preferredHeight: 42
                    Layout.alignment: Qt.AlignTop
                    radius: 11
                    color: {
                        if (root.isExam) {
                            return Qt.rgba(1.0, 0.72, 0.18, 0.22);
                        }
                        if (root.isLab) {
                            return Qt.rgba(0.22, 0.74, 0.97, 0.22);
                        }
                        return Qt.rgba(1, 1, 1, 0.12);
                    }
                    border.width: 1
                    border.color: {
                        if (root.isExam) {
                            return Qt.rgba(1.0, 0.84, 0.42, 0.54);
                        }
                        if (root.isLab) {
                            return Qt.rgba(0.55, 0.85, 1.0, 0.54);
                        }
                        return Qt.rgba(1, 1, 1, 0.28);
                    }

                    // 顶部极细镜面发丝线
                    Rectangle {
                        anchors.left: parent.left
                        anchors.right: parent.right
                        anchors.leftMargin: 7
                        anchors.rightMargin: 7
                        anchors.top: parent.top
                        anchors.topMargin: 1
                        height: 1
                        radius: 0.5
                        color: Qt.rgba(1, 1, 1, 0.34)
                    }

                    ColumnLayout {
                        anchors.centerIn: parent
                        spacing: 0

                        Text {
                            Layout.alignment: Qt.AlignHCenter
                            text: root.course.periodLabel || "—"
                            font.pixelSize: (root.course.periodLabel && root.course.periodLabel.length >= 5) ? 12 : 14
                            font.weight: Font.Bold
                            font.features: ({ "tnum": 1 })
                            color: root.isExam
                                ? "#FDE68A"
                                : (root.isLab ? "#BAE6FD" : root.inkPrimary)
                        }

                        Text {
                            Layout.alignment: Qt.AlignHCenter
                            text: "节"
                            font.pixelSize: 10
                            font.weight: Font.Medium
                            color: root.isExam
                                ? Qt.rgba(0.99, 0.90, 0.54, 0.85)
                                : (root.isLab ? Qt.rgba(0.73, 0.90, 0.99, 0.85) : root.inkMuted)
                        }
                    }
                }

                // 右侧课程主体：课名 + 语义晶体胶囊 / 地点 + 教师
                ColumnLayout {
                    Layout.fillWidth: true
                    Layout.alignment: Qt.AlignVCenter
                    spacing: 4

                    RowLayout {
                        Layout.fillWidth: true
                        spacing: 6

                        Text {
                            Layout.fillWidth: true
                            text: root.course.name || ""
                            font.pixelSize: 14
                            font.weight: Font.DemiBold
                            lineHeight: 1.24
                            color: root.inkPrimary
                            wrapMode: Text.WordWrap
                            maximumLineCount: root.expanded ? 4 : 2
                            elide: Text.ElideRight
                        }

                        // 仅「实验 / 考试」点亮宝石玻璃微胶囊，常态「上课」隐藏降噪
                        Rectangle {
                            visible: root.showStatusBadge
                            Layout.alignment: Qt.AlignTop
                            Layout.topMargin: 1
                            implicitWidth: badgeText.implicitWidth + 12
                            implicitHeight: 19
                            radius: 9.5
                            color: root.isExam
                                ? Qt.rgba(1.0, 0.72, 0.18, 0.22)
                                : Qt.rgba(0.22, 0.74, 0.97, 0.22)
                            border.width: 1
                            border.color: root.isExam
                                ? Qt.rgba(1.0, 0.84, 0.42, 0.54)
                                : Qt.rgba(0.55, 0.85, 1.0, 0.54)

                            Text {
                                id: badgeText
                                anchors.centerIn: parent
                                text: root.course.status || ""
                                font.pixelSize: 11
                                font.weight: Font.DemiBold
                                color: root.isExam ? "#FDE68A" : "#BAE6FD"
                            }
                        }
                    }

                    // 地点优先 + 教师右对齐
                    RowLayout {
                        Layout.fillWidth: true
                        visible: (root.course.location && root.course.location.length > 0)
                            || (root.course.teacher && root.course.teacher.length > 0)
                        spacing: 10

                        Text {
                            Layout.fillWidth: true
                            visible: root.course.location && root.course.location.length > 0
                            text: root.course.location || ""
                            font.pixelSize: 12
                            font.weight: Font.Normal
                            color: root.inkSecondary
                            elide: Text.ElideRight
                            maximumLineCount: 1
                        }

                        Text {
                            visible: root.course.teacher && root.course.teacher.length > 0
                            text: root.course.teacher || ""
                            font.pixelSize: 12
                            font.weight: Font.Normal
                            color: root.inkMuted
                            horizontalAlignment: Text.AlignRight
                        }
                    }
                }
            }

            // 原地展开微凹详情槽
            Item {
                id: detailDrawer
                Layout.fillWidth: true
                Layout.leftMargin: 69
                Layout.rightMargin: 8
                Layout.bottomMargin: root.expanded ? 8 : 0
                implicitHeight: root.expanded ? detailWell.implicitHeight : 0
                clip: true
                opacity: root.expanded ? 1.0 : 0.0

                Behavior on implicitHeight {
                    enabled: !root.reducedMotion
                    NumberAnimation { duration: 185; easing.type: Easing.OutCubic }
                }
                Behavior on opacity {
                    enabled: !root.reducedMotion
                    NumberAnimation { duration: 145 }
                }

                Rectangle {
                    id: detailWell
                    anchors.left: parent.left
                    anchors.right: parent.right
                    implicitHeight: detailCol.implicitHeight + 12
                    radius: 10
                    color: Qt.rgba(0, 0, 0, 0.22)
                    border.width: 1
                    border.color: Qt.rgba(1, 1, 1, 0.18)

                    ColumnLayout {
                        id: detailCol
                        anchors.left: parent.left
                        anchors.right: parent.right
                        anchors.top: parent.top
                        anchors.margins: 6
                        anchors.leftMargin: 9
                        anchors.rightMargin: 9
                        spacing: 3

                        Text {
                            Layout.fillWidth: true
                            visible: root.course.location && root.course.location.length > 0
                            text: "地点  " + root.course.location
                            font.pixelSize: 12
                            color: root.inkSecondary
                            wrapMode: Text.WordWrap
                        }

                        Text {
                            Layout.fillWidth: true
                            visible: (root.course.weeksLabel && root.course.weeksLabel.length > 0)
                                || (root.course.periodFull && root.course.periodFull.length > 0)
                            text: {
                                var items = [];
                                if (root.course.weeksLabel) items.push(root.course.weeksLabel);
                                if (root.course.periodFull) items.push(root.course.periodFull);
                                if (root.course.status) items.push(root.course.status);
                                return items.join("   ");
                            }
                            font.pixelSize: 11
                            color: root.inkMuted
                            wrapMode: Text.WordWrap
                        }
                    }
                }
            }
        }

        // 极淡发丝分割线（绝不打断左侧节次纵脊）
        Rectangle {
            visible: root.showDivider && !root.expanded && !root.rowHovered
            anchors.left: parent.left
            anchors.leftMargin: 69
            anchors.right: parent.right
            anchors.rightMargin: 8
            anchors.bottom: parent.bottom
            height: 1
            color: Qt.rgba(1, 1, 1, 0.13)
        }
    }

    MouseArea {
        anchors.fill: mainSurface
        cursorShape: root.hasDetails ? Qt.PointingHandCursor : Qt.ArrowCursor
        onClicked: {
            if (root.hasDetails) {
                root.expanded = !root.expanded;
            }
        }
    }

    HoverHandler {
        onHoveredChanged: root.rowHovered = hovered
        onPointChanged: root.pointerX = point.position.x
    }
}
