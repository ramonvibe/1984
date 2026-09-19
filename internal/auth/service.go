package auth

import (
 "context"
 "crypto/pbkdf2"
 "crypto/rand"
 "crypto/sha256"
 "crypto/subtle"
 "encoding/base64"
 "errors"
 "fmt"
 "net/mail"
 "strconv"
 "strings"
 "time"

 "github.com/jackc/pgx/v5"
 "github.com/jackc/pgx/v5/pgxpool"
 "github.com/ramon/trackline/internal/database"
 "github.com/ramon/trackline/internal/validate"
)

var ErrCredentials = errors.New("email ou senha inválidos")
var ErrForbidden = errors.New("acesso de administrador necessário")
type User = database.GetSessionUserRow
type Service struct { Pool *pgxpool.Pool; Queries *database.Queries }

func Token() string { return base64.RawURLEncoding.EncodeToString(randBytes(32)) }
func randBytes(size int) []byte { value:=make([]byte,size); if _,err:=rand.Read(value); err!=nil { panic(err) }; return value }
func TokenHash(token string) []byte { sum:=sha256.Sum256([]byte(token)); return sum[:] }

func HashPassword(password string) (string,error) {
 if len(password)<12 || len(password)>256 { return "",errors.New("a senha deve ter entre 12 e 256 caracteres") }
 salt:=randBytes(16)
 key,err:=pbkdf2.Key(sha256.New,password,salt,600000,32)
 if err!=nil { return "",err }
 return "pbkdf2-sha256$600000$"+base64.RawStdEncoding.EncodeToString(salt)+"$"+base64.RawStdEncoding.EncodeToString(key),nil
}
func CheckPassword(hash,password string) bool {
 parts:=strings.Split(hash,"$")
 if len(parts)!=4 || parts[0]!="pbkdf2-sha256" || len(password)>256 { return false }
 rounds,err:=strconv.Atoi(parts[1]); if err!=nil || rounds<100000 || rounds>1000000 { return false }
 salt,err:=base64.RawStdEncoding.DecodeString(parts[2]); if err!=nil || len(salt)!=16 { return false }
 expected,err:=base64.RawStdEncoding.DecodeString(parts[3]); if err!=nil || len(expected)!=32 { return false }
 key,err:=pbkdf2.Key(sha256.New,password,salt,rounds,32)
 return err==nil && subtle.ConstantTimeCompare(key,expected)==1
}
func identity(name,email string) (string,string,error) {
 name,err:=validate.Required("name",name,120); if err!=nil { return "","",err }
 email=strings.ToLower(strings.TrimSpace(email))
 address,err:=mail.ParseAddress(email)
 if err!=nil || address.Address!=email || len(email)>254 { return "","",errors.New("digite um email válido") }
 return name,email,nil
}

func (s Service) Setup(ctx context.Context,workspace,name,email,password string) (string,error) {
 workspace,err:=validate.Required("workspace name",workspace,120); if err!=nil { return "",err }
 name,email,err=identity(name,email); if err!=nil { return "",err }
 hash,err:=HashPassword(password); if err!=nil { return "",err }
 token:=Token()
 err=database.Transaction(ctx,s.Pool,func(q *database.Queries) error {
  if err:=q.LockOnboarding(ctx); err!=nil { return err }
  count,err:=q.WorkspaceCount(ctx); if err!=nil { return err }; if count!=0 { return errors.New("o espaço de trabalho já foi criado") }
  workspace,err:=q.CreateWorkspace(ctx,workspace); if err!=nil { return err }
  user,err:=q.CreateUser(ctx,database.CreateUserParams{WorkspaceID:workspace.ID,Name:name,Email:email,PasswordHash:hash,Role:"admin"}); if err!=nil { return err }
  for _,label:=range []string{"backend","frontend","performance","auth","documentation"} {
   if _,err=q.CreateLabel(ctx,database.CreateLabelParams{WorkspaceID:workspace.ID,Name:label,Color:"#64748b"}); err!=nil { return err }
  }
  return q.CreateSession(ctx,database.CreateSessionParams{TokenHash:TokenHash(token),UserID:user.ID,ExpiresAt:database.Timestamp(time.Now().Add(30*24*time.Hour))})
 })
 return token,err
}
func (s Service) Login(ctx context.Context,email,password string) (string,error) {
 user,err:=s.Queries.GetUserByEmail(ctx,strings.ToLower(strings.TrimSpace(email)))
 if errors.Is(err,pgx.ErrNoRows) {
  // Keep missing-account responses on the same password derivation path.
  _,_ = pbkdf2.Key(sha256.New,password,make([]byte,16),600000,32)
  return "",ErrCredentials
 }
 if err!=nil { return "",err }
 if !CheckPassword(user.PasswordHash,password) { return "",ErrCredentials }
 token:=Token()
 err=s.Queries.CreateSession(ctx,database.CreateSessionParams{TokenHash:TokenHash(token),UserID:user.ID,ExpiresAt:database.Timestamp(time.Now().Add(30*24*time.Hour))})
 return token,err
}
func (s Service) AddUser(ctx context.Context,actor User,name,email,password,role string) error {
 if actor.Role!="admin" { return ErrForbidden }
 name,email,err:=identity(name,email); if err!=nil { return err }
 if _,err=validate.OneOf("role",role,"admin","member"); err!=nil { return err }
 hash,err:=HashPassword(password); if err!=nil { return err }
 _,err=s.Queries.CreateUser(ctx,database.CreateUserParams{WorkspaceID:actor.WorkspaceID,Name:name,Email:email,PasswordHash:hash,Role:role})
 if err!=nil { return fmt.Errorf("create user: %w",err) }; return nil
}
