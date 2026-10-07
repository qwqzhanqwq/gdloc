package history

// maxTraceD 限制回溯轨迹的规模：只有近乎整体重写的文件才会超过，
// 那种情况下退化成"整份删除 + 整份新增"更省内存，结果也仍然合理。
const maxTraceD = 2000

// diffLines 求两个版本的行差异，返回新增行在 b 中的下标与删除行在 a 中的下标，均按升序。
// 这里用的是 Myers 算法，和 `git diff -U0` 给出的行号等价：
// 新增行按新版本的下标取分类，删除行按旧版本的下标取分类。
func diffLines(a, b []string) (added, deleted []int) {
	// 公共前后缀不参与 diff：既是常见情况，也让结果更接近直觉。
	p := 0
	for p < len(a) && p < len(b) && a[p] == b[p] {
		p++
	}
	s := 0
	for s < len(a)-p && s < len(b)-p && a[len(a)-1-s] == b[len(b)-1-s] {
		s++
	}
	midA, midB := a[p:len(a)-s], b[p:len(b)-s]

	switch {
	case len(midA) == 0 && len(midB) == 0:
		return nil, nil
	case len(midA) == 0:
		return seq(p, len(midB)), nil
	case len(midB) == 0:
		return nil, seq(p, len(midA))
	}

	addedMid, deletedMid, ok := myers(midA, midB)
	if !ok {
		return seq(p, len(midB)), seq(p, len(midA))
	}
	return shift(addedMid, p), shift(deletedMid, p)
}

// myers 对（已去掉公共前后缀的）两个序列求最短编辑脚本，返回相对下标。
func myers(a, b []string) (added, deleted []int, ok bool) {
	n, m := len(a), len(b)
	offset := n + m
	v := make([]int32, 2*(n+m)+3)
	trace := make([][]int32, 0, 32)

	for d := 0; d <= n+m; d++ {
		if d > maxTraceD {
			return nil, nil, false
		}
		// 记录本轮开始前的 V：回溯时用它复现同一套 k 选择。
		snap := make([]int32, 2*d+1)
		copy(snap, v[offset-d:offset+d+1])
		trace = append(trace, snap)

		for k := -d; k <= d; k += 2 {
			var x int
			if k == -d || (k != d && v[offset+k-1] < v[offset+k+1]) {
				x = int(v[offset+k+1])
			} else {
				x = int(v[offset+k-1]) + 1
			}
			y := x - k
			for x < n && y < m && a[x] == b[y] {
				x++
				y++
			}
			v[offset+k] = int32(x)
			if x >= n && y >= m {
				add, del := backtrack(trace, d, n, m)
				return add, del, true
			}
		}
	}
	return nil, nil, false
}

// backtrack 借助每轮 d 开始时的 V 快照还原编辑路径。
func backtrack(trace [][]int32, d, n, m int) (added, deleted []int) {
	x, y := n, m
	for step := d; step >= 1; step-- {
		snap := trace[step]
		at := func(k int) int { return int(snap[k+step]) }
		k := x - y
		prevK := k - 1
		if k == -step || (k != step && at(k-1) < at(k+1)) {
			prevK = k + 1
		}
		prevX := at(prevK)
		prevY := prevX - prevK
		for x > prevX && y > prevY { // 沿对角线回退的相同行
			x--
			y--
		}
		if x == prevX {
			added = append(added, prevY) // b[prevY] 是新增行
		} else {
			deleted = append(deleted, prevX) // a[prevX] 是删除行
		}
		x, y = prevX, prevY
	}
	reverse(added)
	reverse(deleted)
	return added, deleted
}

func seq(from, n int) []int {
	out := make([]int, 0, n)
	for i := 0; i < n; i++ {
		out = append(out, from+i)
	}
	return out
}

func shift(idx []int, delta int) []int {
	if delta == 0 || idx == nil {
		return idx
	}
	out := make([]int, len(idx))
	for i, v := range idx {
		out[i] = v + delta
	}
	return out
}

func reverse(idx []int) {
	for i, j := 0, len(idx)-1; i < j; i, j = i+1, j-1 {
		idx[i], idx[j] = idx[j], idx[i]
	}
}
