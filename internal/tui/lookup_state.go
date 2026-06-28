package tui

import (
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/szporwolik/cqops/internal/qrz"
	"github.com/szporwolik/cqops/internal/wavelog"
)

// lookupState holds QRZ and Wavelog lookup status, pending request flags,
// results, and last-looked-up state for partner/callbook display.
type lookupState struct {
	// QRZ/callbook lookup.
	qrzOnline     bool
	qrzForceCheck bool
	qrzNeed       bool
	qrzCall       string
	qrzLast       time.Time
	qrzLastCall   string

	// Wavelog connectivity and private lookup.
	wlOnline       bool
	wlForceCheck   bool
	wlFailCount    int       // consecutive connection failures
	wlNextRetry    time.Time // next retry attempt after backoff
	wlStationName  string
	wlStationLabel string
	wlNeed         bool
	wlCall         string
	wlLast         time.Time
	wlLastCall     string

	// Partner/callbook display data.
	partnerData    *qrz.CallData
	qrzLookupDone  bool
	qrzLookupCall  string // call the QRZ done flag is for
	wlPrivateData  *wavelog.PrivateLookupResult
	wlLookupDone   bool
	wlLookupCall   string // call the WL done flag is for
	wlLastBand     string
	wlLastMode     string
	wlDispatchTime time.Time // last time a WL lookup was dispatched; for timeout

	// pendingLookupCmd is set by onFieldExit when the call field is left
	// via Tab/arrows. handleFormKey batches it so lookups fire immediately.
	pendingLookupCmd tea.Cmd
}
