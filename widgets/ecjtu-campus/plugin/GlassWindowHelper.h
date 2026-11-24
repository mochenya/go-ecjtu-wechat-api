#pragma once

#include <QObject>
#include <QPointer>
#include <QRegion>
#include <QTimer>
#include <QWindow>
#include <QtQml/qqmlregistration.h>

// 为透明 PlasmaCore.Dialog 提供精准超椭圆轮廓的 KWin 活色模糊与饱和度增强，
// 并通过事件过滤器在 DialogPrivate::updateTheme() 清除背景特效后立即重新提交给 KWin。
class GlassWindowHelper : public QObject
{
    Q_OBJECT
    QML_ELEMENT

public:
    explicit GlassWindowHelper(QObject *parent = nullptr);
    ~GlassWindowHelper() override;

    // 查询当前 KWin 合成器是否支持背景模糊。
    Q_INVOKABLE bool blurEffectAvailable() const;

    // 绑定精准 Squircle 区域到 KWin BlurBehind 与 BackgroundContrast，
    // 排除外围透明柔影区，并天然兼容 kwin-effects-glass 等第三方玻璃插件。
    Q_INVOKABLE void applyGlassEffect(QWindow *window,
                                      qreal width,
                                      qreal height,
                                      qreal cornerRadius,
                                      qreal insetPad,
                                      bool isDark,
                                      bool enabled);

    // 构造与 GLSL sdSquircleBox(power=3.8) 对齐的超椭圆 QRegion。
    static QRegion buildSquircleRegion(int width,
                                       int height,
                                       qreal cornerRadius,
                                       qreal insetPad);

protected:
    bool eventFilter(QObject *watched, QEvent *event) override;

private:
    void scheduleReapply(bool withSettlePass);
    void reapplyNow();

    QPointer<QWindow> m_window;
    qreal m_fallbackWidth = 0;
    qreal m_fallbackHeight = 0;
    qreal m_cornerRadius = 28;
    qreal m_insetPad = 14;
    bool m_isDark = false;
    bool m_enabled = true;

    QTimer m_reapplyTimer;
    QTimer m_settleTimer;
};
