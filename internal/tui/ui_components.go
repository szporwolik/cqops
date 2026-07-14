package tui

// uiComponents holds all menu, editor, chooser, and sub-model pointers
// that represent distinct UI sub-screens or overlays.
type uiComponents struct {
	chooser         *LogbookChooser
	rigChooser      *RigChooser
	contestChooser  *ContestChooser
	operatorChooser *OperatorChooser
	configMenu      *GeneralMenu
	integrationMenu *IntegrationMenu
	callbookMenu    *CallbookMenu
	notifMenu       *NotificationsMenu
	mainMenu        *MainMenu
	logViewer       *LogViewer
	logbookEditor   *LogbookEditor
}
