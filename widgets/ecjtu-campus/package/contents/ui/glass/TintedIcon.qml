import QtQuick
import QtQuick.Effects

Item {
    id: root

    property url source
    property color color: "#ffffff"
    property int iconSize: 16

    readonly property bool useEffect: GraphicsInfo.api !== GraphicsInfo.Software

    implicitWidth: iconSize
    implicitHeight: iconSize
    width: iconSize
    height: iconSize

    Image {
        id: rawIcon
        anchors.fill: parent
        source: root.source
        sourceSize.width: Math.max(2, Math.round(root.width * 2))
        sourceSize.height: Math.max(2, Math.round(root.height * 2))
        fillMode: Image.PreserveAspectFit
        smooth: true
        mipmap: true
        visible: !root.useEffect
    }

    MultiEffect {
        anchors.fill: rawIcon
        visible: root.useEffect
        source: rawIcon
        colorization: 1.0
        colorizationColor: root.color
        brightness: 0.0
    }
}
