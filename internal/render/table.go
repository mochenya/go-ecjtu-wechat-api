package render

import (
	"strconv"
	"strings"
)

// row 是一行表格数据，note 是挂在行下方的缩进续行（考试备注用）。
type row struct {
	cells   []string
	noteCol int
	note    string
}

// table 按终端显示宽度对齐：text/tabwriter 不计 CJK 宽度，所以这里自己算。
type table struct {
	headers []string
	rows    []row
	widths  []int
}

func newTable(headers ...string) *table {
	return &table{headers: headers}
}

func (t *table) add(cells ...string) {
	t.rows = append(t.rows, row{cells: cells, noteCol: -1})
}

// note 给最后一行挂一条续行，从第 col 列起缩进。
func (t *table) note(col int, text string) {
	if len(t.rows) == 0 {
		return
	}
	last := &t.rows[len(t.rows)-1]
	last.noteCol, last.note = col, text
}

func (t *table) compute() {
	t.widths = make([]int, len(t.headers))
	for i, h := range t.headers {
		t.widths[i] = DisplayWidth(h)
	}
	for _, r := range t.rows {
		for i, c := range r.cells {
			if i < len(t.widths) && DisplayWidth(c) > t.widths[i] {
				t.widths[i] = DisplayWidth(c)
			}
		}
	}
}

// columnOffset 返回第 col 列的起始显示列（0 基），供续行缩进用。
func (t *table) columnOffset(col int) int {
	if t.widths == nil {
		t.compute()
	}
	offset := 0
	for i := 0; i < col && i < len(t.widths); i++ {
		offset += t.widths[i] + columnGap
	}
	return offset
}

const columnGap = 2

// String 渲染表头、分隔线与数据行；列间固定两个空格，行尾不留空白。
func (t *table) String() string {
	t.compute()
	var b strings.Builder
	b.WriteString(t.line(t.headers))
	b.WriteString("\n")
	b.WriteString(t.rule())
	b.WriteString("\n")
	for _, r := range t.rows {
		b.WriteString(t.line(r.cells))
		b.WriteString("\n")
		if r.note != "" {
			b.WriteString(strings.Repeat(" ", t.columnOffset(r.noteCol)) + r.note)
			b.WriteString("\n")
		}
	}
	return strings.TrimRight(b.String(), "\n")
}

func (t *table) line(cells []string) string {
	parts := make([]string, 0, len(cells))
	for i, c := range cells {
		if i == len(cells)-1 {
			parts = append(parts, c) // 最后一列不补齐，避免行尾空白
			continue
		}
		parts = append(parts, pad(c, t.widths[i]))
	}
	return strings.Join(parts, strings.Repeat(" ", columnGap))
}

func (t *table) rule() string {
	parts := make([]string, len(t.widths))
	for i, w := range t.widths {
		parts[i] = strings.Repeat("-", w)
	}
	return strings.Join(parts, strings.Repeat(" ", columnGap))
}

func pad(s string, width int) string {
	if n := width - DisplayWidth(s); n > 0 {
		return s + strings.Repeat(" ", n)
	}
	return s
}

// headerLine 拼视图首行：姓名/学期与条数。页头缺失（空态页）时只写条数，不留空括号。
func headerLine(name, term, noun string, count int) string {
	countPart := "共 " + strconv.Itoa(count) + " " + noun
	detail := strings.TrimSpace(name + " " + term)
	if detail == "" {
		return countPart
	}
	return detail + "（" + countPart + "）"
}
