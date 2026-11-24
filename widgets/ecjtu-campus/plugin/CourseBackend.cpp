#include "CourseBackend.h"

#include <QDate>
#include <QDateTime>
#include <QTimeZone>
#include <QTimer>

namespace {

constexpr auto kTokenEnvName = "ECJTU_WEIXIN_TOKEN";
constexpr int kMaxDiagnosticChars = 480;

QString readRawToken()
{
    return qEnvironmentVariable(kTokenEnvName).trimmed();
}

} // namespace

CourseBackend::CourseBackend(QObject *parent)
    : QObject(parent)
{
    refreshTokenState();
    m_activeDate = shanghaiDateForDay(m_selectedDay);
}

CourseBackend::~CourseBackend()
{
    cancelRunningProcess();
}

bool CourseBackend::hasToken() const
{
    return m_hasToken;
}

CourseBackend::State CourseBackend::state() const
{
    return m_state;
}

QString CourseBackend::selectedDay() const
{
    return m_selectedDay;
}

QString CourseBackend::activeDate() const
{
    return m_activeDate;
}

QString CourseBackend::rawJson() const
{
    return m_rawJson;
}

QString CourseBackend::errorCode() const
{
    return m_errorCode;
}

QString CourseBackend::errorTitle() const
{
    return m_errorTitle;
}

QString CourseBackend::errorHint() const
{
    return m_errorHint;
}

QString CourseBackend::diagnosticDetail() const
{
    return m_diagnosticDetail;
}

QString CourseBackend::shanghaiDateForDay(const QString &day)
{
    const QTimeZone shanghaiTz(QByteArrayLiteral("Asia/Shanghai"));
    const QDateTime now = shanghaiTz.isValid()
        ? QDateTime::currentDateTimeUtc().toTimeZone(shanghaiTz)
        : QDateTime::currentDateTimeUtc().addSecs(8 * 3600);
    QDate target = now.date();
    if (day == QStringLiteral("tomorrow")) {
        target = target.addDays(1);
    }
    return target.toString(QStringLiteral("yyyy-MM-dd"));
}

QString CourseBackend::sanitizeDiagnostic(const QString &text, const QString &secret)
{
    QString out = text.trimmed();
    if (!secret.isEmpty() && secret.size() >= 4) {
        const QString masked = secret.left(4) + QStringLiteral("***");
        out.replace(secret, masked);
    }
    if (out.size() > kMaxDiagnosticChars) {
        out = out.left(kMaxDiagnosticChars) + QStringLiteral("…");
    }
    return out;
}

bool CourseBackend::refreshTokenState()
{
    const bool present = !readRawToken().isEmpty();
    if (m_hasToken != present) {
        m_hasToken = present;
        emit hasTokenChanged();
    }
    return present;
}

void CourseBackend::openAndRefreshToday()
{
    if (m_selectedDay != QStringLiteral("today")) {
        m_selectedDay = QStringLiteral("today");
        emit selectedDayChanged();
    }

    const QString today = shanghaiDateForDay(QStringLiteral("today"));
    if (m_activeDate != today) {
        m_activeDate = today;
        emit activeDateChanged();
    }

    if (!refreshTokenState()) {
        cancelRunningProcess();
        ++m_requestSeq;
        clearResult();
        clearError();
        setState(State::MissingToken);
        return;
    }

    startQueryForDate(today, false);
}

void CourseBackend::selectDay(const QString &day)
{
    const QString normalized = (day == QStringLiteral("tomorrow"))
        ? QStringLiteral("tomorrow")
        : QStringLiteral("today");

    if (m_selectedDay != normalized) {
        m_selectedDay = normalized;
        emit selectedDayChanged();
    }

    const QString targetDate = shanghaiDateForDay(normalized);
    if (m_activeDate != targetDate) {
        m_activeDate = targetDate;
        emit activeDateChanged();
    }

    if (!refreshTokenState()) {
        cancelRunningProcess();
        ++m_requestSeq;
        clearResult();
        clearError();
        setState(State::MissingToken);
        return;
    }

    startQueryForDate(targetDate, false);
}

void CourseBackend::retry()
{
    const QString targetDate = shanghaiDateForDay(m_selectedDay);
    if (m_activeDate != targetDate) {
        m_activeDate = targetDate;
        emit activeDateChanged();
    }

    if (!refreshTokenState()) {
        cancelRunningProcess();
        ++m_requestSeq;
        clearResult();
        clearError();
        setState(State::MissingToken);
        return;
    }

    startQueryForDate(targetDate, true);
}

void CourseBackend::recheckToken()
{
    if (!refreshTokenState()) {
        cancelRunningProcess();
        ++m_requestSeq;
        clearResult();
        clearError();
        setState(State::MissingToken);
        return;
    }

    const QString targetDate = shanghaiDateForDay(m_selectedDay);
    if (m_activeDate != targetDate) {
        m_activeDate = targetDate;
        emit activeDateChanged();
    }
    startQueryForDate(targetDate, true);
}

