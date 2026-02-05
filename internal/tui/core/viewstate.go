package core

// ViewState represents which view is currently active.
type ViewState int

const (
	ViewLoading ViewState = iota
	ViewPRList
	ViewPRDetail
	ViewDiff
	ViewReview
	ViewHelp
	ViewRepoSwitch
	ViewInbox
)
