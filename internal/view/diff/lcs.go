package diff

// DiffOp classifies a row in a line-diff.
type DiffOp int

const (
	OpEqual   DiffOp = iota // both sides identical
	OpChanged               // left and right differ (paired del + ins)
	OpDelete                // only present on the left
	OpInsert                // only present on the right
)

// DiffPair is one aligned row in a side-by-side diff. For OpDelete the
// right side is empty; for OpInsert the left side is empty.
type DiffPair struct {
	Op    DiffOp
	Left  string
	Right string
}

// AlignLines produces an aligned line-level diff between left and right.
// Uses Longest Common Subsequence to find equal lines, then walks the DP
// table to emit a sequence of equal / delete / insert ops. Adjacent
// delete-then-insert is merged into a single OpChanged so the rendered
// rows stay aligned and you can see what swapped.
//
// O(n*m) time and space — fine for SKILL.md files, which are small.
func AlignLines(left, right []string) []DiffPair {
	n, m := len(left), len(right)

	// DP table: dp[i][j] = length of LCS of left[:i] and right[:j].
	dp := make([][]int, n+1)
	for i := range dp {
		dp[i] = make([]int, m+1)
	}
	for i := 1; i <= n; i++ {
		for j := 1; j <= m; j++ {
			if left[i-1] == right[j-1] {
				dp[i][j] = dp[i-1][j-1] + 1
			} else if dp[i-1][j] >= dp[i][j-1] {
				dp[i][j] = dp[i-1][j]
			} else {
				dp[i][j] = dp[i][j-1]
			}
		}
	}

	// Walk back from (n,m) building a reversed op list.
	type rawOp struct {
		kind DiffOp
		l, r string
	}
	var reversed []rawOp
	i, j := n, m
	for i > 0 && j > 0 {
		switch {
		case left[i-1] == right[j-1]:
			reversed = append(reversed, rawOp{OpEqual, left[i-1], right[j-1]})
			i--
			j--
		case dp[i-1][j] >= dp[i][j-1]:
			reversed = append(reversed, rawOp{OpDelete, left[i-1], ""})
			i--
		default:
			reversed = append(reversed, rawOp{OpInsert, "", right[j-1]})
			j--
		}
	}
	for i > 0 {
		reversed = append(reversed, rawOp{OpDelete, left[i-1], ""})
		i--
	}
	for j > 0 {
		reversed = append(reversed, rawOp{OpInsert, "", right[j-1]})
		j--
	}

	// Reverse into chronological order.
	ops := make([]rawOp, len(reversed))
	for k := range reversed {
		ops[len(reversed)-1-k] = reversed[k]
	}

	// Merge adjacent delete↔insert into a single OpChanged row regardless
	// of which side appeared first in the walkback (the tie-break in the
	// DP can flip the order), so swaps render side-by-side.
	var out []DiffPair
	for k := 0; k < len(ops); k++ {
		o := ops[k]
		if k+1 < len(ops) {
			next := ops[k+1]
			if o.kind == OpDelete && next.kind == OpInsert {
				out = append(out, DiffPair{Op: OpChanged, Left: o.l, Right: next.r})
				k++
				continue
			}
			if o.kind == OpInsert && next.kind == OpDelete {
				out = append(out, DiffPair{Op: OpChanged, Left: next.l, Right: o.r})
				k++
				continue
			}
		}
		out = append(out, DiffPair{Op: o.kind, Left: o.l, Right: o.r})
	}
	return out
}
