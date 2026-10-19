// Focus bridge for the Workforce Intelligence desktop agent on GNOME Wayland,
// where apps cannot read other windows themselves. Exposes ONLY the focused
// application's identity on the session bus — never window titles, screen
// content or input. Read-only: nothing here changes the desktop.
//
// D-Bus: com.workforceintelligence.Focus1 at /com/workforceintelligence/Focus
//   GetFocusedApp()     -> s app_id                 (e.g. "org.gnome.Console.desktop")
//   GetFocusedAppInfo() -> (s app_id, s name, i pid) (name e.g. "Console")
//   signal FocusChanged(s app_id)
import Gio from 'gi://Gio';
import GLib from 'gi://GLib';
import Shell from 'gi://Shell';
import {Extension} from 'resource:///org/gnome/shell/extensions/extension.js';

const BUS_NAME = 'com.workforceintelligence.Focus1';
const OBJECT_PATH = '/com/workforceintelligence/Focus';
const IFACE = `
<node>
  <interface name="com.workforceintelligence.Focus1">
    <method name="GetFocusedApp">
      <arg name="app_id" type="s" direction="out"/>
    </method>
    <method name="GetFocusedAppInfo">
      <arg name="app_id" type="s" direction="out"/>
      <arg name="name" type="s" direction="out"/>
      <arg name="pid" type="i" direction="out"/>
    </method>
    <signal name="FocusChanged">
      <arg name="app_id" type="s"/>
    </signal>
  </interface>
</node>`;

// Identifiers only: strip control characters and cap the length.
const clean = s => String(s ?? '').replace(/[\x00-\x1f\x7f]/g, '').slice(0, 256);

export default class FocusBridgeExtension extends Extension {
    enable() {
        this._dbus = Gio.DBusExportedObject.wrapJSObject(IFACE, this);
        this._dbus.export(Gio.DBus.session, OBJECT_PATH);
        this._nameId = Gio.bus_own_name_on_connection(
            Gio.DBus.session, BUS_NAME, Gio.BusNameOwnerFlags.NONE, null, null);
        this._focusId = global.display.connect('notify::focus-window', () => this._emitFocus());
    }

    disable() {
        if (this._focusId) {
            global.display.disconnect(this._focusId);
            this._focusId = 0;
        }
        if (this._nameId) {
            Gio.bus_unown_name(this._nameId);
            this._nameId = 0;
        }
        this._dbus?.unexport();
        this._dbus = null;
    }

    GetFocusedApp() {
        return this._info()[0];
    }

    GetFocusedAppInfo() {
        return this._info();
    }

    _info() {
        const win = global.display.get_focus_window();
        if (!win)
            return ['', '', 0];
        const app = Shell.WindowTracker.get_default().get_window_app(win);
        const wmClass = win.get_wm_class?.() ?? '';
        const appId = app?.get_id?.() || win.get_gtk_application_id?.() || wmClass;
        const name = app?.get_name?.() || wmClass || appId;
        return [clean(appId), clean(name), win.get_pid?.() ?? 0];
    }

    _emitFocus() {
        this._dbus?.emit_signal('FocusChanged', new GLib.Variant('(s)', [this._info()[0]]));
    }
}
