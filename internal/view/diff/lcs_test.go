package diff

import (
	"reflect"
	"testing"
)

func TestAlignLines_Identical(t *testing.T) {
	left := []string{"a", "b", "c"}
	right := []string{"a", "b", "c"}
	got := AlignLines(left, right)
	want := []DiffPair{
		{Op: OpEqual, Left: "a", Right: "a"},
		{Op: OpEqual, Left: "b", Right: "b"},
		{Op: OpEqual, Left: "c", Right: "c"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v\nwant %+v", got, want)
	}
}

func TestAlignLines_Insert(t *testing.T) {
	left := []string{"a", "c"}
	right := []string{"a", "b", "c"}
	got := AlignLines(left, right)
	want := []DiffPair{
		{Op: OpEqual, Left: "a", Right: "a"},
		{Op: OpInsert, Left: "", Right: "b"},
		{Op: OpEqual, Left: "c", Right: "c"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v\nwant %+v", got, want)
	}
}

func TestAlignLines_Delete(t *testing.T) {
	left := []string{"a", "b", "c"}
	right := []string{"a", "c"}
	got := AlignLines(left, right)
	want := []DiffPair{
		{Op: OpEqual, Left: "a", Right: "a"},
		{Op: OpDelete, Left: "b", Right: ""},
		{Op: OpEqual, Left: "c", Right: "c"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v\nwant %+v", got, want)
	}
}

// A delete immediately followed by an insert collapses into a single
// "changed" pair so swaps stay aligned.
func TestAlignLines_ChangedMergesAdjacentDelIns(t *testing.T) {
	left := []string{"hello", "world"}
	right := []string{"hello", "WORLD"}
	got := AlignLines(left, right)
	want := []DiffPair{
		{Op: OpEqual, Left: "hello", Right: "hello"},
		{Op: OpChanged, Left: "world", Right: "WORLD"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v\nwant %+v", got, want)
	}
}

// Critical regression: with insertion in the middle, every later line
// must still pair up to its equal counterpart, not slip out of phase.
func TestAlignLines_NoCascadingShift(t *testing.T) {
	left := []string{"a", "b", "c", "d"}
	right := []string{"a", "INSERTED", "b", "c", "d"}
	got := AlignLines(left, right)
	want := []DiffPair{
		{Op: OpEqual, Left: "a", Right: "a"},
		{Op: OpInsert, Left: "", Right: "INSERTED"},
		{Op: OpEqual, Left: "b", Right: "b"},
		{Op: OpEqual, Left: "c", Right: "c"},
		{Op: OpEqual, Left: "d", Right: "d"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v\nwant %+v", got, want)
	}
}
