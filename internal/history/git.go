// Package history 通过只读的 git 命令统计提交历史里的代码增量。
// 规则见 AGENTS.md 4.7：只跑只读子命令，按作者日期分桶，逐行分类复用 counter。
package history

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// zeroSHA 是 git 在 raw 输出里表示"该侧不存在"的全零对象名。
const zeroSHA = "0000000000000000000000000000000000000000"

// gitArgs 是所有 git 调用的公共前缀：禁止可选锁，并让路径不做 C 风格转义。
func gitArgs(args ...string) []string {
	return append([]string{"--no-optional-locks", "-c", "core.quotepath=false"}, args...)
}

// gitEnv 关掉分页与交互提示，保证 git 在管道中安静退出。
func gitEnv() []string {
	return append(os.Environ(), "GIT_PAGER=cat", "GIT_TERMINAL_PROMPT=0", "GIT_OPTIONAL_LOCKS=0")
}

// gitRunner 在固定目录里运行只读 git 命令。
type gitRunner struct {
	dir string
}

func (g *gitRunner) run(args ...string) ([]byte, error) {
	cmd := exec.Command("git", gitArgs(args...)...)
	cmd.Dir = g.dir
	cmd.Env = gitEnv()
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if msg := strings.TrimSpace(stderr.String()); msg != "" {
			return nil, errors.New(msg)
		}
		return nil, fmt.Errorf("cannot run git: %v", err)
	}
	return stdout.Bytes(), nil
}

// repo 是仓库定位信息。
type repo struct {
	root   string // 仓库根，OS 路径分隔符
	prefix string // 扫描根相对仓库根的路径，以 / 结尾；扫描根即仓库根时为空串
}

// detectRepo 定位 root 所属的 git 工作区。root 不在仓库内或找不到 git 时返回错误。
func detectRepo(root string) (repo, error) {
	g := &gitRunner{dir: root}
	out, err := g.run("rev-parse", "--show-toplevel", "--show-prefix")
	if err != nil {
		return repo{}, fmt.Errorf("not a git repository: %v", err)
	}
	lines := strings.Split(strings.ReplaceAll(string(out), "\r\n", "\n"), "\n")
	top := strings.TrimSpace(lines[0])
	if top == "" {
		return repo{}, errors.New("not a git repository: empty toplevel")
	}
	r := repo{root: filepath.Clean(filepath.FromSlash(top))}
	if len(lines) > 1 {
		r.prefix = filepath.ToSlash(strings.TrimSpace(lines[1]))
	}
	return r, nil
}

// hasHead 判断仓库是否已有提交（空仓库时 HEAD 尚未诞生）。
func (g *gitRunner) hasHead() bool {
	_, err := g.run("rev-parse", "--verify", "--quiet", "HEAD")
	return err == nil
}

// catFile 是一个常驻的 git cat-file --batch 进程，用来按对象名读取 blob 内容。
// 逐行分类的结果按 blob 缓存，因此同一个版本只会被解析一次。
type catFile struct {
	cmd *exec.Cmd
	in  io.WriteCloser
	out *bufio.Reader
}

func startCatFile(dir string) (*catFile, error) {
	cmd := exec.Command("git", gitArgs("cat-file", "--batch")...)
	cmd.Dir = dir
	cmd.Env = gitEnv()
	in, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("cannot run git: %v", err)
	}
	out, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("cannot run git: %v", err)
	}
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("cannot run git: %v", err)
	}
	return &catFile{cmd: cmd, in: in, out: bufio.NewReader(out)}, nil
}

// get 读取一个对象；对象不存在时 ok 为 false。
func (c *catFile) get(rev string) ([]byte, bool, error) {
	if _, err := io.WriteString(c.in, rev+"\n"); err != nil {
		return nil, false, err
	}
	header, err := c.out.ReadString('\n')
	if err != nil {
		return nil, false, err
	}
	header = strings.TrimRight(header, "\r\n")
	if strings.HasSuffix(header, " missing") || strings.HasSuffix(header, " ambiguous") {
		return nil, false, nil
	}
	fields := strings.Fields(header)
	if len(fields) != 3 {
		return nil, false, fmt.Errorf("unexpected git cat-file output %q", header)
	}
	size, err := strconv.Atoi(fields[2])
	if err != nil || size < 0 {
		return nil, false, fmt.Errorf("unexpected git cat-file size %q", fields[2])
	}
	buf := make([]byte, size+1) // 对象内容后跟一个换行
	if _, err := io.ReadFull(c.out, buf); err != nil {
		return nil, false, err
	}
	return buf[:size], true, nil
}

// close 关掉 stdin 让子进程自行退出，避免留下常驻进程。
func (c *catFile) close() {
	_ = c.in.Close()
	_, _ = io.Copy(io.Discard, c.out)
	_ = c.cmd.Wait()
}
