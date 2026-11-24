#include "GlassWindowHelper.h"

#include <QColor>
#include <QEvent>
#include <QPolygon>
#include <QVariant>
#include <QtMath>

#ifdef HAVE_KF6_WINDOWSYSTEM
#include <KWindowEffects>
#endif

namespace {

constexpr double kSquirclePower = 3.8;
constexpr int kCornerSegments = 14;

} // namespace

GlassWindowHelper::GlassWindowHelper(QObject *parent)
    : QObject(parent)
{
    m_reapplyTimer.setSingleShot(true);
    m_reapplyTimer.setInterval(0);
    connect(&m_reapplyTimer, &QTimer::timeout, this, &GlassWindowHelper::reapplyNow);

    // 二次兜底：PlasmaQuick::Dialog 首次映射到 Wayland 时可能以队列信号触发 updateTheme() 清除模糊。
    m_settleTimer.setSingleShot(true);
    m_settleTimer.setInterval(55);
    connect(&m_settleTimer, &QTimer::timeout, this, &GlassWindowHelper::reapplyNow);
}

GlassWindowHelper::~GlassWindowHelper()
{
    if (m_window) {
        m_window->removeEventFilter(this);
    }
}

bool GlassWindowHelper::blurEffectAvailable() const
{
#ifdef HAVE_KF6_WINDOWSYSTEM
    return KWindowEffects::isEffectAvailable(KWindowEffects::BlurBehind);
#else
    return false;
#endif
}

QRegion GlassWindowHelper::buildSquircleRegion(int width,
                                               int height,
                                               qreal cornerRadius,
                                               qreal insetPad)
{
    // 向内微缩 0.5px，确保 KWin 模糊严格落在镜面切边以内，绝不溢出到透明阴影边距。
    const double pad = qMax(0.0, static_cast<double>(insetPad)) + 0.5;
    const double left = pad;
    const double top = pad;
    const double right = static_cast<double>(width) - pad;
    const double bottom = static_cast<double>(height) - pad;
    const double gw = right - left;
    const double gh = bottom - top;
    if (gw < 4.0 || gh < 4.0) {
        return QRegion();
    }

    const double r = qBound(0.0, static_cast<double>(cornerRadius), qMin(gw, gh) * 0.5);
    if (r < 1.0) {
        return QRegion(qRound(left), qRound(top), qRound(gw), qRound(gh));
    }

    const double exponent = 2.0 / kSquirclePower;
    QPolygon poly;
    poly.reserve((kCornerSegments + 1) * 4);

    auto appendCorner = [&](double cx, double cy, double sx, double sy, bool reverse) {
        for (int i = 0; i <= kCornerSegments; ++i) {
            const int idx = reverse ? (kCornerSegments - i) : i;
            const double theta = (M_PI * 0.5) * static_cast<double>(idx) / static_cast<double>(kCornerSegments);
            const double dx = r * std::pow(qMax(0.0, std::cos(theta)), exponent);
            const double dy = r * std::pow(qMax(0.0, std::sin(theta)), exponent);
            poly.append(QPoint(qRound(cx + sx * dx), qRound(cy + sy * dy)));
        }
    };

    // 顺时针拼接四个 Apple 超椭圆圆角：右上 -> 右下 -> 左下 -> 左上
    appendCorner(right - r, top + r, +1.0, -1.0, true);
    appendCorner(right - r, bottom - r, +1.0, +1.0, false);
    appendCorner(left + r, bottom - r, -1.0, +1.0, true);
    appendCorner(left + r, top + r, -1.0, -1.0, false);

    return QRegion(poly);
}

void GlassWindowHelper::applyGlassEffect(QWindow *window,
                                         qreal width,
                                         qreal height,
                                         qreal cornerRadius,
                                         qreal insetPad,
                                         bool isDark,
                                         bool enabled)
{
    if (!window) {
        return;
    }

    if (m_window != window) {
        if (m_window) {
            m_window->removeEventFilter(this);
        }
        m_window = window;
        m_window->installEventFilter(this);
    }

    m_fallbackWidth = width;
    m_fallbackHeight = height;
    m_cornerRadius = cornerRadius;
    m_insetPad = insetPad;
    m_isDark = isDark;
    m_enabled = enabled;

    reapplyNow();
    scheduleReapply(true);
}

bool GlassWindowHelper::eventFilter(QObject *watched, QEvent *event)
{
    if (watched == m_window && event) {
        switch (event->type()) {
        case QEvent::Show:
        case QEvent::Expose:
        case QEvent::PlatformSurface:
        case QEvent::Resize:
            scheduleReapply(true);
            break;
        case QEvent::Move:
        case QEvent::WindowActivate:
            scheduleReapply(false);
            break;
        default:
            break;
        }
    }
    return QObject::eventFilter(watched, event);
}

void GlassWindowHelper::scheduleReapply(bool withSettlePass)
{
    if (!m_window) {
        return;
    }
    m_reapplyTimer.start();
    if (withSettlePass) {
        m_settleTimer.start();
    }
}

void GlassWindowHelper::reapplyNow()
{
    if (!m_window) {
        return;
    }

    m_window->setProperty("color", QColor(Qt::transparent));

#ifdef HAVE_KF6_WINDOWSYSTEM
    if (!m_enabled) {
        KWindowEffects::enableBlurBehind(m_window, false);
        KWindowEffects::enableBackgroundContrast(m_window, false);
        m_window->requestUpdate();
        return;
    }

    const int w = m_window->width() > 4 ? m_window->width() : qRound(m_fallbackWidth);
    const int h = m_window->height() > 4 ? m_window->height() : qRound(m_fallbackHeight);
    const QRegion region = buildSquircleRegion(w, h, m_cornerRadius, m_insetPad);
    if (region.isEmpty()) {
        return;
    }

    KWindowEffects::enableBlurBehind(m_window, true, region);

    // 对标 rdev/liquid-glass-react 的高饱和通透液态玻璃调校：
    // 高饱和（1.62）让背景壁纸与暗色窗口透出宝石般丰富色彩；轻微降亮（0.88）压住刺眼纯白过曝，
    // 让透明玻璃上的纯白文字在亮色壁纸、深色终端与白底窗口下都保持极高对比度与通透质感。
    const qreal contrast = 1.05;
    const qreal intensity = m_isDark ? 0.84 : 0.88;
    const qreal saturation = 1.62;
    KWindowEffects::enableBackgroundContrast(m_window, true, contrast, intensity, saturation, region);

    // 关键：在 Wayland 下主动触发一次帧更新，促使 Qt 立刻提交 wl_surface.commit() 使 KWin 模糊即刻生效
    m_window->requestUpdate();
#endif
}
