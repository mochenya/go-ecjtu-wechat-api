import QtQuick
import QtQuick.Layouts
import org.kde.kirigami as Kirigami
import org.kde.plasma.core as PlasmaCore
import org.kde.plasma.plasmoid

import io.github.mochenya.ecjtucampus.backend 1.0 as CampusBackend
import "glass" as Glass

PlasmoidItem {
    id: root

    readonly property bool isDesktopWidget: Plasmoid.location === PlasmaCore.Types.Floating
    readonly property bool darkPalette: Kirigami.Theme.backgroundColor.hslLightness < 0.5
    readonly property bool reducedMotion: Kirigami.Units.longDuration === 0
    readonly property int popupShadowPad: 14

    // 桌面与面板统一去除 Plasma 默认厚白底框：
    // - 桌面常驻态由自研双频 GLSL 折射真实壁纸
    // - 面板弹窗态改用透明 PlasmaCore.Dialog(NoBackground) + C++ KWindowEffects 超椭圆活色模糊
    Plasmoid.backgroundHints: PlasmaCore.Types.NoBackground

    property Item compactAnchor: null
    property double lastPopupHideMs: 0

    toolTipMainText: "华交校园"
    toolTipSubText: "点击查看今明课表"

    CampusBackend.CourseBackend {
        id: courseBackend
    }

    CampusBackend.GlassWindowHelper {
        id: glassWindowHelper
    }

    function syncPopupGlass() {
        if (!liquidPopup.visible) {
            return;
        }
        glassWindowHelper.applyGlassEffect(
            liquidPopup,
            liquidPopup.width,
            liquidPopup.height,
            28,
            root.popupShadowPad,
            root.darkPalette,
            true
        );
    }

    function openOrTogglePopup() {
        if (root.isDesktopWidget) {
            return;
        }
        // 防止点击面板图标时因 hideOnWindowDeactivate 先隐藏再被 onClicked 立刻重新弹出
        if (!liquidPopup.visible && (Date.now() - root.lastPopupHideMs) < 220) {
            return;
        }
        if (liquidPopup.visible) {
            liquidPopup.visible = false;
            return;
        }
        if (root.compactAnchor) {
            liquidPopup.visualParent = root.compactAnchor;
        }
        courseBackend.openAndRefreshToday();
        liquidPopup.visible = true;
        syncPopupGlass();
        Qt.callLater(syncPopupGlass);
    }

    // 兼容快捷键触发展开：面板态自动重定向到自研透明液态玻璃弹窗
    onExpandedChanged: {
        if (!root.isDesktopWidget && expanded) {
            root.expanded = false;
            root.openOrTogglePopup();
        }
    }

    onDarkPaletteChanged: Qt.callLater(syncPopupGlass)

    Component.onCompleted: {
        if (isDesktopWidget) {
            courseBackend.openAndRefreshToday();
        }
    }

    // 面板态自研透明液态玻璃弹窗：彻底脱离 Breeze 80% 乳白底板，精确裁剪 KWin 活色模糊区域
    PlasmaCore.Dialog {
        id: liquidPopup
        visible: false
        backgroundHints: PlasmaCore.Dialog.NoBackground
        color: "transparent"
        visualParent: root.compactAnchor
        location: Plasmoid.location
        type: PlasmaCore.Dialog.AppletPopup
        hideOnWindowDeactivate: true
        floating: 6

        onVisibleChanged: {
            if (visible) {
                popupPanel.forceActiveFocus();
                root.syncPopupGlass();
                Qt.callLater(root.syncPopupGlass);
            } else {
                root.lastPopupHideMs = Date.now();
            }
        }
        onWidthChanged: root.syncPopupGlass()
        onHeightChanged: root.syncPopupGlass()

        mainItem: Item {
            width: popupPanel.implicitWidth
            height: popupPanel.implicitHeight

            CampusPanel {
                id: popupPanel
                anchors.fill: parent
                backend: courseBackend
                drawOuterFrame: true
                insetPad: root.popupShadowPad
                isDark: root.darkPalette
                reducedMotion: root.reducedMotion
                backdropItem: null
                focus: true

                Keys.onEscapePressed: liquidPopup.visible = false
            }
        }
    }

    // 面板常驻触发器：[GH] ECJTU（静默融于面板，悬停/展开浮现极清澈微光胶囊）
    compactRepresentation: Item {
        id: compactRoot

        readonly property bool isVerticalPanel: Plasmoid.formFactor === PlasmaCore.Types.Vertical
        readonly property bool activeVisual: compactMouse.containsMouse || liquidPopup.visible

        Layout.minimumWidth: isVerticalPanel ? Kirigami.Units.iconSizes.smallMedium : compactRow.implicitWidth + 18
        Layout.preferredWidth: isVerticalPanel ? Kirigami.Units.iconSizes.smallMedium : compactRow.implicitWidth + 18
        Layout.minimumHeight: Kirigami.Units.iconSizes.smallMedium

        Component.onCompleted: root.compactAnchor = compactRoot
        Component.onDestruction: {
            if (root.compactAnchor === compactRoot) {
                root.compactAnchor = null;
            }
        }

        // 清澈微光玻璃胶囊指示态（告别呆板灰方块）
        Rectangle {
            anchors.centerIn: parent
            width: Math.min(parent.width, compactRow.implicitWidth + 14)
            height: Math.min(parent.height - 4, 26)
            radius: height * 0.5
            opacity: compactRoot.activeVisual ? 1.0 : 0.0
            color: root.darkPalette
                ? Qt.rgba(1, 1, 1, liquidPopup.visible ? 0.16 : 0.10)
                : Qt.rgba(1, 1, 1, liquidPopup.visible ? 0.56 : 0.36)
            border.width: 1
            border.color: root.darkPalette
                ? Qt.rgba(1, 1, 1, liquidPopup.visible ? 0.32 : 0.18)
                : Qt.rgba(1, 1, 1, liquidPopup.visible ? 0.90 : 0.65)

            Behavior on opacity {
                enabled: !root.reducedMotion
                NumberAnimation { duration: 150; easing.type: Easing.OutQuad }
            }
        }

        RowLayout {
            id: compactRow
            anchors.centerIn: parent
            spacing: 5

            Glass.TintedIcon {
                source: Qt.resolvedUrl("../icons/github-mark.svg")
                iconSize: 15
                color: Kirigami.Theme.textColor
            }

            Text {
                visible: !compactRoot.isVerticalPanel
                text: "ECJTU"
                font.pixelSize: 12
                font.weight: Font.DemiBold
                font.letterSpacing: 0.5
                color: Kirigami.Theme.textColor
            }
        }

        MouseArea {
            id: compactMouse
            anchors.fill: parent
            hoverEnabled: true
            cursorShape: Qt.PointingHandCursor
            onClicked: root.openOrTogglePopup()
        }
    }

    // 桌面常驻主视图（直接穿透桌面壁纸，零阴影裁剪框）
    fullRepresentation: Item {
        Layout.minimumWidth: 344
        Layout.preferredWidth: desktopPanel.implicitWidth
        Layout.maximumWidth: 420
        Layout.minimumHeight: 240
        Layout.preferredHeight: desktopPanel.implicitHeight
        Layout.maximumHeight: 580

        CampusPanel {
            id: desktopPanel
            anchors.fill: parent
            backend: courseBackend
            drawOuterFrame: true
            insetPad: 0
            isDark: root.darkPalette
            reducedMotion: root.reducedMotion
            backdropItem: (root.isDesktopWidget && Plasmoid.containment && Plasmoid.containment.wallpaperGraphicsObject)
                ? Plasmoid.containment.wallpaperGraphicsObject
                : null
        }
    }
}
