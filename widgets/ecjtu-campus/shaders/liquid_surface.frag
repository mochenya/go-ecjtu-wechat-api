#version 440

// 自研 Clear Liquid Glass 主外壳片元着色器（v4 · 对标 rdev/liquid-glass-react 光学语言）：
// - 彻底去除厚重 26px 枕头内阴影与发灰泥边，主体区域极致平顺清澈，让背景色彩与光影饱满透出
// - Apple G2 连续曲率超椭圆（Squircle, n=3.8）轮廓 + 柔和浮空外投影（0 12px 38px rgba(0,0,0,0.28)）
// - 刀锋级 1.5px 方向性镜面玻璃切边（135° + 光标随动双峰斜向高光）+ 3px 极轻内侧玻璃微光
// - 桌面壁纸模式：rdev/liquid-glass-react 风格边缘弯曲折射 + 仅限边缘的 RGB 三通道色散 + 1.55x 高饱和度通透柔焦
// - 面板弹窗模式：极致轻薄中性烟晶光学膜（仅 ~19% 不透明度），完全释放 KWin 活色模糊的真实色彩与景深

layout(location = 0) in vec2 qt_TexCoord0;
layout(location = 0) out vec4 fragColor;

layout(std140, binding = 0) uniform buf {
    mat4 qt_Matrix;
    float qt_Opacity;
    vec2 itemSize;
    float cornerRadius;
    float insetPad;
    float hasBackdrop;
    float bevelWidth;
    float glassThickness;
    float ior;
    float dispersion;
    float specularStrength;
    float isDark;
    vec2 pointerNorm;
    vec4 tintColor;
} ubuf;

layout(binding = 1) uniform sampler2D sharpSource;
layout(binding = 2) uniform sampler2D softSource;

// Apple 连续曲率超椭圆圆角矩形 SDF（与 C++ GlassWindowHelper::buildSquircleRegion 完全同参 n = 3.8）
float sdSquircleBox(vec2 p, vec2 b, float r, float power) {
    vec2 q = abs(p) - b + vec2(r);
    vec2 mq = max(q, vec2(0.0));
    float cornerDist = (mq.x > 0.0 || mq.y > 0.0)
        ? pow(pow(max(mq.x, 1e-5), power) + pow(max(mq.y, 1e-5), power), 1.0 / power)
        : 0.0;
    return cornerDist + min(max(q.x, q.y), 0.0) - r;
}

vec2 squircleNormal2D(vec2 p, vec2 b, float r, float power) {
    vec2 eps = vec2(0.75, 0.0);
    float gx = sdSquircleBox(p + eps.xy, b, r, power) - sdSquircleBox(p - eps.xy, b, r, power);
    float gy = sdSquircleBox(p + eps.yx, b, r, power) - sdSquircleBox(p - eps.yx, b, r, power);
    vec2 g = vec2(gx, gy);
    float len = length(g);
    return len > 1e-4 ? g / len : vec2(0.0, -1.0);
}

vec3 adjustVibrancy(vec3 rgb, float saturation) {
    float luma = dot(rgb, vec3(0.2126, 0.7152, 0.0722));
    return clamp(mix(vec3(luma), rgb, saturation), 0.0, 1.0);
}

vec2 foldUv(vec2 uv) {
    vec2 m = mod(uv, 2.0);
    vec2 folded = mix(m, 2.0 - m, step(1.0, m));
    return clamp(folded, 0.001, 0.999);
}

// 柔和悬浮投影（贴合 rdev/liquid-glass-react: 0 12px 40px rgba(0,0,0,0.28)）
float computeShadowAlpha(vec2 px, vec2 glassHalf, float r, float power, vec2 uv, vec2 size) {
    float dShadow = sdSquircleBox(px - vec2(0.0, 4.5), glassHalf, r, power);
    float tightShadow = exp(-max(dShadow, 0.0) * 0.42) * 0.19;
    float softShadow = exp(-max(dShadow, 0.0) * 0.16) * 0.16;
    float edgeFade = smoothstep(0.0, 3.0, min(min(uv.x * size.x, (1.0 - uv.x) * size.x),
                                             min(uv.y * size.y, (1.0 - uv.y) * size.y)));
    return clamp((tightShadow + softShadow) * edgeFade, 0.0, 0.40);
}

