package tui

// uiComponents holds all menu, editor, chooser, and sub-model pointers
// that represent distinct UI sub-screens or overlays.
type uiComponents struct {
	chooser         *LogbookChooser
	rigChooser      *RigChooser
	configMenu      *GeneralMenu
	callbookMenu    *CallbookMenu
	integrationMenu *IntegrationMenu
	notifMenu       *NotificationsMenu
	mainMenu        *MainMenu
	logViewer       *LogViewer
	logbookEditor   *LogbookEditor
}
