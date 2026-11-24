#include <QDir>
#include <QFile>
#include <QSignalSpy>
#include <QTemporaryDir>
#include <QtTest>

#include "CourseBackend.h"
#include "GlassWindowHelper.h"

namespace {

class ScopedEnv
{
public:
    ScopedEnv(const char *name, const QByteArray &value)
        : m_name(name)
        , m_hadOld(qEnvironmentVariableIsSet(name))
        , m_old(qgetenv(name))
    {
        if (value.isNull()) {
            qunsetenv(name);
        } else {
            qputenv(name, value);
        }
    }

    ~ScopedEnv()
    {
        if (m_hadOld) {
            qputenv(m_name.constData(), m_old);
        } else {
            qunsetenv(m_name.constData());
        }
    }

private:
    QByteArray m_name;
    bool m_hadOld;
    QByteArray m_old;
};

} // namespace

class TestCourseBackend : public QObject
{
    Q_OBJECT

private slots:
    void initTestCase()
    {
        m_fakeBinDir = QStringLiteral(ECJTU_TEST_DIR) + QStringLiteral("/fake-bin");
        QVERIFY2(QFile::exists(m_fakeBinDir + QStringLiteral("/ecjtu")),
                 "fake ecjtu executable must exist");
    }

    void testMissingTokenDoesNotSpawnProcess()
    {
        QTemporaryDir tmpDir;
        QVERIFY(tmpDir.isValid());
        const QString logPath = tmpDir.filePath(QStringLiteral("calls.log"));

        ScopedEnv tokenEnv("ECJTU_WEIXIN_TOKEN", QByteArray());
        ScopedEnv pathEnv("PATH", m_fakeBinDir.toUtf8() + ":" + qgetenv("PATH"));
        ScopedEnv logEnv("ECJTU_FAKE_LOG", logPath.toUtf8());

        CourseBackend backend;
        backend.openAndRefreshToday();

        QCOMPARE(backend.hasToken(), false);
        QCOMPARE(backend.state(), CourseBackend::State::MissingToken);
        QVERIFY(backend.rawJson().isEmpty());
        QVERIFY(!QFile::exists(logPath));
    }

    void testSuccessNormalAndTomorrow()
    {
        ScopedEnv tokenEnv("ECJTU_WEIXIN_TOKEN", QByteArrayLiteral("TESTFAKEOPENID1234567890"));
        ScopedEnv pathEnv("PATH", m_fakeBinDir.toUtf8() + ":" + qgetenv("PATH"));
        ScopedEnv modeEnv("ECJTU_FAKE_MODE", QByteArrayLiteral("normal"));
        const QString tomorrow = CourseBackend::shanghaiDateForDay(QStringLiteral("tomorrow"));
        ScopedEnv tomorrowEnv("ECJTU_FAKE_TOMORROW", tomorrow.toUtf8());

        CourseBackend backend;
        backend.openAndRefreshToday();

        QTRY_COMPARE_WITH_TIMEOUT(backend.state(), CourseBackend::State::Ready, 3000);
        QCOMPARE(backend.selectedDay(), QStringLiteral("today"));
        QVERIFY(backend.rawJson().contains(QStringLiteral("大学英语Ⅲ")));
        QVERIFY(backend.rawJson().contains(QStringLiteral("数据库系统原理")));

        backend.selectDay(QStringLiteral("tomorrow"));
        QTRY_COMPARE_WITH_TIMEOUT(backend.state(), CourseBackend::State::Ready, 3000);
        QCOMPARE(backend.selectedDay(), QStringLiteral("tomorrow"));
        QCOMPARE(backend.activeDate(), tomorrow);
        QVERIFY(backend.rawJson().contains(QStringLiteral("数据结构与算法")));
    }

    void testDuplicateClickSameDayIgnoredWhileLoading()
    {
        QTemporaryDir tmpDir;
        QVERIFY(tmpDir.isValid());
        const QString logPath = tmpDir.filePath(QStringLiteral("calls.log"));

        ScopedEnv tokenEnv("ECJTU_WEIXIN_TOKEN", QByteArrayLiteral("TESTFAKEOPENID1234567890"));
        ScopedEnv pathEnv("PATH", m_fakeBinDir.toUtf8() + ":" + qgetenv("PATH"));
        ScopedEnv modeEnv("ECJTU_FAKE_MODE", QByteArrayLiteral("normal"));
        ScopedEnv delayEnv("ECJTU_FAKE_DELAY_MS", QByteArrayLiteral("220"));
        ScopedEnv logEnv("ECJTU_FAKE_LOG", logPath.toUtf8());

        CourseBackend backend;
        backend.openAndRefreshToday();
        QCOMPARE(backend.state(), CourseBackend::State::Loading);

        // 同一天查询进行中重复点击，不应再次启动子进程
        backend.selectDay(QStringLiteral("today"));
        backend.openAndRefreshToday();

        QTRY_COMPARE_WITH_TIMEOUT(backend.state(), CourseBackend::State::Ready, 3000);

        QFile logFile(logPath);
        QVERIFY(logFile.open(QIODevice::ReadOnly | QIODevice::Text));
        const QList<QByteArray> lines = logFile.readAll().trimmed().split('\n');
        QCOMPARE(lines.size(), 1);
    }

