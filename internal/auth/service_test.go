package auth
import "testing"
func TestPasswordAndSessionTokens(t *testing.T) {
 first,err:=HashPassword("a secure test password"); if err!=nil { t.Fatal(err) }
 second,err:=HashPassword("a secure test password"); if err!=nil { t.Fatal(err) }
 if first==second { t.Fatal("password salts must differ") }
 if !CheckPassword(first,"a secure test password") || CheckPassword(first,"wrong password") || CheckPassword("invalid","password") { t.Fatal("password verification failed") }
 if _,err:=HashPassword("short"); err==nil { t.Fatal("short passwords must fail") }
 one,two:=Token(),Token()
 if len(one)!=43 || one==two || len(TokenHash(one))!=32 { t.Fatal("invalid random session token") }
}
