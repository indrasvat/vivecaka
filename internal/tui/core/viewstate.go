package core

// ViewState represents which view is currently active.
type ViewState int

const (
	ViewBanner ViewState = iota
	ViewLoading
	ViewPRList
	ViewPRDetail
	ViewDiff
	ViewReview
	ViewHelp
	ViewRepoSwitch
	ViewInbox
	ViewFilter
)
