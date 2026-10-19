//! Menu bar (status bar) item: shows tracking state and offers Start / Stop /
//! Open main UI / Quit. Also the single place pause state is changed, so the tray,
//! the dashboard pill, and the trackers stay in sync.
//!
//! Indicator (glanceable, updated every couple seconds):
//!   🟢 tracking   🟡 idle (present, not counting active time)   🔴 paused

use std::sync::atomic::{AtomicBool, AtomicU8, Ordering};
use std::sync::Arc;
use std::thread;
use std::time::Duration;

use tauri::{
    menu::{Menu, MenuItem, PredefinedMenuItem},
    tray::TrayIconBuilder,
    AppHandle, Emitter, Manager,
};

use crate::trackers::TrackerControl;

const TRAY_ID: &str = "main";

/// Handles to the Start/Stop items, so refresh() can enable/disable them.
/// Managed in Tauri state.
struct MenuItems {
    start: MenuItem<tauri::Wry>,
    stop: MenuItem<tauri::Wry>,
}

/// True while the user is on the welcome/login/onboarding surfaces (not the
/// dashboard). Start stays disabled there — tracking is resumed by finishing
/// setup, not from the tray. Defaults to true until the UI reports otherwise.
static IN_SETUP: AtomicBool = AtomicBool::new(true);

/// Native (tray) strings. The app is English-only.
fn tr(key: &str) -> &str {
    match key {
        "open" => "Open main UI",
        "start" => "Start",
        "stop" => "Stop",
        "version" => "Version",
        "quit" => "Quit BiBoTracking",
        "tip_tracking" => "tracking",
        "tip_idle" => "idle (not counting)",
        "tip_paused" => "paused",
        _ => key,
    }
}

#[derive(Clone, Copy, PartialEq)]
enum State {
    Tracking,
    Idle,
    Paused,
}

impl State {
    fn code(self) -> u8 {
        match self {
            State::Tracking => 0,
            State::Idle => 1,
            State::Paused => 2,
        }
    }
    fn label(self) -> &'static str {
        match self {
            State::Tracking => "tracking",
            State::Idle => "idle",
            State::Paused => "paused",
        }
    }
}

/// Last broadcast state code, so we only emit on change. 255 = "unset".
static LAST_STATE: AtomicU8 = AtomicU8::new(255);

// State-colored menu-bar glyphs (not template images, so the tint shows).
const ICON_TRACKING: &[u8] = include_bytes!("../../icons/tray/tray-tracking.png");
const ICON_IDLE: &[u8] = include_bytes!("../../icons/tray/tray-idle.png");
const ICON_PAUSED: &[u8] = include_bytes!("../../icons/tray/tray-paused.png");

fn icon_for(state: State) -> tauri::image::Image<'static> {
    let bytes = match state {
        State::Tracking => ICON_TRACKING,
        State::Idle => ICON_IDLE,
        State::Paused => ICON_PAUSED,
    };
    tauri::image::Image::from_bytes(bytes).expect("decode tray icon")
}

/// Build the tray icon + menu and start the status updater. Call once during setup.
pub fn build(app: &AppHandle, control: Arc<TrackerControl>) -> tauri::Result<()> {
    let open = MenuItem::with_id(app, "open", tr("open"), true, None::<&str>)?;
    // Start resumes tracking, but only from the dashboard (disabled during
    // welcome/login/onboarding — see IN_SETUP). BRI-22
    let start = MenuItem::with_id(app, "start", tr("start"), false, None::<&str>)?;
    let stop = MenuItem::with_id(app, "stop", tr("stop"), true, None::<&str>)?;
    let version = MenuItem::with_id(
        app,
        "version",
        version_label(app),
        false,
        None::<&str>,
    )?;
    let quit = MenuItem::with_id(app, "quit", tr("quit"), true, None::<&str>)?;
    let menu = Menu::with_items(
        app,
        &[
            &open,
            &PredefinedMenuItem::separator(app)?,
            &start,
            &stop,
            &PredefinedMenuItem::separator(app)?,
            &version,
            &quit,
        ],
    )?;

    TrayIconBuilder::with_id(TRAY_ID)
        .icon(icon_for(State::Tracking))
        .icon_as_template(false) // keep our state tint colors
        .menu(&menu)
        .on_menu_event(|app, event| match event.id().as_ref() {
            "open" => show_main(app),
            "start" => set_paused(app, false),
            "stop" => set_paused(app, true),
            "quit" => app.exit(0),
            _ => {}
        })
        .build(app)?;

    // Keep handles so refresh() can enable/disable Start vs Stop.
    app.manage(MenuItems { start, stop });

    refresh(app);
    start_status_updater(app.clone(), control);
    Ok(())
}