void main() {
    const float SQUIRCLE_N = 3.8;

    vec2 size = max(ubuf.itemSize, vec2(4.0));
    vec2 halfSize = size * 0.5;
    float pad = max(ubuf.insetPad, 0.0);
    vec2 glassHalf = max(halfSize - vec2(pad), vec2(2.0));
    float r = min(ubuf.cornerRadius, min(glassHalf.x, glassHalf.y));

    vec2 px = qt_TexCoord0 * size - halfSize;
    float d = sdSquircleBox(px, glassHalf, r, SQUIRCLE_N);

    // 1. 玻璃外侧：仅绘制通透浮空投影（无任何方块裁边）
    float shapeAlpha = 1.0 - smoothstep(-0.75, 0.75, d);
    if (d > 0.75) {
        if (pad <= 0.5) {
            fragColor = vec4(0.0);
            return;
        }
        float shAlpha = computeShadowAlpha(px, glassHalf, r, SQUIRCLE_N, qt_TexCoord0, size);
        vec3 shColor = vec3(0.01, 0.02, 0.05);
        fragColor = vec4(shColor * shAlpha, shAlpha) * ubuf.qt_Opacity;
        return;
    }

    // 2. 玻璃内侧：零厚重枕头阴影，只保留刀锋级 1.5px 镜面切边 + 3px 极微内缘反光
    float distIn = max(-d, 0.0);
    vec2 n2 = squircleNormal2D(px, glassHalf, r, SQUIRCLE_N);

    // 复刻 rdev/liquid-glass-react 双层 135° 光标随动镜面描边（外切边 0..1.5px + 极微内光 1..4px）
    vec2 mouseOffset = (ubuf.pointerNorm - vec2(0.5)) * 2.0; // [-1, 1]
    float angle = radians(135.0 + mouseOffset.x * 26.0);
    vec2 gradDir = vec2(cos(angle), sin(angle));
    vec2 normPos = px / max(glassHalf, vec2(1.0));
    float gradT = clamp(dot(normPos, gradDir) * 0.5 + 0.5 + mouseOffset.y * 0.08, 0.0, 1.0);

    // 双峰对角迎光（模拟 screen + overlay 两层斜向渐变描边）+ 上/下边缘自然高光承托
    float diagPeak1 = exp(-pow((gradT - 0.24) / 0.22, 2.0));
    float diagPeak2 = exp(-pow((gradT - 0.78) / 0.24, 2.0));
    float topCatch = smoothstep(0.15, -0.95, n2.y) * 0.28;
    float bottomCatch = smoothstep(-0.15, 0.95, n2.y) * 0.16;

    // 刀锋外描边（0..1.5px）：四周保持连续清晰 0.5px 白晶发丝，对角弧面与顶边明亮闪耀
    float hairlineMask = smoothstep(1.35, 0.15, distIn);
    float outerBorderMask = smoothstep(1.85, 0.25, distIn);
    float innerGlowMask = smoothstep(4.2, 0.6, distIn) * smoothstep(0.0, 1.0, distIn);

    float rimStrength = clamp(0.46 + diagPeak1 * 0.44 + diagPeak2 * 0.34 + topCatch + bottomCatch, 0.42, 1.0);
    float rimAlpha = clamp((hairlineMask * 0.48 + outerBorderMask * rimStrength * 0.48) * ubuf.specularStrength, 0.0, 0.92);
    float innerGlowAlpha = clamp(innerGlowMask * (0.11 + diagPeak1 * 0.08 + topCatch * 0.18) * ubuf.specularStrength, 0.0, 0.22);

    // 极克制的顶部天光与斜向丝绸微反光（< 4%，不制造白雾，仅赋予玻璃体表存在感）
    float localY = clamp((px.y + glassHalf.y) / max(glassHalf.y * 2.0, 1.0), 0.0, 1.0);
    float surfaceSheen = pow(1.0 - localY, 4.5) * 0.032 + diagPeak1 * 0.014;

    // 角部极细棱镜微光（仅附着在 1.5px 切边拐角，无浑浊灰环）
    float cornerCurv = clamp(abs(n2.x * n2.y) * 2.0, 0.0, 1.0);
    float prismPhase = dot(n2, vec2(0.707, -0.707));
    vec3 prismTint = mix(vec3(0.84, 0.96, 1.0), vec3(1.0, 0.91, 0.98), clamp(prismPhase * 0.5 + 0.5, 0.0, 1.0));
    vec3 rimColor = mix(vec3(1.0), prismTint, cornerCurv * 0.28);

    vec3 outRgb;
    float outAlpha;

    if (ubuf.hasBackdrop > 0.5) {
        // === 桌面壁纸模式：rdev/liquid-glass-react 边缘折射 + 边缘 RGB 色散 + 深奶油柔焦 + 1.55x 高饱和 ===
        float edgeW = max(ubuf.bevelWidth, 12.0);
        float tEdge = clamp(distIn / edgeW, 0.0, 1.0);
        // 仅在边缘 16..18px 产生平滑液态弯曲折射，阅读区 100% 零畸变
        float edgeCurve = pow(1.0 - smoothstep(0.0, 1.0, tEdge), 1.85);
        float dispMag = ubuf.glassThickness * 0.52 * edgeCurve;

        // 模拟 rdev/liquid-glass-react shader-utils.ts 的边缘内折位移 + 轻度 RGB 通道色散
        float chrom = clamp(ubuf.dispersion, 0.0, 0.12) * 3.8;
        vec2 baseShift = -n2 * (dispMag / size);
        vec2 uvR = foldUv(qt_TexCoord0 + baseShift * (1.0 + chrom));
        vec2 uvG = foldUv(qt_TexCoord0 + baseShift);
        vec2 uvB = foldUv(qt_TexCoord0 + baseShift * (1.0 - chrom));

        // 中心采用 3-pass 丝滑奶油柔焦背景（如参考图一般顺滑）；极边缘 10px 混入少许锐利折射提升晶体边缘张力
        vec3 softBg;
        softBg.r = texture(softSource, uvR).r;
        softBg.g = texture(softSource, uvG).g;
        softBg.b = texture(softSource, uvB).b;

        vec3 sharpBg;
        sharpBg.r = texture(sharpSource, uvR).r;
        sharpBg.g = texture(sharpSource, uvG).g;
        sharpBg.b = texture(sharpSource, uvB).b;

        float lipSharp = pow(1.0 - smoothstep(0.0, 10.0, distIn), 2.2) * 0.28;
        vec3 bg = mix(softBg, sharpBg, lipSharp);

        // 对标 rdev/liquid-glass-react: saturation 155%，并微压高光防止纯白过曝吞掉白字
        bg = adjustVibrancy(bg, 1.55);
        bg = pow(max(bg, vec3(0.0)), vec3(1.05)) * 0.92;

        // 极致轻薄的均匀烟晶玻璃层（零厚框、零发灰奶膜）
        vec3 body = mix(bg, ubuf.tintColor.rgb, ubuf.tintColor.a);
        body += vec3(surfaceSheen);
        body = mix(body, vec3(1.0), innerGlowAlpha);
        outRgb = mix(body, rimColor, rimAlpha);
        outAlpha = shapeAlpha;
    } else {
        // === 面板弹窗模式（透明 PlasmaCore.Dialog + KWin 活色模糊）：
        // 彻底放行 KWin 奶油模糊与 1.62x 饱和色彩，本层只提供极薄均匀晶体滤膜 + 刀锋级 1.5px 高光玻璃边！
        vec3 baseRgb = ubuf.tintColor.rgb + vec3(surfaceSheen * 1.4);
        float baseAlpha = clamp(ubuf.tintColor.a + surfaceSheen * 0.5, 0.0, 0.34);

        // 叠加极薄 3px 内缘高光与 1.5px 刀锋白晶切边（标准 Porter-Duff Over 预乘合成，绝不发灰）
        vec4 basePremul = vec4(baseRgb * baseAlpha, baseAlpha);
        vec4 glowPremul = vec4(vec3(1.0) * innerGlowAlpha, innerGlowAlpha);
        vec4 rimPremul = vec4(rimColor * rimAlpha, rimAlpha);

        vec4 comp = glowPremul + basePremul * (1.0 - glowPremul.a);
        comp = rimPremul + comp * (1.0 - rimPremul.a);

        outAlpha = clamp(comp.a, 0.0, 0.96) * shapeAlpha;
        outRgb = comp.a > 1e-4 ? clamp(comp.rgb / comp.a, 0.0, 1.0) : baseRgb;
    }

    // 与阴影外缘 1px 抗锯齿平滑融合
    if (pad > 0.5 && d > -0.75) {
        float shAlpha = computeShadowAlpha(px, glassHalf, r, SQUIRCLE_N, qt_TexCoord0, size);
        vec3 shColor = vec3(0.01, 0.02, 0.05);
        vec4 glassPremul = vec4(clamp(outRgb, 0.0, 1.0) * outAlpha, outAlpha);
        vec4 shPremul = vec4(shColor * shAlpha, shAlpha);
        fragColor = (glassPremul + shPremul * (1.0 - glassPremul.a)) * ubuf.qt_Opacity;
        return;
    }

    outRgb = clamp(outRgb, 0.0, 1.0);
    fragColor = vec4(outRgb * outAlpha, outAlpha) * ubuf.qt_Opacity;
}
