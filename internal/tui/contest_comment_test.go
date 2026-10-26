package tui

import (
	"strings"
	"testing"

	"github.com/szporwolik/cqops/internal/config"
)

// contestModel returns a minimal model with one contest defined, ready to be
// activated by the individual tests.
func contestModel() *Model {
	m := newTestModel()
	m.App.Config.Contests = map[string]config.Contest{
		"c1": {Name: "CQ WPX CW", ContestID: "CQ-WPX-CW"},
		"c2": {Name: "SP DX Contest", ContestID: "SPDX"},
	}
	return m
}

func comment(m *Model) string { return strings.TrimSpace(m.fields[fieldComment].Value()) }

// TestContestCommentPrefilledOnNewQSO: with a contest active, a fresh QSO form
// carries the contest name in the comment.
func TestContestCommentPrefilledOnNewQSO(t *testing.T) {
	m := contestModel()
	m.App.Logbook.ActiveContest = "c1"

	m.resetQSOFields()
	if got := comment(m); got != "CQ WPX CW" {
		t.Fatalf("comment = %q, want the active contest name", got)
	}

	// A second new QSO keeps it.
	m.fields[fieldComment].SetValue("")
	m.resetQSOFields()
	if got := comment(m); got != "CQ WPX CW" {
		t.Fatalf("comment after a second reset = %q, want %q", got, "CQ WPX CW")
	}
}

// TestContestCommentNeverOverwritesOperatorText: a comment typed by the
// operator survives every prefill path.
func TestContestCommentNeverOverwritesOperatorText(t *testing.T) {
	m := contestModel()
	m.App.Logbook.ActiveContest = "c1"
	m.applyContestComment()

	m.fields[fieldComment].SetValue("via QSL only")
	m.applyContestComment()    // tick
	m.prefillContestExchange() // call commit / save
	m.contestCommentFor = ""   // even a forced re-evaluation
	m.applyContestComment()
	if got := comment(m); got != "via QSL only" {
		t.Fatalf("comment = %q, want the operator text untouched", got)
	}
}

// TestContestCommentFollowsContestSwitch: the tag written for the previous
// contest is replaced (but operator text is not).
func TestContestCommentFollowsContestSwitch(t *testing.T) {
	m := contestModel()
	m.App.Logbook.ActiveContest = "c1"
	m.applyContestComment()
	if got := comment(m); got != "CQ WPX CW" {
		t.Fatalf("precondition: comment = %q", got)
	}

	m.App.Logbook.ActiveContest = "c2"
	m.applyContestComment()
	if got := comment(m); got != "SP DX Contest" {
		t.Fatalf("comment after switching contest = %q, want the new contest name", got)
	}

	// Operator text survives a switch.
	m.fields[fieldComment].SetValue("my note")
	m.App.Logbook.ActiveContest = "c1"
	m.applyContestComment()
	if got := comment(m); got != "my note" {
		t.Fatalf("operator text was replaced on a contest switch: %q", got)
	}
}

// TestContestCommentRemovedWhenContestCleared: clearing the active contest
// removes the tag but leaves operator text alone.
func TestContestCommentRemovedWhenContestCleared(t *testing.T) {
	m := contestModel()
	m.App.Logbook.ActiveContest = "c1"
	m.applyContestComment()

	m.App.Logbook.ActiveContest = ""
	m.applyContestComment()
	if got := comment(m); got != "" {
		t.Fatalf("comment = %q after clearing the contest, want empty", got)
	}

	// Re-tag, then replace with operator text, then clear: text stays.
	m.App.Logbook.ActiveContest = "c1"
	m.contestCommentFor = ""
	m.applyContestComment()
	m.fields[fieldComment].SetValue("kept")
	m.App.Logbook.ActiveContest = ""
	m.applyContestComment()
	if got := comment(m); got != "kept" {
		t.Fatalf("operator text was removed with the contest tag: %q", got)
	}
}

// TestContestCommentRespectsKeepComment: a comment retained across QSOs is the
// operator's choice and is not replaced by the contest name.
func TestContestCommentRespectsKeepComment(t *testing.T) {
	m := contestModel()
	m.App.Logbook.ActiveContest = "c1"
	m.keepComment = true
	m.fields[fieldComment].SetValue("retained note")

	m.resetQSOFields()
	if got := comment(m); got != "retained note" {
		t.Fatalf("retained comment = %q, want it preserved", got)
	}
}

// TestContestCommentNoContestLeavesFieldAlone: without an active contest the
// comment is not touched at all.
func TestContestCommentNoContestLeavesFieldAlone(t *testing.T) {
	m := contestModel()
	m.fields[fieldComment].SetValue("plain")
	m.resetQSOFields()

	m.fields[fieldComment].SetValue("plain")
	m.applyContestComment()
	if got := comment(m); got != "plain" {
		t.Fatalf("comment = %q, want %q", got, "plain")
	}
}

// TestContestCommentNotReAddedAfterDeliberateClear: clearing the tag during a
// contest sticks until a new QSO starts (the tick must not fight the operator).
func TestContestCommentNotReAddedAfterDeliberateClear(t *testing.T) {
	m := contestModel()
	m.App.Logbook.ActiveContest = "c1"
	m.applyContestComment()

	m.fields[fieldComment].SetValue("")
	m.applyContestComment() // tick
	m.prefillContestExchange()
	if got := comment(m); got != "" {
		t.Fatalf("comment = %q, want the deliberate clear respected", got)
	}

	// A new QSO brings the tag back.
	m.resetQSOFields()
	if got := comment(m); got != "CQ WPX CW" {
		t.Fatalf("comment after a new QSO = %q, want the contest name", got)
	}
}
