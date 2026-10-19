//! Bridge into BiBo's local DB. API integration into BiBo storage/sync is required.
use rusqlite::{params, Connection};
use workforce_input_protocol::InputBucket;

pub const LOCAL_MIGRATION: &str = include_str!("../../../sql/001_local_input.sql");

pub fn initialize(conn: &Connection) -> rusqlite::Result<()> { conn.execute_batch(LOCAL_MIGRATION) }

pub fn enqueue(conn: &Connection, b: &InputBucket) -> Result<(), Box<dyn std::error::Error>> {
    b.validate()?;
    let coverage = match b.coverage {
      workforce_input_protocol::Coverage::Complete => "complete",
      workforce_input_protocol::Coverage::Partial => "partial",
      workforce_input_protocol::Coverage::Unavailable => "unavailable",
    };
    // snake_case to match the wire format ("permission_denied", not "permissiondenied").
    let reason = b.gap_reason.as_ref().map(|r| r.as_str());
    conn.execute("INSERT INTO input_activity_bucket (client_uuid,start_ts,interval_s,keyboard_presses,mouse_clicks,pointer_motion_packets,covered_ms,coverage,gap_reason,source,updated_at,synced) VALUES (?1,?2,?3,?4,?5,?6,?7,?8,?9,?10,?11,0) ON CONFLICT(client_uuid) DO UPDATE SET keyboard_presses=excluded.keyboard_presses, mouse_clicks=excluded.mouse_clicks,pointer_motion_packets=excluded.pointer_motion_packets,covered_ms=excluded.covered_ms,coverage=excluded.coverage,gap_reason=excluded.gap_reason,updated_at=excluded.updated_at,synced=0",
      params![b.client_uuid,b.start_ts,b.interval_s,b.keyboard_presses,b.mouse_clicks,b.pointer_motion_packets,b.covered_ms,coverage,reason,b.source,b.updated_at])?;
    Ok(())
}

pub fn pending_json(conn: &Connection, limit: usize) -> rusqlite::Result<Vec<serde_json::Value>> {
 let mut st=conn.prepare("SELECT client_uuid,start_ts,interval_s,keyboard_presses,mouse_clicks,pointer_motion_packets,covered_ms,coverage,gap_reason,source,updated_at FROM input_activity_bucket WHERE synced=0 ORDER BY id LIMIT ?1")?;
 let rows = st.query_map(params![limit as i64],|row| {
   Ok(serde_json::json!({"client_uuid":row.get::<_,String>(0)?,"start_ts":row.get::<_,i64>(1)?,"interval_s":row.get::<_,u16>(2)?,"keyboard_presses":row.get::<_,u32>(3)?,"mouse_clicks":row.get::<_,u32>(4)?,"pointer_motion_packets":row.get::<_,u32>(5)?,"covered_ms":row.get::<_,u32>(6)?,"coverage":row.get::<_,String>(7)?,"gap_reason":row.get::<_,Option<String>>(8)?,"source":row.get::<_,String>(9)?,"updated_at":row.get::<_,i64>(10)?}))
 })?.collect();
 rows
}

pub fn acknowledge(conn:&Connection, ids:&[String])->rusqlite::Result<usize> {
 let tx=conn.unchecked_transaction()?;
 let mut n=0;
 for id in ids { n += tx.execute("UPDATE input_activity_bucket SET synced=1 WHERE client_uuid=?1",[id])?; }
 tx.commit()?;
 Ok(n)
}

#[cfg(test)]
mod tests {
 use super::*;
 use workforce_input_protocol::{Coverage, GapReason};
 #[test] fn migration_builds() {let c=Connection::open_in_memory().unwrap();initialize(&c).unwrap();}
 #[test] fn gap_reason_stored_snake_case_and_acked() {
  let c=Connection::open_in_memory().unwrap(); initialize(&c).unwrap();
  let b=InputBucket{client_uuid:"u1".into(),start_ts:0,interval_s:60,keyboard_presses:0,mouse_clicks:0,pointer_motion_packets:0,
    covered_ms:0,coverage:Coverage::Unavailable,gap_reason:Some(GapReason::PermissionDenied),source:"linux-evdev-v1".into(),updated_at:1};
  enqueue(&c,&b).unwrap();
  let p=pending_json(&c,10).unwrap();
  assert_eq!(p[0]["gap_reason"],"permission_denied");
  assert_eq!(acknowledge(&c,&["u1".into()]).unwrap(),1);
  assert!(pending_json(&c,10).unwrap().is_empty());
 }
}