void CourseBackend::startQueryForDate(const QString &date, bool forceRestart)
{
    // 同一天正在查询且未要求强制重查时，忽略重复触发。
    if (!forceRestart
        && m_state == State::Loading
        && m_runningProcess
        && m_runningProcess->state() != QProcess::NotRunning
        && m_runningDate == date) {
        return;
    }

    // 切换日期或强制重查：递增序号，确保旧进程的任何输出都不会覆盖当前选中日。
    cancelRunningProcess();
    const quint64 reqSeq = ++m_requestSeq;
    m_runningDate = date;

    clearError();
    setState(State::Loading);

    auto *proc = new QProcess(this);
    m_runningProcess = proc;

    connect(proc, &QProcess::errorOccurred, this,
            [this, proc, reqSeq](QProcess::ProcessError procErr) {
                if (reqSeq != m_requestSeq) {
                    return;
                }
                if (procErr == QProcess::FailedToStart) {
                    const QString detail = sanitizeDiagnostic(proc->errorString(), readRawToken());
                    m_runningProcess = nullptr;
                    m_runningDate.clear();
                    clearResult();
                    setError(QStringLiteral("not_found"),
                             QStringLiteral("未找到 ecjtu 命令"),
                             QStringLiteral("请确认 ecjtu 已安装并位于 Plasma 会话的 PATH 中。"),
                             detail);
                    setState(State::Error);
                    proc->deleteLater();
                }
            });

    connect(proc, &QProcess::finished, this,
            [this, proc, reqSeq](int exitCode, QProcess::ExitStatus exitStatus) {
                const QByteArray stdoutBytes = proc->readAllStandardOutput();
                const QByteArray stderrBytes = proc->readAllStandardError();
                proc->deleteLater();

                if (reqSeq != m_requestSeq) {
                    return;
                }

                if (m_runningProcess == proc) {
                    m_runningProcess = nullptr;
                    m_runningDate.clear();
                }

                const QString secret = readRawToken();
                const QString diag = sanitizeDiagnostic(QString::fromUtf8(stderrBytes), secret);

                if (exitStatus != QProcess::NormalExit) {
                    clearResult();
                    setError(QStringLiteral("crashed"),
                             QStringLiteral("查询进程异常中断"),
                             QStringLiteral("ecjtu 进程未能正常结束，请稍后重试。"),
                             diag);
                    setState(State::Error);
                    return;
                }

                if (exitCode == 0) {
                    const QString jsonText = QString::fromUtf8(stdoutBytes).trimmed();
                    clearError();
                    if (m_rawJson != jsonText) {
                        m_rawJson = jsonText;
                        emit rawJsonChanged();
                    }
                    setState(State::Ready);
                    return;
                }

                clearResult();
                switch (exitCode) {
                case 2:
                    setError(QStringLiteral("invalid_input"),
                             QStringLiteral("查询参数或环境配置无效"),
                             QStringLiteral("请检查 ECJTU_WEIXIN_TOKEN 与相关环境变量格式。"),
                             diag);
                    break;
                case 3:
                    setError(QStringLiteral("not_bound"),
                             QStringLiteral("教务凭证未绑定或已失效"),
                             QStringLiteral("请前往华东交通大学教务微信公众号重新绑定后再试。"),
                             diag);
                    break;
                case 4:
                    setError(QStringLiteral("upstream"),
                             QStringLiteral("教务系统暂时无法访问"),
                             QStringLiteral("连接超时或学校服务器返回异常，请稍后重试。"),
                             diag);
                    break;
                case 5:
                    setError(QStringLiteral("parse"),
                             QStringLiteral("课表页面解析失败"),
                             QStringLiteral("上游页面结构可能已变更，请检查 ecjtu 更新。"),
                             diag);
                    break;
                default:
                    setError(QStringLiteral("unknown"),
                             QStringLiteral("课表查询失败"),
                             QStringLiteral("ecjtu 返回异常退出码 %1，请查看诊断信息。").arg(exitCode),
                             diag);
                    break;
                }
                setState(State::Error);
            });

    // 直接启动程序名，不经过 shell，不把 token 放进命令行参数。
    proc->start(QStringLiteral("ecjtu"),
                {QStringLiteral("course"), QStringLiteral("--date"), date});
}

void CourseBackend::cancelRunningProcess()
{
    if (!m_runningProcess) {
        m_runningDate.clear();
        return;
    }

    QProcess *oldProc = m_runningProcess;
    m_runningProcess = nullptr;
    m_runningDate.clear();

    oldProc->disconnect(this);
    if (oldProc->state() == QProcess::NotRunning) {
        oldProc->deleteLater();
        return;
    }

    connect(oldProc, &QProcess::finished, oldProc, &QObject::deleteLater);
    oldProc->terminate();
    QPointer<QProcess> guard(oldProc);
    QTimer::singleShot(300, oldProc, [guard]() {
        if (guard && guard->state() != QProcess::NotRunning) {
            guard->kill();
        }
    });
}

void CourseBackend::setState(State nextState)
{
    if (m_state == nextState) {
        return;
    }
    m_state = nextState;
    emit stateChanged();
}

void CourseBackend::setError(const QString &code,
                             const QString &title,
                             const QString &hint,
                             const QString &diagnostic)
{
    m_errorCode = code;
    m_errorTitle = title;
    m_errorHint = hint;
    m_diagnosticDetail = diagnostic;
    emit errorChanged();
}

void CourseBackend::clearError()
{
    if (m_errorCode.isEmpty() && m_errorTitle.isEmpty()
        && m_errorHint.isEmpty() && m_diagnosticDetail.isEmpty()) {
        return;
    }
    m_errorCode.clear();
    m_errorTitle.clear();
    m_errorHint.clear();
    m_diagnosticDetail.clear();
    emit errorChanged();
}

void CourseBackend::clearResult()
{
    if (m_rawJson.isEmpty()) {
        return;
    }
    m_rawJson.clear();
    emit rawJsonChanged();
}
