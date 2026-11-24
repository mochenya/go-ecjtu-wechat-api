#version 440

// 自研液态水滴玻璃胶囊着色器（v4 · 通透水晶胶囊）：
// - 彻底去除浑浊实白贴片与发灰暗圈，呈现 rdev/liquid-glass-react 风格清透液态水滴滑块
// - 刀锋级 1.2px 方向性镜面玻璃边 + 极轻内侧水光 + 拖拽/切换时动态折射与 RGB 色散

layout(location = 0) in vec2 qt_TexCoord0;
layout(location = 0) out vec4 fragColor;

layout(std140, binding = 0) uniform buf {
    mat4 qt_Matrix;
    float qt_Opacity;
    vec2 itemSize;
    float cornerRadius;
    float hasBackdrop;
    float stretch;
    float isDark;
    float hoverAmount;
    vec2 pointerNorm;
} ubuf;

layout(binding = 1) uniform sampler2D source;

float sdSquirclePill(vec2 p, vec2 b, float r, float power) {
    vec2 q = abs(p) - b + vec2(r);
    vec2 mq = max(q, vec2(0.0));
    float cornerDist = (mq.x > 0.0 || mq.y > 0.0)
        ? pow(pow(max(mq.x, 1e-5), power) + pow(max(mq.y, 1e-5), power), 1.0 / power)
        : 0.0;
    return cornerDist + min(max(q.x, q.y), 0.0) - r;
}

vec2 pillNormal2D(vec2 p, vec2 b, float r, float power) {
    vec2 eps = vec2(0.65, 0.0);
    float gx = sdSquirclePill(p + eps.xy, b, r, power) - sdSquirclePill(p - eps.xy, b, r, power);
    float gy = sdSquirclePill(p + eps.yx, b, r, power) - sdSquirclePill(p - eps.yx, b, r, power);
    vec2 g = vec2(gx, gy);
    float len = length(g);
    return len > 1e-4 ? g / len : vec2(0.0, -1.0);
}

vec4 sampleSoft(vec2 uv, vec2 stepUv) {
    vec4 acc = texture(source, clamp(uv, 0.001, 0.999)) * 0.40;
    acc += texture(source, clamp(uv + vec2(-stepUv.x, 0.0), 0.001, 0.999)) * 0.15;
    acc += texture(source, clamp(uv + vec2( stepUv.x, 0.0), 0.001, 0.999)) * 0.15;
    acc += texture(source, clamp(uv + vec2(0.0, -stepUv.y), 0.001, 0.999)) * 0.15;
    acc += texture(source, clamp(uv + vec2(0.0,  stepUv.y), 0.001, 0.999)) * 0.15;
    return acc;
}

