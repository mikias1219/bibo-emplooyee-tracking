package orgemp

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/pbkdf2"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"testing"
)

func TestMapProfileActiveJob(t *testing.T) {
	var raw any
	_ = json.Unmarshal([]byte(`{"data":{
		"id":"6f1c3c2e-8a1b-4c55-9e0e-2b1f4a9d7c11","firebaseId":"fb1","email":"Abebe@Example.com",
		"firstName":"Abebe","middleName":"","lastName":"Kebede","tenantId":"t1",
		"role":{"slug":"Owner","name":"Owner"},"deletedAt":null,
		"employeeJobInformation":[
			{"isPositionActive":false,"departmentId":"old"},
			{"isPositionActive":true,"departmentLeadOrNot":true,
			 "department":{"id":"d1","name":"Engineering"},"position":{"name":"Lead Dev"}}]}}`), &raw)

	p := mapProfile(unwrapRecord(raw))
	if p.ID == "" || p.FirebaseUID != "fb1" || p.Email != "abebe@example.com" || p.TenantID != "t1" {
		t.Fatalf("identity fields: %+v", p)
	}
	if p.FullName != "Abebe Kebede" || p.RoleSlug != "Owner" || p.Deleted {
		t.Fatalf("name/role: %+v", p)
	}
	if p.DepartmentID != "d1" || p.DepartmentName != "Engineering" || p.JobTitle != "Lead Dev" || !p.IsDepartmentLead {
		t.Fatalf("job: %+v", p)
	}
}

func TestUnwrapListShapes(t *testing.T) {
	for _, body := range []string{
		`[{"id":"a","email":"a@x"}]`,
		`{"data":[{"id":"a","email":"a@x"}]}`,
		`{"data":{"items":[{"id":"a","email":"a@x"}]}}`,
		`{"data":{"data":[{"id":"a","email":"a@x"}]}}`,
	} {
		var raw any
		_ = json.Unmarshal([]byte(body), &raw)
		if got := unwrapList(raw); len(got) != 1 {
			t.Errorf("%s → %d items", body, len(got))
		}
	}
}

// encryptLikeNode reproduces Operation's EncryptionService.encryptText.
func encryptLikeNode(t *testing.T, key, ivHex, salt, plain string) string {
	k, _ := pbkdf2.Key(sha256.New, key, nodeHexUTF8(salt), 1000, 32)
	iv, _ := hex.DecodeString(ivHex)
	block, _ := aes.NewCipher(k)
	gcm, err := cipher.NewGCMWithNonceSize(block, len(iv))
	if err != nil {
		t.Fatal(err)
	}
	return ivHex + ":" + hex.EncodeToString(gcm.Seal(nil, iv, []byte(plain), nil))
}

func TestDecryptRoundTrip(t *testing.T) {
	const key, iv = "secret-key", "00112233445566778899aabbccddeeff"
	for _, salt := range []string{"default-salt", "73616c74"} {
		d, err := NewDecrypter(key, iv, salt)
		if err != nil || d == nil {
			t.Fatalf("decrypter: %v", err)
		}
		enc := encryptLikeNode(t, key, iv, salt, `{"id":"x"}`)
		if !LooksEncrypted(enc) {
			t.Fatal("not detected as encrypted")
		}
		got, err := d.Decrypt(enc)
		if err != nil || got != `{"id":"x"}` {
			t.Fatalf("salt %q: got %q err %v", salt, got, err)
		}
	}
}

func TestNodeHexUTF8DefaultSalt(t *testing.T) {
	// Buffer.from('default-salt','hex') = [0xde,0xfa] → two U+FFFD in UTF-8.
	if got := hex.EncodeToString(nodeHexUTF8("default-salt")); got != "efbfbdefbfbd" {
		t.Fatalf("got %s", got)
	}
}