/// "Version 1.4.1" — localized word + the app version from tauri.conf.json.
fn version_label(app: &AppHandle) -> String {
    format!("{} {}", tr("version"), app.package_info().version)
}

/// Record whether the user is still in setup (welcome/login/onboarding) and pause
/// tracking accordingly; on the dashboard tracking resumes and Start becomes usable.
pub fn set_in_setup(app: &AppHandle, in_setup: bool) {
    IN_SETUP.store(in_setup, Ordering::Relaxed);
    set_paused(app, in_setup);
}

/// Show + focus the main window (it may be hidden in menu-bar-only mode).
pub fn show_main(app: &AppHandle) {
    if let Some(w) = app.get_webview_window("main") {
        let _ = w.show();
        let _ = w.set_focus();
    }
}

/// The single source of truth for changing pause state: updates the trackers, the
/// tray indicator, and notifies the UI via an event.
pub fn set_paused(app: &AppHandle, paused: bool) {
    if let Some(c) = app.try_state::<Arc<TrackerControl>>() {
        c.paused.store(paused, Ordering::Relaxed);
    }
    refresh(app); // emits the new tracking-state + updates the badge
}

/// Recompute the current state and update the tray (dispatched to the main thread,
/// since AppKit status-item updates must happen there).
pub fn refresh(app: &AppHandle) {
    let (paused, threshold) = match app.try_state::<Arc<TrackerControl>>() {
        Some(c) => (
            c.paused.load(Ordering::Relaxed),
            c.idle_threshold_s.load(Ordering::Relaxed) as f64,
        ),
        None => (false, 60.0),
    };
    let state = if paused {
        State::Paused
    } else if crate::platform::idle_seconds() >= threshold {
        State::Idle
    } else {
        State::Tracking
    };

    // Broadcast to the UI only when the state actually changes.
    if LAST_STATE.swap(state.code(), Ordering::Relaxed) != state.code() {
        let _ = app.emit("tracking-state", state.label());
    }

    let app2 = app.clone();
    let _ = app.run_on_main_thread(move || render(&app2, state));
}

fn render(app: &AppHandle, state: State) {
    if let Some(tray) = app.tray_by_id(TRAY_ID) {
        let word = match state {
            State::Tracking => tr("tip_tracking"),
            State::Idle => tr("tip_idle"),
            State::Paused => tr("tip_paused"),
        };
        let tip = format!("BiBoTracking — {word}");
        // The glyph's tint conveys the state — no separate dot/badge needed.
        let _ = tray.set_icon(Some(icon_for(state)));
        let _ = tray.set_tooltip(Some(tip));
    }

    // Stop is available only while running (tracking/idle); Start only while
    // paused AND on the dashboard (never from the setup surfaces).
    let paused = state == State::Paused;
    if let Some(items) = app.try_state::<MenuItems>() {
        let _ = items.start.set_enabled(paused && !IN_SETUP.load(Ordering::Relaxed));
        let _ = items.stop.set_enabled(!paused);
    }
}

fn start_status_updater(app: AppHandle, _control: Arc<TrackerControl>) {
    thread::spawn(move || loop {
        thread::sleep(Duration::from_secs(2));
        refresh(&app);
    });
}