void main() {
    const float PILL_N = 3.2;

    vec2 size = max(ubuf.itemSize, vec2(4.0));
    vec2 halfSize = size * 0.5;
    vec2 pillHalf = max(halfSize - vec2(1.0, 1.5), vec2(2.0));
    float r = min(ubuf.cornerRadius, min(pillHalf.x, pillHalf.y));

    vec2 px = qt_TexCoord0 * size - halfSize;
    float d = sdSquirclePill(px, pillHalf, r, PILL_N);
    float mask = 1.0 - smoothstep(-0.65, 0.65, d);

    // 极轻悬浮接触影
    if (d > 0.65) {
        float dSh = sdSquirclePill(px - vec2(0.0, 1.2), pillHalf, r, PILL_N);
        float sh = exp(-max(dSh, 0.0) * 1.45) * 0.22;
        fragColor = vec4(vec3(0.0), sh) * ubuf.qt_Opacity;
        return;
    }

    float distIn = max(-d, 0.0);
    vec2 n2 = pillNormal2D(px, pillHalf, r, PILL_N);

    // 刀锋级 1.2px 方向性白晶边 + 顶部细微弧光（零暗圈、零实白贴片）
    float localY = clamp((px.y + pillHalf.y) / max(pillHalf.y * 2.0, 1.0), 0.0, 1.0);
    float topSheen = pow(1.0 - localY, 2.8) * 0.10;
    float outerRim = smoothstep(1.45, 0.15, distIn);
    float innerGlow = smoothstep(3.2, 0.5, distIn) * smoothstep(0.0, 0.8, distIn);

    vec2 ptrBias = (ubuf.pointerNorm - vec2(0.5)) * 2.0;
    float angle = radians(135.0 + ptrBias.x * 24.0);
    vec2 gradDir = vec2(cos(angle), sin(angle));
    vec2 normPos = px / max(pillHalf, vec2(1.0));
    float gradT = clamp(dot(normPos, gradDir) * 0.5 + 0.5, 0.0, 1.0);
    float diagGlint = exp(-pow((gradT - 0.25) / 0.24, 2.0)) * 0.42
                    + exp(-pow((gradT - 0.78) / 0.26, 2.0)) * 0.28;
    float rimAlpha = clamp(outerRim * (0.42 + diagGlint + smoothstep(0.1, -0.9, n2.y) * 0.22), 0.0, 0.90);

    // 清透亮晶液滴本体（静止仅 ~19% 透亮白晶，悬停微亮，拖拽拉伸略微变薄透出折射张力）
    float fillAlpha = clamp(0.19 + ubuf.hoverAmount * 0.045 - ubuf.stretch * 0.045 + topSheen, 0.12, 0.34);
    vec3 glassTint = vec3(1.0, 1.0, 1.0);

    vec3 rgb;
    float outAlpha;

    if (ubuf.hasBackdrop > 0.5) {
        float edgeCurve = pow(1.0 - smoothstep(0.0, 10.0, distIn), 1.7);
        float shiftPx = (4.5 + ubuf.stretch * 5.0) * edgeCurve;
        float chrom = 0.16 + ubuf.stretch * 0.16;

        vec2 magUv = (qt_TexCoord0 - vec2(0.5)) * (1.0 - 0.025 - ubuf.stretch * 0.025) + vec2(0.5);
        vec2 baseOff = -n2 * (shiftPx / size);
        vec2 blurStep = vec2(1.2) / size;

        vec4 sR = sampleSoft(magUv + baseOff * (1.0 + chrom), blurStep);
        vec4 sG = sampleSoft(magUv + baseOff, blurStep);
        vec4 sB = sampleSoft(magUv + baseOff * (1.0 - chrom), blurStep);

        float bgAlpha = clamp((sR.a + sG.a + sB.a) / 3.0, 0.0, 1.0);
        vec3 bgRgb = bgAlpha > 0.01 ? clamp(vec3(sR.r, sG.g, sB.b) / bgAlpha, 0.0, 1.0) : vec3(0.0);
        // 与主壳中性烟晶基底对齐亮度，避免桌面亮色壁纸下胶囊局部过亮发白
        bgRgb = mix(bgRgb * 0.84, vec3(0.06, 0.09, 0.15), 0.14);

        // Porter-Duff Over：极轻白晶水滴 + 3px 内微光 + 1.2px 锐利白边叠在折射背景之上
        float bodyA = clamp(fillAlpha * 0.92 + innerGlow * 0.12, 0.0, 0.36);
        vec4 bgPremul = vec4(bgRgb * bgAlpha, bgAlpha);
        vec4 bodyPremul = vec4(glassTint * bodyA, bodyA);
        vec4 rimPremul = vec4(vec3(1.0) * rimAlpha, rimAlpha);

        vec4 comp = bodyPremul + bgPremul * (1.0 - bodyPremul.a);
        comp = rimPremul + comp * (1.0 - rimPremul.a);

        outAlpha = clamp(comp.a, 0.0, 0.96) * mask;
        rgb = comp.a > 1e-4 ? clamp(comp.rgb / comp.a, 0.0, 1.0) : glassTint;
    } else {
        float bodyA = clamp(fillAlpha + innerGlow * 0.14, 0.0, 0.44);
        vec4 bodyPremul = vec4(glassTint * bodyA, bodyA);
        vec4 rimPremul = vec4(vec3(1.0) * rimAlpha, rimAlpha);
        vec4 comp = rimPremul + bodyPremul * (1.0 - rimPremul.a);

        outAlpha = clamp(comp.a, 0.0, 0.94) * mask;
        rgb = comp.a > 1e-4 ? clamp(comp.rgb / comp.a, 0.0, 1.0) : glassTint;
    }

    rgb = clamp(rgb, 0.0, 1.0);
    fragColor = vec4(rgb * outAlpha, outAlpha) * ubuf.qt_Opacity;
}
