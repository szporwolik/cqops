package tui

import (
	"time"

	"github.com/szporwolik/cqops/internal/qrz"
	"github.com/szporwolik/cqops/internal/wavelog"
)

// lookupState holds QRZ and Wavelog lookup status, pending request flags,
// results, and last-looked-up state for partner/callbook display.
type lookupState struct {
	// QRZ/callbook lookup.
	qrzOnline   bool
	qrzNeed     bool
	qrzCall     string
	qrzLast     time.Time
	qrzLastCall string

	// Wavelog connectivity and private lookup.
	wlOnline       bool
	wlForceCheck   bool
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

	// pendingSave is set when Enter is pressed while lookups are still
	// in progress. The save fires automatically once both complete.
	pendingSave bool
}
