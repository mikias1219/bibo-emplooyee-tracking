//! Content-free wire contract. NEVER add key codes, button names, coordinates or text.
use serde::{Deserialize, Serialize};

pub const VERSION: u8 = 1;
pub const MAX_FRAME_BYTES: usize = 4096;

#[derive(Clone, Debug, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all="snake_case")]
pub enum Coverage { Complete, Partial, Unavailable }

#[derive(Clone, Debug, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all="snake_case")]
pub enum GapReason { Paused, Locked, InactiveSession, PermissionDenied, DeviceLost, InputOverflow, ServiceDisconnected }

#[derive(Clone, Debug, Serialize, Deserialize)]
#[serde(deny_unknown_fields)]
pub struct InputBucket {
    pub client_uuid: String,
    pub start_ts: i64,
    pub interval_s: u16,
    pub keyboard_presses: u32,
    pub mouse_clicks: u32,
    pub pointer_motion_packets: u32,
    pub covered_ms: u32,
    pub coverage: Coverage,
    pub gap_reason: Option<GapReason>,
    pub source: String,
    pub updated_at: i64,
}
impl GapReason {
    /// The snake_case wire/storage name (matches the serde representation).
    pub fn as_str(&self) -> &'static str {
        match self {
            GapReason::Paused => "paused",
            GapReason::Locked => "locked",
            GapReason::InactiveSession => "inactive_session",
            GapReason::PermissionDenied => "permission_denied",
            GapReason::DeviceLost => "device_lost",
            GapReason::InputOverflow => "input_overflow",
            GapReason::ServiceDisconnected => "service_disconnected",
        }
    }
}

impl InputBucket {
    pub fn validate(&self) -> Result<(), &'static str> {
        if !(1..=60).contains(&self.interval_s) { return Err("interval_s must be 1..60"); }
        if self.covered_ms > (self.interval_s as u32) * 1000 { return Err("covered_ms exceeds interval"); }
        if self.coverage == Coverage::Complete && self.covered_ms != (self.interval_s as u32)*1000 { return Err("complete requires full coverage"); }
        if self.coverage == Coverage::Unavailable && self.covered_ms != 0 { return Err("unavailable requires zero coverage"); }
        if self.coverage != Coverage::Complete && self.gap_reason.is_none() { return Err("incomplete coverage requires reason"); }
        if self.coverage == Coverage::Complete && self.gap_reason.is_some() { return Err("complete bucket cannot have gap_reason"); }
        if self.source != "linux-evdev-v1" { return Err("unsupported source"); }
        Ok(())
    }
}

#[derive(Debug, Deserialize, Serialize)]
#[serde(tag="type", rename_all="snake_case", deny_unknown_fields)]
pub enum ClientMessage { Hello { version: u8 }, Begin { session: String }, Pause, Resume, Stop, Ping }
#[derive(Debug, Deserialize, Serialize)]
#[serde(tag="type", rename_all="snake_case")]
pub enum ServerMessage { Ready { version: u8 }, Status { state: String }, Bucket { bucket: InputBucket }, Error { code: String } }

#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn rejects_fake_complete_coverage() {
        let b = InputBucket {client_uuid:"x".into(),start_ts:0,interval_s:60,keyboard_presses:0,mouse_clicks:0,pointer_motion_packets:0,covered_ms:100,coverage:Coverage::Complete,gap_reason:None,source:"linux-evdev-v1".into(),updated_at:0};
        assert!(b.validate().is_err());
    }
    #[test]
    fn gap_reason_names_match_serde() {
        for r in [GapReason::Paused, GapReason::Locked, GapReason::InactiveSession, GapReason::PermissionDenied,
                  GapReason::DeviceLost, GapReason::InputOverflow, GapReason::ServiceDisconnected] {
            assert_eq!(serde_json::to_value(&r).unwrap(), r.as_str());
        }
    }
    #[test]
    fn refuses_unknown_sensitive_fields() {
        let b = r#"{"client_uuid":"x","start_ts":0,"interval_s":60,"keyboard_presses":0,"mouse_clicks":0,"pointer_motion_packets":0,"covered_ms":60000,"coverage":"complete","gap_reason":null,"source":"linux-evdev-v1","updated_at":0,"keycode":42}"#;
        assert!(serde_json::from_str::<InputBucket>(b).is_err());
    }
}
