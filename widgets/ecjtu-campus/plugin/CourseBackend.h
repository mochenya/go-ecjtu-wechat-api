#pragma once

#include <QObject>
#include <QPointer>
#include <QProcess>
#include <QString>
#include <QtQml/qqmlregistration.h>

class CourseBackend : public QObject
{
    Q_OBJECT
    QML_ELEMENT

    Q_PROPERTY(bool hasToken READ hasToken NOTIFY hasTokenChanged)
    Q_PROPERTY(State state READ state NOTIFY stateChanged)
    Q_PROPERTY(QString selectedDay READ selectedDay NOTIFY selectedDayChanged)
    Q_PROPERTY(QString activeDate READ activeDate NOTIFY activeDateChanged)
    Q_PROPERTY(QString rawJson READ rawJson NOTIFY rawJsonChanged)
    Q_PROPERTY(QString errorCode READ errorCode NOTIFY errorChanged)
    Q_PROPERTY(QString errorTitle READ errorTitle NOTIFY errorChanged)
    Q_PROPERTY(QString errorHint READ errorHint NOTIFY errorChanged)
    Q_PROPERTY(QString diagnosticDetail READ diagnosticDetail NOTIFY errorChanged)

public:
    enum class State {
        Idle = 0,
        MissingToken,
        Loading,
        Ready,
        Error,
    };
    Q_ENUM(State)

    explicit CourseBackend(QObject *parent = nullptr);
    ~CourseBackend() override;

    bool hasToken() const;
    State state() const;
    QString selectedDay() const;
    QString activeDate() const;
    QString rawJson() const;
    QString errorCode() const;
    QString errorTitle() const;
    QString errorHint() const;
    QString diagnosticDetail() const;

    // 弹窗展开时调用：检测环境变量，未配置时不启动查询；已配置则按东八区查询今天。
    Q_INVOKABLE void openAndRefreshToday();

    // 切换查询日（"today" 或 "tomorrow"）；同一天查询中重复点击不重启进程。
    Q_INVOKABLE void selectDay(const QString &day);

    // 重新查询当前选中日期。
    Q_INVOKABLE void retry();

    // 重新检测环境变量；存在凭证时自动查询当前选中日（默认为今天）。
    Q_INVOKABLE void recheckToken();

    // 供工具与测试复用：按 Asia/Shanghai 计算今天或明天日期（YYYY-MM-DD）。
    static QString shanghaiDateForDay(const QString &day);

    // 兜底脱敏：替换可能出现在 stderr 中的敏感 openid 为前 4 位加 ***。
    static QString sanitizeDiagnostic(const QString &text, const QString &secret);

signals:
    void hasTokenChanged();
    void stateChanged();
    void selectedDayChanged();
    void activeDateChanged();
    void rawJsonChanged();
    void errorChanged();

private:
    bool refreshTokenState();
    void startQueryForDate(const QString &date, bool forceRestart);
    void cancelRunningProcess();
    void setState(State nextState);
    void setError(const QString &code,
                  const QString &title,
                  const QString &hint,
                  const QString &diagnostic);
    void clearError();
    void clearResult();

    bool m_hasToken = false;
    State m_state = State::Idle;
    QString m_selectedDay = QStringLiteral("today");
    QString m_activeDate;
    QString m_rawJson;
    QString m_errorCode;
    QString m_errorTitle;
    QString m_errorHint;
    QString m_diagnosticDetail;

    quint64 m_requestSeq = 0;
    QPointer<QProcess> m_runningProcess;
    QString m_runningDate;
};
