import pathlib,sqlite3,unittest,uuid
SQL=(pathlib.Path(__file__).resolve().parents[1]/'sql/001_local_input.sql').read_text()
class TestSchema(unittest.TestCase):
 def setUp(self):
  self.db=sqlite3.connect(':memory:');self.db.executescript(SQL)
 def insert(self,coverage='complete',covered=60000,reason=None):
  self.db.execute('INSERT INTO input_activity_bucket(client_uuid,start_ts,interval_s,keyboard_presses,mouse_clicks,pointer_motion_packets,covered_ms,coverage,gap_reason,source,updated_at) VALUES (?,?,?,?,?,?,?,?,?,?,?)',(str(uuid.uuid4()),1,60,13,7,9,covered,coverage,reason,'linux-evdev-v1',2))
 def test_complete_roundtrip(self):
  self.insert();self.assertEqual(self.db.execute('SELECT synced,keyboard_presses FROM input_activity_bucket').fetchone(),(0,13))
 def test_incomplete_requires_reason(self):
  with self.assertRaises(sqlite3.IntegrityError):self.insert('partial',100,None)
 def test_complete_requires_full_coverage(self):
  with self.assertRaises(sqlite3.IntegrityError):self.insert('complete',100,None)
 def test_unavailable_zero_coverage(self):
  with self.assertRaises(sqlite3.IntegrityError):self.insert('unavailable',100,'device_lost')
 def test_client_uuid_required(self):
  self.insert()
  with self.assertRaises(sqlite3.IntegrityError):self.db.execute('UPDATE input_activity_bucket SET client_uuid=? WHERE id=1',(None,))
 def test_retry_uuid_unique(self):
  self.insert();u=self.db.execute('SELECT client_uuid FROM input_activity_bucket').fetchone()[0]
  with self.assertRaises(sqlite3.IntegrityError):
   self.db.execute('INSERT INTO input_activity_bucket(client_uuid,start_ts,interval_s,covered_ms,coverage,source,updated_at) VALUES (?,1,60,60000,?,?,2)',(u,'complete','linux-evdev-v1'))
if __name__=='__main__':unittest.main()
