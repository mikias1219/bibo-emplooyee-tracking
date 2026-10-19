//! SECURITY-SAFE PROTOTYPE: a peer-authenticated IPC service, NOT an input
//! capture implementation. No /dev/input access is enabled: device/seat
//! attestation and consent verification are required first (see README).
use std::{
    env, fs,
    io::{BufRead, BufReader, Read, Write},
    os::unix::{
        fs::PermissionsExt,
        io::AsRawFd,
        net::{UnixListener, UnixStream},
    },
    path::Path,
    time::Duration,
};
use workforce_input_protocol::{ClientMessage, ServerMessage, MAX_FRAME_BYTES, VERSION};

fn peer_uid(fd: i32) -> std::io::Result<u32> {
    let mut cred: libc::ucred = unsafe { std::mem::zeroed() };
    let mut len = std::mem::size_of::<libc::ucred>() as libc::socklen_t;
    let rc = unsafe {
        libc::getsockopt(
            fd,
            libc::SOL_SOCKET,
            libc::SO_PEERCRED,
            &mut cred as *mut _ as *mut _,
            &mut len,
        )
    };
    if rc != 0 {
        return Err(std::io::Error::last_os_error());
    }
    Ok(cred.uid)
}

fn respond(stream: &mut UnixStream, msg: &ServerMessage) -> std::io::Result<()> {
    let mut b = serde_json::to_vec(msg)?;
    b.push(b'\n');
    stream.write_all(&b)
}

/// Handle one client. Errors are returned to the accept loop, which logs them
/// and keeps serving — a misbehaving peer must never stop the daemon.
fn handle(mut s: UnixStream, approved_uid: Option<u32>) -> std::io::Result<()> {
    // A stalled client must not block the (sequential) accept loop.
    s.set_read_timeout(Some(Duration::from_secs(2)))?;
    s.set_write_timeout(Some(Duration::from_secs(2)))?;

    let uid = peer_uid(s.as_raw_fd())?;
    if Some(uid) != approved_uid {
        return respond(&mut s, &ServerMessage::Error { code: "unauthorized_peer".into() });
    }

    // Bounded read: never buffer more than one frame (+1 byte to detect overflow).
    let mut line = String::new();
    BufReader::new(s.try_clone()?.take(MAX_FRAME_BYTES as u64 + 1)).read_line(&mut line)?;
    if line.len() > MAX_FRAME_BYTES {
        return respond(&mut s, &ServerMessage::Error { code: "oversized_frame".into() });
    }

    match serde_json::from_str::<ClientMessage>(&line) {
        Ok(ClientMessage::Hello { version }) if version == VERSION => {
            respond(&mut s, &ServerMessage::Ready { version: VERSION })
        }
        Ok(ClientMessage::Ping) => respond(
            &mut s,
            &ServerMessage::Status { state: "collector_not_enrolled".into() },
        ),
        _ => respond(&mut s, &ServerMessage::Error { code: "not_enrolled_or_unsupported".into() }),
    }
}

fn main() -> std::io::Result<()> {
    let path = env::var("WI_INPUTD_SOCK")
        .unwrap_or_else(|_| "/run/workforce-inputd/inputd.sock".to_string());
    if let Some(parent) = Path::new(&path).parent() {
        fs::create_dir_all(parent)?;
    }
    if Path::new(&path).exists() {
        fs::remove_file(&path)?;
    }
    let listener = UnixListener::bind(&path)?;
    // Owner + group only; access is granted by an administrator-managed group,
    // never world-connectable regardless of the process umask.
    fs::set_permissions(&path, fs::Permissions::from_mode(0o660))?;

    // WI_ALLOWED_UID alone is NOT sufficient to authorize monitoring (README).
    let approved_uid = env::var("WI_ALLOWED_UID").ok().and_then(|x| x.parse::<u32>().ok());
    eprintln!("workforce-inputd IPC prototype; input collection disabled pending security integration");

    for accepted in listener.incoming() {
        let result = accepted.and_then(|s| handle(s, approved_uid));
        if let Err(e) = result {
            eprintln!("workforce-inputd: client error: {e}");
        }
    }
    Ok(())
}