    void testSwitchDayDropsStaleResult()
    {
        const QString today = CourseBackend::shanghaiDateForDay(QStringLiteral("today"));
        const QString tomorrow = CourseBackend::shanghaiDateForDay(QStringLiteral("tomorrow"));

        ScopedEnv tokenEnv("ECJTU_WEIXIN_TOKEN", QByteArrayLiteral("TESTFAKEOPENID1234567890"));
        ScopedEnv pathEnv("PATH", m_fakeBinDir.toUtf8() + ":" + qgetenv("PATH"));
        ScopedEnv modeEnv("ECJTU_FAKE_MODE", QByteArrayLiteral("race_today_slow"));
        ScopedEnv todayEnv("ECJTU_FAKE_TODAY", today.toUtf8());

        CourseBackend backend;
        backend.openAndRefreshToday();
        QCOMPARE(backend.state(), CourseBackend::State::Loading);

        // 立即切到明天，旧的今天请求不能覆盖明天的结果
        backend.selectDay(QStringLiteral("tomorrow"));
        QTRY_COMPARE_WITH_TIMEOUT(backend.state(), CourseBackend::State::Ready, 3000);

        QTest::qWait(450);
        QCOMPARE(backend.selectedDay(), QStringLiteral("tomorrow"));
        QCOMPARE(backend.activeDate(), tomorrow);
        QVERIFY(backend.rawJson().contains(QStringLiteral("新请求操作系统")));
        QVERIFY(!backend.rawJson().contains(QStringLiteral("旧请求大学英语Ⅲ")));
    }

    void testExitCodeClassificationAndSanitization()
    {
        const QByteArray secret = QByteArrayLiteral("ABCD9999SECRETTOKEN123456");
        ScopedEnv tokenEnv("ECJTU_WEIXIN_TOKEN", secret);
        ScopedEnv pathEnv("PATH", m_fakeBinDir.toUtf8() + ":" + qgetenv("PATH"));

        {
            ScopedEnv modeEnv("ECJTU_FAKE_MODE", QByteArrayLiteral("exit3"));
            CourseBackend backend;
            backend.openAndRefreshToday();
            QTRY_COMPARE_WITH_TIMEOUT(backend.state(), CourseBackend::State::Error, 3000);
            QCOMPARE(backend.errorCode(), QStringLiteral("not_bound"));
            QVERIFY(!backend.diagnosticDetail().contains(QString::fromLatin1(secret)));
            QVERIFY(backend.diagnosticDetail().contains(QStringLiteral("ABCD***")));
        }

        {
            ScopedEnv modeEnv("ECJTU_FAKE_MODE", QByteArrayLiteral("exit4"));
            CourseBackend backend;
            backend.openAndRefreshToday();
            QTRY_COMPARE_WITH_TIMEOUT(backend.state(), CourseBackend::State::Error, 3000);
            QCOMPARE(backend.errorCode(), QStringLiteral("upstream"));
        }

        {
            ScopedEnv modeEnv("ECJTU_FAKE_MODE", QByteArrayLiteral("exit5"));
            CourseBackend backend;
            backend.openAndRefreshToday();
            QTRY_COMPARE_WITH_TIMEOUT(backend.state(), CourseBackend::State::Error, 3000);
            QCOMPARE(backend.errorCode(), QStringLiteral("parse"));
        }
    }

    void testExecutableNotFound()
    {
        QTemporaryDir emptyPath;
        QVERIFY(emptyPath.isValid());

        ScopedEnv tokenEnv("ECJTU_WEIXIN_TOKEN", QByteArrayLiteral("TESTFAKEOPENID1234567890"));
        ScopedEnv pathEnv("PATH", emptyPath.path().toUtf8());

        CourseBackend backend;
        backend.openAndRefreshToday();

        QTRY_COMPARE_WITH_TIMEOUT(backend.state(), CourseBackend::State::Error, 3000);
        QCOMPARE(backend.errorCode(), QStringLiteral("not_found"));
    }

    void testSquircleBlurRegionRespectsInsetAndCorners()
    {
        const QRegion region = GlassWindowHelper::buildSquircleRegion(392, 480, 28.0, 14.0);
        QVERIFY(!region.isEmpty());
        // 中心点必须在模糊区域内
        QVERIFY(region.contains(QPoint(196, 240)));
        // 外围透明柔影边距（0..14px）与外角尖端绝不能被包含
        QVERIFY(!region.contains(QPoint(4, 4)));
        QVERIFY(!region.contains(QPoint(15, 15)));
        QVERIFY(!region.contains(QPoint(388, 476)));
        // 紧贴直边内侧必须落在区域内
        QVERIFY(region.contains(QPoint(196, 20)));
        QVERIFY(region.contains(QPoint(20, 240)));
    }

private:
    QString m_fakeBinDir;
};

QTEST_MAIN(TestCourseBackend)
#include "test_course_backend.moc"
