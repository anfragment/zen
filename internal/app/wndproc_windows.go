package app

import (
	"context"
	"syscall"
	"unsafe"

	"github.com/wailsapp/wails/v2/pkg/runtime"
	"golang.org/x/sys/windows"
)

const (
	GWL_WNDPROC = uint(0xFFFFFFFFFFFFFFFC) // -4 in two's complement. Should be fine as long as we only support 64-bit architectures.

	WM_ENDSESSION = 0x0016
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
	processId := uint32(windows.GetCurrentProcessId())
	windowHandle := findWindowByProcessId(processId)
	originalWndProc := getWindowProcPointer(windowHandle)

	newWndProc := func(hwnd windows.Handle, msg uint32, wParam, lParam uintptr) uintptr {
		switch msg {
		case WM_ENDSESSION:
			runtime.Quit(ctx)
			// https://learn.microsoft.com/en-us/windows/win32/shutdown/wm-endsession#return-value
			return 0
		}

		return callWindowProc(originalWndProc, hwnd, msg, wParam, lParam)
	}

	subclassWndProc(windowHandle, newWndProc)
}

func findWindowByProcessId(processId uint32) windows.Handle {
	var targetHwnd windows.Handle
	cb := func(hwnd windows.Handle, _ uintptr) uintptr {
		_, wndProcessId := getWindowThreadProcessId(hwnd)
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
	wndProc, _, _ := procGetWindowLongPtrW.Call(uintptr(hwnd), uintptr(GWL_WNDPROC))
	return wndProc
}

func getWindowThreadProcessId(hwnd windows.Handle) (uint32, uint32) {
	var processID uint32
	threadID, _, _ := procGetWindowThreadProcessId.Call(
		uintptr(hwnd),
		uintptr(unsafe.Pointer(&processID)),
	)
	return uint32(threadID), processID
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
		uintptr(GWL_WNDPROC),
		syscall.NewCallback(fn),
	)
}
