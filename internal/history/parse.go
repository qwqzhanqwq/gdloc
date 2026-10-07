package history

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"time"
)

// change 是一次文件改动。status 为 git 的状态字母，重命名时 oldPath/newPath 不同。
type change struct {
	status  byte
	oldMode string
	newMode string
	oldSHA  string
	newSHA  string
	oldPath string
	newPath string
}

// commit 是一个非合并提交及其改动列表。
type commit struct {
	date    time.Time
	changes []change
}

// logFormat 用 0x1e 标记记录开始、0x1f 结束头部，避免与路径内容混淆。
const logFormat = "%x1e%H%x00%aI%x00%P%x1f"

// parseLog 解析 `git log --raw -z --format=logFormat` 的输出。
func parseLog(data []byte) ([]commit, error) {
	var out []commit
	for i := 0; i < len(data); {
		switch data[i] {
		case '\x1e':
			i++
			var fields []string
			start := i
			for i < len(data) && data[i] != '\x1f' {
				if data[i] == 0 {
					fields = append(fields, string(data[start:i]))
					start = i + 1
				}
				i++
			}
			fields = append(fields, string(data[start:i]))
			if i < len(data) {
				i++ // 跳过 0x1f
			}
			if len(fields) < 2 {
				return nil, errors.New("unexpected git log header")
			}
			date, err := time.Parse(time.RFC3339, fields[1])
			if err != nil {
				return nil, fmt.Errorf("unexpected git log date %q", fields[1])
			}
			out = append(out, commit{date: date})
		case ':':
			if len(out) == 0 {
				return nil, errors.New("unexpected git raw entry")
			}
			ch, next, err := parseRawEntry(data, i)
			if err != nil {
				return nil, err
			}
			out[len(out)-1].changes = append(out[len(out)-1].changes, ch)
			i = next
		default:
			i++ // 记录之间的 NUL、换行等分隔字节
		}
	}
	return out, nil
}

// parseRawEntries 解析一段 `--raw -z` 输出（没有提交头，如 git diff --raw）。
func parseRawEntries(data []byte) ([]change, error) {
	var out []change
	for i := 0; i < len(data); {
		if data[i] != ':' {
			i++
			continue
		}
		ch, next, err := parseRawEntry(data, i)
		if err != nil {
			return nil, err
		}
		out = append(out, ch)
		i = next
	}
	return out, nil
}

// parseRawEntry 解析一条 raw 记录，返回改动与下一条的起始下标。
// 格式：":<old mode> <new mode> <old sha> <new sha> <status><NUL><path><NUL>"，
// 重命名/复制时路径为旧的在前、新的在后。
func parseRawEntry(data []byte, i int) (change, int, error) {
	end := bytes.IndexByte(data[i:], 0)
	if end < 0 {
		return change{}, 0, errors.New("unexpected git raw entry")
	}
	fields := strings.Fields(string(data[i+1 : i+end]))
	if len(fields) < 5 || fields[4] == "" {
		return change{}, 0, fmt.Errorf("unexpected git raw entry %q", string(data[i:i+end]))
	}
	ch := change{
		oldMode: fields[0], newMode: fields[1],
		oldSHA: fields[2], newSHA: fields[3], status: fields[4][0],
	}
	i += end + 1
	names := 1
	if ch.status == 'R' || ch.status == 'C' {
		names = 2
	}
	for n := 0; n < names; n++ {
		next := bytes.IndexByte(data[i:], 0)
		if next < 0 {
			return change{}, 0, errors.New("unexpected git raw entry path")
		}
		name := string(data[i : i+next])
		i += next + 1
		if n == 0 {
			ch.oldPath = name
		} else {
			ch.newPath = name
		}
	}
	if ch.newPath == "" {
		ch.newPath = ch.oldPath
	}
	return ch, i, nil
}

// treeEntry 是 git ls-tree 的一条记录。
type treeEntry struct {
	mode string
	sha  string
	path string
}

// parseTree 解析 `git ls-tree -r -z` 的输出："<mode> <type> <sha>\t<path><NUL>"。
func parseTree(data []byte) []treeEntry {
	var out []treeEntry
	for _, rec := range bytes.Split(data, []byte{0}) {
		tab := bytes.IndexByte(rec, '\t')
		if tab < 0 {
			continue
		}
		fields := strings.Fields(string(rec[:tab]))
		if len(fields) < 3 {
			continue
		}
		out = append(out, treeEntry{mode: fields[0], sha: fields[2], path: string(rec[tab+1:])})
	}
	return out
}

// splitZ 拆分以 NUL 结尾的路径列表（如 git ls-files -z）。
func splitZ(data []byte) []string {
	var out []string
	for _, rec := range bytes.Split(data, []byte{0}) {
		if len(rec) > 0 {
			out = append(out, string(rec))
		}
	}
	return out
}

// regularFile 判断 git 模式是否为普通文件（排除软链接与子模块）。
func regularFile(mode string) bool {
	if mode == "" || mode == "000000" {
		return false
	}
	return strings.HasPrefix(mode, "100")
}
