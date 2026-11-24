import QtQuick

Item {
    id: root

    property string currentKey: "today"
    property bool isDark: true
    property bool reducedMotion: false
    property bool reducedTransparency: false

    // 传入桌面壁纸即可持续驱动实时液态局部折射与 RGB 色散
    property Item peekSourceItem: null
    property bool peekActive: false
    property int snapshotTrigger: 0

    signal activated(string key)

    implicitHeight: 38
    implicitWidth: 336
    focus: true

    readonly property int activeIndex: currentKey === "tomorrow" ? 1 : 0
    readonly property real trackPadding: 3
    readonly property real pillWidth: Math.max(24, (width - trackPadding * 2) * 0.5)
    readonly property real pillHeight: Math.max(22, height - trackPadding * 2)
    readonly property real targetPillX: trackPadding + activeIndex * pillWidth

    property bool dragging: false
    property real dragPillX: targetPillX
    property real dragStartMouseX: 0
    property real dragStartPillX: 0
    property real lastPillX: targetPillX
    property real velocityStretch: 0.0
    property real impulseStretch: 0.0
    readonly property real stretchAmount: Math.min(1.0, Math.max(impulseStretch, velocityStretch))

    property vector2d pointerNorm: Qt.vector2d(0.5, 0.2)
    property bool hovered: false

    function refreshSnapshot() {
        if (pillContainer.canSnapshot && !pillContainer.livePeek) {
            peekCapture.scheduleUpdate();
        }
    }

    Keys.onLeftPressed: activated("today")
    Keys.onRightPressed: activated("tomorrow")

    onActiveIndexChanged: {
        if (!reducedMotion) {
            stretchAnim.restart();
        }
        Qt.callLater(refreshSnapshot);
    }
    onIsDarkChanged: Qt.callLater(refreshSnapshot)
    onSnapshotTriggerChanged: Qt.callLater(refreshSnapshot)
    onWidthChanged: Qt.callLater(refreshSnapshot)
    Component.onCompleted: Qt.callLater(refreshSnapshot)

    SequentialAnimation {
        id: stretchAnim
        NumberAnimation {
            target: root
            property: "impulseStretch"
            to: 1.0
            duration: 90
            easing.type: Easing.OutQuad
        }
        NumberAnimation {
            target: root
            property: "impulseStretch"
            to: 0.0
            duration: 210
            easing.type: Easing.OutCubic
        }
    }

    // 轻盈微凹透明玻璃轨道（同心圆角 radius = 14，零发灰暗块）
    Rectangle {
        id: track
        anchors.fill: parent
        radius: 14
        color: Qt.rgba(0, 0, 0, 0.16)
        border.width: 1
        border.color: Qt.rgba(1, 1, 1, 0.20)

        // 中央极轻刻度线
        Rectangle {
            anchors.centerIn: parent
            width: 1
            height: 12
            radius: 0.5
            color: Qt.rgba(1, 1, 1, 0.16)
        }
    }

    // 1x1 透明占位纹理
    Rectangle {
        id: emptyTex
        width: 1
        height: 1
        color: "transparent"
        visible: false
    }

    ShaderEffectSource {
        id: emptySource
        sourceItem: emptyTex
        live: false
        visible: false
    }

    // 悬浮通透水晶液滴滑块（果冻形变 + 横向拖拽 + 实时折射）
    Item {
        id: pillContainer
        y: root.trackPadding
        width: root.pillWidth
        height: root.pillHeight
        x: root.dragging ? root.dragPillX : root.targetPillX

        onXChanged: {
            var dx = Math.abs(x - root.lastPillX);
            root.lastPillX = x;
            if (!root.reducedMotion) {
                root.velocityStretch = Math.min(1.0, dx / 18.0);
                velocityDecay.restart();
            }
        }

        NumberAnimation {
            id: velocityDecay
            target: root
            property: "velocityStretch"
            to: 0.0
            duration: 180
            easing.type: Easing.OutCubic
        }

        // 果冻面积守恒形变
        transform: Scale {
            origin.x: pillContainer.width * 0.5
            origin.y: pillContainer.height * 0.5
            xScale: root.reducedMotion ? 1.0 : (1.0 + root.stretchAmount * 0.055)
            yScale: root.reducedMotion ? 1.0 : (1.0 - root.stretchAmount * 0.042)
        }

        Behavior on x {
            enabled: !root.reducedMotion && !root.dragging
            SpringAnimation {
                spring: 5.8
                damping: 0.42
                epsilon: 0.2
            }
        }

        readonly property bool useShader: !root.reducedTransparency && GraphicsInfo.api !== GraphicsInfo.Software
        readonly property bool canSnapshot: useShader
            && root.peekSourceItem !== null
            && width > 4
            && height > 4
        readonly property bool livePeek: canSnapshot
            && (root.peekActive || root.dragging || root.stretchAmount > 0.015)

        readonly property point peekOrigin: canSnapshot
            ? pillContainer.mapToItem(root.peekSourceItem, 0, 0)
            : Qt.point(0, 0)

        ShaderEffectSource {
            id: peekCapture
            visible: false
            sourceItem: pillContainer.canSnapshot ? root.peekSourceItem : null
            sourceRect: Qt.rect(pillContainer.peekOrigin.x,
                                pillContainer.peekOrigin.y,
                                pillContainer.width,
                                pillContainer.height)
            live: pillContainer.livePeek
            smooth: true
        }

        ShaderEffect {
            anchors.fill: parent
            visible: pillContainer.useShader
            property variant source: pillContainer.canSnapshot ? peekCapture : emptySource
            property vector2d itemSize: Qt.vector2d(Math.max(4, pillContainer.width),
                                                    Math.max(4, pillContainer.height))
            property real cornerRadius: 11.0
            property real hasBackdrop: pillContainer.canSnapshot ? 1.0 : 0.0
            property real stretch: root.stretchAmount
            property real isDark: root.isDark ? 1.0 : 0.0
            property real hoverAmount: root.hovered || root.dragging ? 1.0 : 0.0
            property vector2d pointerNorm: root.pointerNorm
            fragmentShader: Qt.resolvedUrl("../shaders/liquid_pill.frag.qsb")
        }

        Rectangle {
            anchors.fill: parent
            visible: !pillContainer.useShader
            radius: 11
            color: Qt.rgba(1, 1, 1, 0.22)
            border.width: 1
            border.color: Qt.rgba(1, 1, 1, 0.40)
        }
    }

    // 清晰高对比纯白文字层
    Row {
        anchors.fill: parent
        anchors.margins: root.trackPadding

        Item {
            width: root.pillWidth
            height: root.pillHeight

            readonly property bool isSelected: root.dragging
                ? (root.dragPillX < root.trackPadding + root.pillWidth * 0.5)
                : (root.activeIndex === 0)

            Text {
                anchors.centerIn: parent
                text: "今天"
                font.pixelSize: 13
                font.weight: parent.isSelected ? Font.DemiBold : Font.Medium
                color: parent.isSelected ? "#FFFFFF" : Qt.rgba(1, 1, 1, 0.68)
                Behavior on color { ColorAnimation { duration: 140 } }
            }
        }

        Item {
            width: root.pillWidth
            height: root.pillHeight

            readonly property bool isSelected: root.dragging
                ? (root.dragPillX >= root.trackPadding + root.pillWidth * 0.5)
                : (root.activeIndex === 1)

            Text {
                anchors.centerIn: parent
                text: "明天"
                font.pixelSize: 13
                font.weight: parent.isSelected ? Font.DemiBold : Font.Medium
                color: parent.isSelected ? "#FFFFFF" : Qt.rgba(1, 1, 1, 0.68)
                Behavior on color { ColorAnimation { duration: 140 } }
            }
        }
    }

    // 统一支持点击切换与液态横向拖拽滑动
    MouseArea {
        id: dragArea
        anchors.fill: parent
        hoverEnabled: true
        cursorShape: root.dragging ? Qt.ClosedHandCursor : Qt.PointingHandCursor

        onPressed: function (mouse) {
            root.dragStartMouseX = mouse.x;
            root.dragStartPillX = pillContainer.x;
            var hitActive = mouse.x >= pillContainer.x && mouse.x <= pillContainer.x + pillContainer.width;
            if (hitActive) {
                root.dragPillX = pillContainer.x;
                root.dragging = true;
            }
        }

        onPositionChanged: function (mouse) {
            if (root.width > 0 && root.height > 0) {
                root.pointerNorm = Qt.vector2d(
                    Math.max(0.0, Math.min(1.0, mouse.x / root.width)),
                    Math.max(0.0, Math.min(1.0, mouse.y / root.height))
                );
            }
            if (!pressed) {
                return;
            }
            var delta = mouse.x - root.dragStartMouseX;
            if (!root.dragging && Math.abs(delta) > 5) {
                root.dragPillX = root.dragStartPillX;
                root.dragging = true;
            }
            if (root.dragging) {
                var minX = root.trackPadding;
                var maxX = root.trackPadding + root.pillWidth;
                root.dragPillX = Math.max(minX, Math.min(maxX, root.dragStartPillX + delta));
            }
        }

        onReleased: function (mouse) {
            var delta = Math.abs(mouse.x - root.dragStartMouseX);
            if (root.dragging && delta >= 6) {
                var target = (root.dragPillX >= root.trackPadding + root.pillWidth * 0.5)
                    ? "tomorrow"
                    : "today";
                root.dragging = false;
                root.activated(target);
            } else {
                root.dragging = false;
                root.activated(mouse.x >= root.width * 0.5 ? "tomorrow" : "today");
            }
        }

        onCanceled: {
            root.dragging = false;
        }

        onEntered: root.hovered = true
        onExited: root.hovered = false
    }
}
