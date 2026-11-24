#version 440

// 9-tap 高斯权重平滑模糊着色器（消除高对比竖线/文字处的对角网格伪影，提供奶油级光学散景）

layout(location = 0) in vec2 qt_TexCoord0;
layout(location = 0) out vec4 fragColor;

layout(std140, binding = 0) uniform buf {
    mat4 qt_Matrix;
    float qt_Opacity;
    vec2 texelSize;
    float offset;
} ubuf;

layout(binding = 1) uniform sampler2D source;

void main() {
    vec2 d = ubuf.texelSize * ubuf.offset;
    vec2 uv = qt_TexCoord0;

    // 9-tap 近似高斯核：中心 0.25 + 四向正交 4 * 0.125 + 四向对角 4 * 0.0625 = 1.0
    vec4 sum = texture(source, clamp(uv, 0.001, 0.999)) * 0.25;
    sum += texture(source, clamp(uv + vec2(-d.x,  0.0), 0.001, 0.999)) * 0.125;
    sum += texture(source, clamp(uv + vec2( d.x,  0.0), 0.001, 0.999)) * 0.125;
    sum += texture(source, clamp(uv + vec2( 0.0, -d.y), 0.001, 0.999)) * 0.125;
    sum += texture(source, clamp(uv + vec2( 0.0,  d.y), 0.001, 0.999)) * 0.125;
    sum += texture(source, clamp(uv + vec2(-d.x, -d.y), 0.001, 0.999)) * 0.0625;
    sum += texture(source, clamp(uv + vec2( d.x, -d.y), 0.001, 0.999)) * 0.0625;
    sum += texture(source, clamp(uv + vec2(-d.x,  d.y), 0.001, 0.999)) * 0.0625;
    sum += texture(source, clamp(uv + vec2( d.x,  d.y), 0.001, 0.999)) * 0.0625;

    fragColor = sum * ubuf.qt_Opacity;
}
