import QtQuick

Item {
    id: root

    // 若运行于桌面壁纸之上（或预览窗口提供壁纸 Item），启用 3-pass 丝滑柔焦 + 边缘液态折射/色散管线；
    // 为 null 时运行于透明 PlasmaCore.Dialog + KWin 活色模糊之上，提供 rdev/liquid-glass-react 风格通透玻璃壳。
    property Item backdropItem: null
    property bool isDark: true
    property real cornerRadius: 28
    property real insetPad: 0
    property bool reducedTransparency: false
    property bool reducedMotion: false

    // 对标 rdev/liquid-glass-react 的清晰边缘折射与刀锋镜面高光参数
    property real bevelWidth: 18.0
    property real glassThickness: 28.0
    property real ior: 1.50
    property real dispersion: 0.055
    property real specularStrength: 1.0

    readonly property bool shaderSupported: GraphicsInfo.api !== GraphicsInfo.Software
    readonly property bool useShader: !reducedTransparency && shaderSupported
    readonly property bool desktopBackdropActive: useShader && backdropItem !== null && width > 8 && height > 8

    property point mappedOrigin: Qt.point(0, 0)
    property vector2d pointerNorm: Qt.vector2d(0.30, 0.24)

    function syncBackdropRect() {
        if (!desktopBackdropActive) {
            return;
        }
        mappedOrigin = root.mapToItem(backdropItem, 0, 0);
    }

    onBackdropItemChanged: syncBackdropRect()
    onXChanged: syncBackdropRect()
    onYChanged: syncBackdropRect()
    onWidthChanged: syncBackdropRect()
    onHeightChanged: syncBackdropRect()
    Component.onCompleted: syncBackdropRect()

    // 1x1 透明占位纹理
    Rectangle {
        id: emptyTexture
        width: 1
        height: 1
        color: "transparent"
        visible: false
    }

    ShaderEffectSource {
        id: emptySource
        sourceItem: emptyTexture
        live: false
        hideSource: false
        visible: false
    }

    // 通道 1：全分辨率清晰壁纸（供边缘 10px 玻璃唇边微弱折射张力）
    ShaderEffectSource {
        id: sharpWallpaperCrop
        visible: false
        sourceItem: root.desktopBackdropActive ? root.backdropItem : null
        sourceRect: Qt.rect(root.mappedOrigin.x, root.mappedOrigin.y, root.width, root.height)
        textureSize: Qt.size(Math.max(4, Math.round(root.width)),
                             Math.max(4, Math.round(root.height)))
        live: root.desktopBackdropActive
        smooth: true
    }

    // 通道 2：3-Pass 奶油级 Dual Kawase 模糊（0.38x 降采样 + 三级渐进扩散，复刻参考图的丝滑光晕景深）
    readonly property int blurTexW: Math.max(4, Math.round(root.width * 0.38))
    readonly property int blurTexH: Math.max(4, Math.round(root.height * 0.38))

    ShaderEffectSource {
        id: softWallpaperCrop
        visible: false
        sourceItem: root.desktopBackdropActive ? root.backdropItem : null
        sourceRect: Qt.rect(root.mappedOrigin.x, root.mappedOrigin.y, root.width, root.height)
        textureSize: Qt.size(root.blurTexW, root.blurTexH)
        live: root.desktopBackdropActive
        smooth: true
    }

    ShaderEffect {
        id: blurPass1
        visible: false
        width: root.blurTexW
        height: root.blurTexH
        property variant source: softWallpaperCrop
        property vector2d texelSize: Qt.vector2d(1.0 / width, 1.0 / height)
        property real offset: 1.15
        fragmentShader: Qt.resolvedUrl("../shaders/kawase_blur.frag.qsb")
    }

    ShaderEffectSource {
        id: blurPass1Source
        visible: false
        sourceItem: root.desktopBackdropActive ? blurPass1 : null
        live: root.desktopBackdropActive
        smooth: true
    }

    ShaderEffect {
        id: blurPass2
        visible: false
        width: root.blurTexW
        height: root.blurTexH
        property variant source: blurPass1Source
        property vector2d texelSize: Qt.vector2d(1.0 / width, 1.0 / height)
        property real offset: 2.25
        fragmentShader: Qt.resolvedUrl("../shaders/kawase_blur.frag.qsb")
    }

    ShaderEffectSource {
        id: blurPass2Source
        visible: false
        sourceItem: root.desktopBackdropActive ? blurPass2 : null
        live: root.desktopBackdropActive
        smooth: true
    }

    ShaderEffect {
        id: blurPass3
        visible: false
        width: root.blurTexW
        height: root.blurTexH
        property variant source: blurPass2Source
        property vector2d texelSize: Qt.vector2d(1.0 / width, 1.0 / height)
        property real offset: 3.45
        fragmentShader: Qt.resolvedUrl("../shaders/kawase_blur.frag.qsb")
    }

    ShaderEffectSource {
        id: blurPass3Source
        visible: false
        sourceItem: root.desktopBackdropActive ? blurPass3 : null
        live: root.desktopBackdropActive
        smooth: true
    }

    // Clear Liquid Glass 主片元着色器
    ShaderEffect {
        id: glassShader
        anchors.fill: parent
        visible: root.useShader

        property variant sharpSource: root.desktopBackdropActive ? sharpWallpaperCrop : emptySource
        property variant softSource: root.desktopBackdropActive ? blurPass3Source : emptySource
        property vector2d itemSize: Qt.vector2d(Math.max(4, root.width), Math.max(4, root.height))
        property real cornerRadius: root.cornerRadius
        property real insetPad: root.insetPad
        property real hasBackdrop: root.desktopBackdropActive ? 1.0 : 0.0
        property real bevelWidth: root.bevelWidth
        property real glassThickness: root.glassThickness
        property real ior: root.ior
        property real dispersion: root.dispersion
        property real specularStrength: root.specularStrength
        property real isDark: root.isDark ? 1.0 : 0.0
        property vector2d pointerNorm: root.pointerNorm
        // 非预乘 RGBA：极轻薄中性烟晶滤膜（桌面 14% / 弹窗 20%），绝不把暗色终端染成水泥灰或把彩壁纸洗白
        property vector4d tintColor: root.desktopBackdropActive
            ? Qt.vector4d(0.06, 0.09, 0.15, 0.14)
            : Qt.vector4d(0.06, 0.09, 0.15, 0.20)

        fragmentShader: Qt.resolvedUrl("../shaders/liquid_surface.frag.qsb")
    }

    // 无障碍降低透明度或软件渲染时的清晰回退底板
    Rectangle {
        anchors.fill: parent
        anchors.margins: root.insetPad
        visible: !root.useShader
        radius: root.cornerRadius
        color: Qt.rgba(0.09, 0.12, 0.18, 0.94)
        border.width: 1
        border.color: Qt.rgba(1, 1, 1, 0.24)
    }

    // 轻量鼠标环境光随动（实时驱动 135° 刀锋玻璃边流光角度，不拦截点击与滚动）
    HoverHandler {
        enabled: !root.reducedMotion && !root.reducedTransparency
        onPointChanged: {
            if (root.width > 0 && root.height > 0) {
                root.pointerNorm = Qt.vector2d(
                    Math.max(0.0, Math.min(1.0, point.position.x / root.width)),
                    Math.max(0.0, Math.min(1.0, point.position.y / root.height))
                );
            }
        }
    }
}
