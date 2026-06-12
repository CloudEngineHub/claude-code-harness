// Package impactscore は judgment-card.v1 の impact_score (0-100) を算出する。
//
// Score 構成:
//   - floor 該当: 100（呼び出し側はカード非発行で hard-stop）
//   - floor 非該当: 0-99 を fingerprint 影響範囲で scaling
//   - files_changed * 5 + lines_changed / 10 (cap 99)
package impactscore

type Inputs struct {
	FloorCategory string // 非空なら floor 該当 → score 100
	FilesChanged  int    // worktree-fingerprint で観測される変更ファイル数
	LinesChanged  int    // 同上、行数
}

// Compute は score (0-100) を返す。負値入力は 0 に clamp、極端な値も 99 に cap。
func Compute(in Inputs) int {
	if in.FloorCategory != "" {
		return 100
	}

	files := in.FilesChanged
	if files < 0 {
		files = 0
	}
	lines := in.LinesChanged
	if lines < 0 {
		lines = 0
	}

	score := files*5 + lines/10
	if score > 99 {
		return 99
	}
	return score
}

// IsHardStop は score=100（floor 該当）かどうかを返す。
func IsHardStop(score int) bool {
	return score == 100
}
