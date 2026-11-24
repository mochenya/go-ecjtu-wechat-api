.pragma library

var WEEKDAY_NAMES = ["", "星期一", "星期二", "星期三", "星期四", "星期五", "星期六", "星期日"];

// 安全解析 CLI 输出的课表 JSON，校验顶层结构，避免将残缺对象渲染到界面。
function parseScheduleJson(rawText) {
    if (!rawText || typeof rawText !== "string") {
        return { ok: false, error: "empty json payload" };
    }
    var parsed;
    try {
        parsed = JSON.parse(rawText);
    } catch (err) {
        return { ok: false, error: String(err) };
    }
    if (!parsed || typeof parsed !== "object" || Array.isArray(parsed)) {
        return { ok: false, error: "root json must be an object" };
    }
    if (!Array.isArray(parsed.courses)) {
        return { ok: false, error: "missing courses array" };
    }

    var courses = [];
    for (var i = 0; i < parsed.courses.length; ++i) {
        var c = parsed.courses[i];
        if (!c || typeof c !== "object") {
            continue;
        }
        var periods = Array.isArray(c.periods) ? c.periods : [];
        var weeks = Array.isArray(c.weeks) ? c.weeks : [];
        courses.push({
            name: typeof c.name === "string" ? c.name : "",
            status: typeof c.status === "string" ? c.status : "",
            timeText: typeof c.timeText === "string" ? c.timeText : "",
            location: typeof c.location === "string" ? c.location : "",
            teacher: typeof c.teacher === "string" ? c.teacher : "",
            periods: periods,
            weeks: weeks,
            periodLabel: formatPeriodsShort(periods),
            periodFull: formatPeriodsFull(periods),
            weeksLabel: formatWeeks(weeks),
            sectionSlot: periodTimeBucket(periods)
        });
    }

    return {
        ok: true,
        schedule: {
            date: typeof parsed.date === "string" ? parsed.date : "",
            weekday: typeof parsed.weekday === "number" ? parsed.weekday : 0,
            week: typeof parsed.week === "number" ? parsed.week : 0,
            courses: courses
        }
    };
}

// 2026-09-14 -> 9月14日（自然中文读法，去掉前导零）
function formatDisplayDate(isoDate) {
    if (!isoDate || typeof isoDate !== "string") {
        return "";
    }
    var m = /^(\d{4})-(\d{2})-(\d{2})$/.exec(isoDate.trim());
    if (!m) {
        return isoDate;
    }
    var month = parseInt(m[2], 10);
    var day = parseInt(m[3], 10);
    return month + "月" + day + "日";
}

function formatWeekday(weekdayNum) {
    if (typeof weekdayNum !== "number" || weekdayNum < 1 || weekdayNum > 7) {
        return "";
    }
    return WEEKDAY_NAMES[weekdayNum];
}

function compressRanges(nums) {
    if (!nums || nums.length === 0) {
        return [];
    }
    var sorted = [];
    for (var i = 0; i < nums.length; ++i) {
        var v = Number(nums[i]);
        if (!isNaN(v)) {
            sorted.push(v);
        }
    }
    if (sorted.length === 0) {
        return [];
    }
    sorted.sort(function (a, b) { return a - b; });

    var segments = [];
    var start = sorted[0];
    var prev = sorted[0];
    for (var j = 1; j < sorted.length; ++j) {
        if (sorted[j] === prev) {
            continue;
        }
        if (sorted[j] === prev + 1) {
            prev = sorted[j];
        } else {
            segments.push(start === prev ? String(start) : (start + "–" + prev));
            start = sorted[j];
            prev = sorted[j];
        }
    }
    segments.push(start === prev ? String(start) : (start + "–" + prev));
    return segments;
}

// 左侧固定宽度节次轨道展示（超长非连续序列优雅截断，详情展开可见完整串）
function formatPeriodsShort(periods) {
    var segs = compressRanges(periods);
    if (segs.length === 0) {
        return "—";
    }
    if (segs.length === 1) {
        return segs[0];
    }
    var joined = segs.join(",");
    if (joined.length <= 6) {
        return joined;
    }
    return segs[0] + "…";
}

function formatPeriodsFull(periods) {
    var segs = compressRanges(periods);
    if (segs.length === 0) {
        return "";
    }
    return "第 " + segs.join(", ") + " 节";
}

function formatWeeks(weeks) {
    if (!weeks || weeks.length === 0) {
        return "";
    }
    var parts = [];
    for (var i = 0; i < weeks.length; ++i) {
        var span = weeks[i];
        if (!span || typeof span.start !== "number" || typeof span.end !== "number") {
            continue;
        }
        parts.push(span.start === span.end ? String(span.start) : (span.start + "–" + span.end));
    }
    if (parts.length === 0) {
        return "";
    }
    return "第 " + parts.join(", ") + " 周";
}

// 用于跨时段（上午 1-4 / 下午 5-8 / 晚间 9+）自然分组，不虚构钟点。
function periodTimeBucket(periods) {
    if (!periods || periods.length === 0) {
        return 0;
    }
    var first = Number(periods[0]);
    if (first <= 4) {
        return 1;
    }
    if (first <= 8) {
        return 2;
    }
    return 3;
}

function sectionSlotLabel(slot) {
    if (slot === 1) return "上午";
    if (slot === 2) return "下午";
    if (slot === 3) return "晚间";
    return "";
}
