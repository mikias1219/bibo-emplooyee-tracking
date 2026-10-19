//! Linux platform backend (see docs/WORKFORCE_INTELLIGENCE_PLAN.md §3).
//!
//! Selected by `#[cfg(target_os = "linux")]` in `platform/mod.rs`. Wayland
//! deliberately hides other apps' windows and global input from normal apps, so
//! each signal uses the desktop's own session-bus service and anything that
//! can't be read honestly is reported as unavailable — never faked:
//!
//! - Idle time: GNOME `org.gnome.Mutter.IdleMonitor` (X11 + Wayland), falling
//!   back to `org.freedesktop.ScreenSaver.GetSessionIdleTime` (KDE and others).
//! - Active app: on GNOME Wayland, our own Shell extension
//!   (`apps/desktop/linux/gnome-extension`) exposes only the focused app's id,
//!   name and pid — never window titles. On X11, `active-win-pos-rs` works.
//! - Keystroke counts: unavailable (would need root access to /dev/input).
//! - Screenshots: X11 directly; on Wayland through the xdg-desktop-portal
//!   Screenshot interface (via `xcap`), which GNOME authorizes with a one-time
//!   user prompt. Full screen only — Wayland never exposes a single window.

use super::{ActiveWindowInfo, CapabilityRow, Permission, PermissionState};
use crate::settings::Settings;
use std::sync::OnceLock;
use zbus::blocking::Connection;

// Must match apps/desktop/linux/gnome-extension/.../extension.js.
const FOCUS_DEST: &str = "com.workforceintelligence.Focus1";
const FOCUS_PATH: &str = "/com/workforceintelligence/Focus";

fn is_wayland() -> bool {
    std::env::var("XDG_SESSION_TYPE").map(|v| v == "wayland").unwrap_or(false)
        || std::env::var_os("WAYLAND_DISPLAY").is_some()
}

/// The session bus, connected once; `None` when there is no session bus.
fn session() -> Option<&'static Connection> {
    static BUS: OnceLock<Option<Connection>> = OnceLock::new();
    BUS.get_or_init(|| Connection::session().ok()).as_ref()
}

/// The focused app from our GNOME Shell extension, if it is installed + enabled.
fn focused_app_from_extension() -> Option<ActiveWindowInfo> {
    let reply = session()?
        .call_method(Some(FOCUS_DEST), FOCUS_PATH, Some(FOCUS_DEST), "GetFocusedAppInfo", &())
        .ok()?;
    let (app_id, name, pid): (String, String, i32) = reply.body().deserialize().ok()?;
    let app_name = if !name.is_empty() {
        name
    } else {
        app_id.trim_end_matches(".desktop").to_string()
    };
    if app_name.is_empty() {
        return None;
    }
    Some(ActiveWindowInfo { app_name, title: None, pid: pid as i64 })
}

/// Whether the active-app source works in this session (drives the setup row).
fn active_app_available() -> bool {
    if is_wayland() {
        focused_app_from_extension().is_some()
    } else {
        active_win_pos_rs::get_active_window().is_ok()
    }
}

/// The active app. Window titles are never collected on Linux.
pub fn active_window() -> Option<ActiveWindowInfo> {
    if let Some(info) = focused_app_from_extension() {
        return Some(info);
    }
    if is_wayland() {
        // XWayland windows would be visible to X11 APIs but native ones not,
        // which gives misleading partial data — require the extension instead.
        return None;
    }
    let w = active_win_pos_rs::get_active_window().ok()?;
    Some(ActiveWindowInfo { app_name: w.app_name, title: None, pid: w.process_id as i64 })
}

/// Linux capture rows for the setup screen. Unavailable signals say so plainly.
pub fn capability_rows(s: &Settings) -> Vec<CapabilityRow> {
    let consented = |enabled: bool| {
        if s.consented && enabled {
            PermissionState::Granted
        } else {
            PermissionState::Denied
        }
    };
    let app_state = if active_app_available() {
        consented(true)
    } else {
        PermissionState::Denied
    };
    let app_desc = if is_wayland() {
        "Tracks which app is in focus (never window titles). On GNOME Wayland this needs \
         the Workforce Intelligence Focus Bridge extension enabled in the Extensions app."
    } else {
        "Tracks which app is in focus (never window titles)."
    };
    let shots_desc = if is_wayland() {
        "Full-screen screenshots through GNOME's screenshot permission (asked once). \
         Window-only capture is not possible on Wayland; skipped while idle or when \
         the focused app is unknown or on the skip-list."
    } else {
        "Captures periodic screenshots of your screen(s)."
    };
    vec![
        CapabilityRow {
            key: "activity".to_string(),
            label: "App activity & idle time".to_string(),
            description: app_desc.to_string(),
            state: app_state,
            required: true,
            can_request: false,
            can_open_settings: false,
        },
        CapabilityRow {
            key: "keystrokes".to_string(),
            label: "Keystroke counts".to_string(),
            description: "Unavailable on Linux: counting needs root access to input devices."
                .to_string(),
            state: PermissionState::Denied,
            required: false,
            can_request: false,
            can_open_settings: false,
        },
        CapabilityRow {
            key: "screenshots".to_string(),
            label: "Screenshots".to_string(),
            description: shots_desc.to_string(),
            state: consented(s.capture_screenshots),
            required: false,
            can_request: false,
            can_open_settings: false,
        },
    ]
}

/// No per-feature settings pane to open on Linux.
pub fn open_settings(_p: Permission) {}

/// Keystroke counting is never available. Screen capture is authorized by the
/// desktop's screenshot portal itself (a refused grant just fails the capture).
pub fn permission_status(p: Permission) -> PermissionState {
    match p {
        Permission::InputMonitoring => PermissionState::Denied,
        _ => PermissionState::Granted,
    }
}

pub fn request_screen_recording() -> bool {
    true
}

/// Wayland screenshots are full screen, so a tick must be skipped unless we
/// know the focused app (to honour the skip-list) and the user is active.
pub fn screenshot_tick_allowed(idle_threshold_s: f64) -> bool {
    if !is_wayland() {
        return true;
    }
    idle_seconds() < idle_threshold_s && focused_app_from_extension().is_some()
}

pub fn request_input_monitoring() -> bool {
    false
}

pub fn request_accessibility() -> bool {
    true
}

/// No keyboard counting on Linux (see module docs). Never called while
/// `permission_status(InputMonitoring)` is `Denied`.
pub fn run_keyboard_tap() -> bool {
    false
}

/// Seconds since the last keyboard/mouse input. When no idle source answers the
/// result is `INFINITY`: unknown is reported as idle, never as active time.
pub fn idle_seconds() -> f64 {
    let Some(bus) = session() else { return f64::INFINITY };
    if let Ok(reply) = bus.call_method(
        Some("org.gnome.Mutter.IdleMonitor"),
        "/org/gnome/Mutter/IdleMonitor/Core",
        Some("org.gnome.Mutter.IdleMonitor"),
        "GetIdletime",
        &(),
    ) {
        if let Ok(ms) = reply.body().deserialize::<u64>() {
            return ms as f64 / 1000.0;
        }
    }
    // KDE reports milliseconds here despite the spec saying seconds.
    if let Ok(reply) = bus.call_method(
        Some("org.freedesktop.ScreenSaver"),
        "/org/freedesktop/ScreenSaver",
        Some("org.freedesktop.ScreenSaver"),
        "GetSessionIdleTime",
        &(),
    ) {
        if let Ok(ms) = reply.body().deserialize::<u32>() {
            return ms as f64 / 1000.0;
        }
    }
    f64::INFINITY
}
