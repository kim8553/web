//go:build ignore

package main

import (
 "crypto/sha256"
 "encoding/hex"
 "os"
 "path/filepath"
 "testing"
)

func TestCurrentShopExchangeResourceFingerprintRejectsChanges(t *testing.T) {
 path:=filepath.Join(t.TempDir(),"shop.ini")
 original:=[]byte("[authoritative]\nrow=unchanged\n")
 if err:=os.WriteFile(path,original,0600);err!=nil {t.Fatal(err)}
 sum:=sha256.Sum256(original)
 expected:=hex.EncodeToString(sum[:])
 if err:=requireFileSHA256(path,"bad-fingerprint",expected);err!=nil {t.Fatalf("matching resource rejected: %v",err)}
 if err:=os.WriteFile(path,[]byte("[authoritative]\nrow=changed\n"),0600);err!=nil {t.Fatal(err)}
 if err:=requireFileSHA256(path,expected);err==nil {t.Fatal("changed resource passed cached fingerprint")}
 if err:=os.Remove(path);err!=nil {t.Fatal(err)}
 if err:=requireFileSHA256(path,expected);err==nil {t.Fatal("missing resource passed fingerprint")}
}
