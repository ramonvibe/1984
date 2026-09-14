package github
import (
 "crypto/hmac"
 "crypto/sha256"
 "encoding/hex"
 "reflect"
 "testing"
)
func TestSignatureAndReferences(t *testing.T) {
 secret:="this is a test webhook secret only"
 body:=[]byte("payload")
 mac:=hmac.New(sha256.New,[]byte(secret));mac.Write(body)
 signature:="sha256="+hex.EncodeToString(mac.Sum(nil))
 if !VerifySignature(secret,signature,body) || VerifySignature(secret,signature,[]byte("changed")) || VerifySignature("",signature,body) { t.Fatal("signature validation failed") }
 got:=References("fix(PLAT-142): auth, PLAT-142 and API-8; skip PLAT-0 and X-1")
 if !reflect.DeepEqual(got,[]string{"PLAT-142","API-8"}) { t.Fatal(got) }
}
