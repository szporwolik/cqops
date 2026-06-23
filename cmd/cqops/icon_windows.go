//go:build windows

package main

import (
	"syscall"
	"unsafe"
)

var (
	kernel32             = syscall.NewLazyDLL("kernel32.dll")
	procGetModuleHandle  = kernel32.NewProc("GetModuleHandleW")
	procGetConsoleWindow = kernel32.NewProc("GetConsoleWindow")

	user32          = syscall.NewLazyDLL("user32.dll")
	procLoadIcon    = user32.NewProc("LoadIconW")
	procSendMessage = user32.NewProc("SendMessageW")
)

const (
	WM_SETICON  = 0x0080
	ICON_SMALL  = 0
	ICON_BIG    = 1
	IDI_APPICON = 1 // resource ID of the first icon in RT_GROUP_ICON
)

// setConsoleIcon loads the embedded icon resource and applies it to the
// console window so Windows Terminal shows the CQOps icon in the tab,
// taskbar, and Alt+Tab switcher.
func setConsoleIcon() {
	// Get the handle to the current console window.
	hwnd, _, _ := procGetConsoleWindow.Call()
	if hwnd == 0 {
		return
	}

	// Get the module handle (HMODULE) for our .exe so we can load resources.
	hInstance, _, _ := procGetModuleHandle.Call(0)
	if hInstance == 0 {
		return
	}

	// Load the icon from the embedded resource (IDI_APPICON = 1).
	// LoadIcon loads from the module's resources.
	hIcon, _, _ := procLoadIcon.Call(hInstance, uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr("#1"))))
	if hIcon == 0 {
		// Try MAKEINTRESOURCE(1) as numeric
		hIcon, _, _ = procLoadIcon.Call(hInstance, 1)
	}
	if hIcon == 0 {
		return
	}

	// Set both small (tab/taskbar) and large (Alt+Tab) icons.
	procSendMessage.Call(hwnd, WM_SETICON, ICON_SMALL, hIcon)
	procSendMessage.Call(hwnd, WM_SETICON, ICON_BIG, hIcon)
}

// This file's init is intentionally empty — setConsoleIcon is called
// explicitly from main.go to control timing.
