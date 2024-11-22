package app

import (
	"context"
	"syscall"
	"unsafe"

	"github.com/wailsapp/wails/v2/pkg/runtime"
	"golang.org/x/sys/windows"
)

const (
	// GWLP_WNDPROC is used with GetWindowLongPtrW and SetWindowLongPtrW to retrieve and overwrite a window's WndProc.
	// Its value, -4 in two's complement, is defined here explicitly as a uintptr to avoid compiler overflow warnings
	// when converting to an unsigned type.
	// This is safe as long as we only target 64-bit architectures.
	GWLP_WNDPROC = uintptr(0xFFFFFFFFFFFFFFFC)

	// WM_ENDSESSION message informs the application about a session ending.
	//
	// For more message number identifiers, see https://gitlab.winehq.org/wine/wine/-/wikis/Wine-Developer's-Guide/List-of-Windows-Messages.
	WM_ENDSESSION       = 0x0016
	ENDSESSION_CLOSEAPP = 0x1
)

var (
	modUser32 = windows.NewLazySystemDLL("user32.dll")

	procEnumWindows              = modUser32.NewProc("EnumWindows")
	procGetWindowThreadProcessId = modUser32.NewProc("GetWindowThreadProcessId")
	procGetWindowLongPtrW        = modUser32.NewProc("GetWindowLongPtrW")
	procSetWindowLongPtrW        = modUser32.NewProc("SetWindowLongPtrW")
	procCallWindowProcW          = modUser32.NewProc("CallWindowProcW")
)

func runShutdownOnWmEndsession(ctx context.Context) {
	processId := windows.GetCurrentProcessId()
	windowHandle := findWindowByProcessId(processId)
	originalWndProc := getWindowProcPointer(windowHandle)

	newWndProc := func(hwnd windows.Handle, msg uint32, wParam, lParam uintptr) uintptr {
		// lParam: ENDSESSION_CLOSEAPP && wParam: FALSE identifies a condition where the application should not shut down:
		// https://learn.microsoft.com/en-us/windows/win32/shutdown/wm-endsession#parameters
		if msg == WM_ENDSESSION && !(lParam == ENDSESSION_CLOSEAPP && wParam == 0) {
			runtime.Quit(ctx)
			// https://learn.microsoft.com/en-us/windows/win32/shutdown/wm-endsession#return-value
			return 0
		}

		// Let Wails's WndProc handle other messages.
		return callWindowProc(originalWndProc, hwnd, msg, wParam, lParam)
	}

	subclassWndProc(windowHandle, newWndProc)
}

func findWindowByProcessId(processId uint32) windows.Handle {
	var targetHwnd windows.Handle
	cb := func(hwnd windows.Handle, _ uintptr) uintptr {
		wndProcessId := getWindowProcessId(hwnd)
		if wndProcessId == processId {
			targetHwnd = hwnd
			return 0
		}
		return 1
	}
	procEnumWindows.Call(syscall.NewCallback(cb), 0)
	return targetHwnd
}

func getWindowProcPointer(hwnd windows.Handle) uintptr {
	wndProc, _, _ := procGetWindowLongPtrW.Call(uintptr(hwnd), GWLP_WNDPROC)
	return wndProc
}

func getWindowProcessId(hwnd windows.Handle) uint32 {
	var processId uint32
	procGetWindowThreadProcessId.Call(
		uintptr(hwnd),
		uintptr(unsafe.Pointer(&processId)),
	)
	return processId
}

func callWindowProc(lpPrevWndFunc uintptr, hwnd windows.Handle, msg uint32, wParam, lParam uintptr) uintptr {
	ret, _, _ := procCallWindowProcW.Call(
		lpPrevWndFunc,
		uintptr(hwnd),
		uintptr(msg),
		wParam,
		lParam,
	)
	return ret
}

func subclassWndProc(hwnd windows.Handle, fn any) {
	procSetWindowLongPtrW.Call(
		uintptr(hwnd),
		GWLP_WNDPROC,
		syscall.NewCallback(fn),
	)
}
