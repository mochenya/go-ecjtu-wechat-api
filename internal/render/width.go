package render

import (
	"strings"
	"unicode"
)

// 双格区间：中日韩文字、全角形式，以及本仓实际会出现的 East Asian Ambiguous 区段
// （课名含 Ⅰ Ⅱ Ⅲ，地点含中文与全角括号）。区间外的字符按 1 格处理：
// 只会让该列比终端实际显示略窄，不会让对齐失控。
var wideRanges = []struct{ lo, hi rune }{
	{0x1100, 0x115F}, // Hangul Jamo 初声
	{0x2000, 0x2FFF}, // 通用标点、制表符、方块、罗马数字 Ⅰ-ⅿ、带圈字母数字（Ambiguous 主区段）
	{0x3000, 0x303E}, // CJK 符号与标点（含全角空格、〈〉（））
	{0x3041, 0x33FF}, // 假名、注音、兼容谚文、带圈 CJK
	{0x3400, 0x4DBF}, // CJK 扩展 A
	{0x4E00, 0x9FFF}, // CJK 统一表意文字
	{0xAC00, 0xD7A3}, // 谚文音节
	{0xF900, 0xFAFF}, // CJK 兼容表意文字
	{0xFE30, 0xFE6F}, // CJK 兼容形式与小写法变
	{0xFF00, 0xFF60}, // 全角形式
	{0xFFE0, 0xFFE6}, // 全角符号
}

// ambiguousWide 是 Latin-1 里被普遍按双格渲染的几个符号（° ± × ÷ ¤ § ¨）。
const ambiguousWide = "¤§¨°±×÷"

// runeWidth 返回单个 rune 的显示宽度：组合记号 0，控制字符 0，宽字符 2，其余 1。
func runeWidth(r rune) int {
	switch {
	case r < 0x20 || r == 0x7f:
		return 0
	case unicode.In(r, unicode.Mn, unicode.Me, unicode.Cf):
		return 0
	case unicode.Is(unicode.Han, r) || unicode.Is(unicode.Hiragana, r) ||
		unicode.Is(unicode.Katakana, r) || unicode.Is(unicode.Hangul, r):
		return 2
	case strings.ContainsRune(ambiguousWide, r):
		return 2
	}
	for _, rg := range wideRanges {
		if r >= rg.lo && r <= rg.hi {
			return 2
		}
	}
	return 1
}

// DisplayWidth 返回 s 在终端里的显示宽度；CJK 列宽必须参与对齐计算。
func DisplayWidth(s string) int {
	total := 0
	for _, r := range s {
		total += runeWidth(r)
	}
	return total
}
