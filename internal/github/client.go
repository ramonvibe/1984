package github

import (
 "bytes"
 "context"
 "crypto"
 "crypto/hmac"
 "crypto/rand"
 "crypto/rsa"
 "crypto/sha256"
 "crypto/x509"
 "encoding/base64"
 "encoding/hex"
 "encoding/json"
 "encoding/pem"
 "errors"
 "fmt"
 "io"
 "net/http"
 "net/url"
 "regexp"
 "strconv"
 "strings"
 "time"
)

type Client struct { AppID int64; Key *rsa.PrivateKey; BaseURL string; HTTP *http.Client; WebhookSecret string }
type APIError struct { Status int }
func (e *APIError) Error() string { return fmt.Sprintf("GitHub returned HTTP %d; check App permissions and repository access",e.Status) }

func NewClient(appID int64,privateKey,secret,baseURL string) (*Client,error) {
 c:=&Client{AppID:appID,BaseURL:strings.TrimRight(baseURL,"/"),WebhookSecret:secret,HTTP:&http.Client{Timeout:20*time.Second,CheckRedirect:func(_ *http.Request,_ []*http.Request) error { return http.ErrUseLastResponse }}}
 if appID==0 && privateKey=="" && secret=="" { return c,nil }
 if appID<=0 || privateKey=="" || len(secret)<32 { return nil,errors.New("GitHub requires App ID, private key and a webhook secret of at least 32 bytes") }
 block,_:=pem.Decode([]byte(privateKey)); if block==nil { return nil,errors.New("invalid GitHub private key PEM") }
 key,err:=x509.ParsePKCS1PrivateKey(block.Bytes)
 if err!=nil {
  parsed,parseErr:=x509.ParsePKCS8PrivateKey(block.Bytes); if parseErr!=nil { return nil,errors.New("invalid GitHub RSA private key") }
  var ok bool; key,ok=parsed.(*rsa.PrivateKey); if !ok { return nil,errors.New("GitHub private key must be RSA") }
 }
 if key.N.BitLen()<2048 { return nil,errors.New("GitHub RSA key must be at least 2048 bits") }
 c.Key=key; return c,nil
}
func (c *Client) Enabled() bool { return c!=nil && c.AppID>0 && c.Key!=nil }
func (c *Client) JWT() (string,error) {
 if !c.Enabled() { return "",errors.New("GitHub App is not configured") }
 header:=base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"RS256","typ":"JWT"}`))
 claims,_:=json.Marshal(map[string]any{"iat":time.Now().Add(-time.Minute).Unix(),"exp":time.Now().Add(9*time.Minute).Unix(),"iss":strconv.FormatInt(c.AppID,10)})
 message:=header+"."+base64.RawURLEncoding.EncodeToString(claims)
 digest:=sha256.Sum256([]byte(message))
 signed,err:=rsa.SignPKCS1v15(rand.Reader,c.Key,crypto.SHA256,digest[:]); if err!=nil { return "",err }
 return message+"."+base64.RawURLEncoding.EncodeToString(signed),nil
}
func (c *Client) request(ctx context.Context,method,path,token string,body,output any) error {
 var reader io.Reader
 if body!=nil { data,err:=json.Marshal(body); if err!=nil { return err }; reader=bytes.NewReader(data) }
 request,err:=http.NewRequestWithContext(ctx,method,c.BaseURL+path,reader); if err!=nil { return err }
 request.Header.Set("Authorization","Bearer "+token)
 request.Header.Set("Accept","application/vnd.github+json")
 request.Header.Set("X-GitHub-Api-Version","2022-11-28")
 request.Header.Set("Content-Type","application/json")
 request.Header.Set("User-Agent","Trackline")
 response,err:=c.HTTP.Do(request); if err!=nil { return fmt.Errorf("GitHub request failed: %w",err) }; defer response.Body.Close()
 if response.StatusCode<200 || response.StatusCode>=300 { return &APIError{Status:response.StatusCode} }
 if output==nil { return nil }
 return json.NewDecoder(io.LimitReader(response.Body,4<<20)).Decode(output)
}
func (c *Client) InstallationToken(ctx context.Context,id int64) (string,error) {
 jwt,err:=c.JWT(); if err!=nil { return "",err }
 var result struct{Token string `json:"token"`}
 err=c.request(ctx,"POST",fmt.Sprintf("/app/installations/%d/access_tokens",id),jwt,struct{}{},&result)
 if err==nil && result.Token=="" { return "",errors.New("GitHub returned an empty installation token") }
 return result.Token,err
}
type Installation struct { ID int64 `json:"id"`; Account struct{ID int64 `json:"id"`; Login string `json:"login"`} `json:"account"` }
type Repository struct { ID int64 `json:"id"`; Name string `json:"name"`; FullName string `json:"full_name"`; DefaultBranch string `json:"default_branch"`; Owner struct{Login string `json:"login"`} `json:"owner"` }
var repositoryName=regexp.MustCompile(`^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$`)
func (c *Client) VerifyRepository(ctx context.Context,installationID int64,fullName string) (Installation,Repository,error) {
 var installation Installation; var repository Repository
 if !repositoryName.MatchString(fullName) || installationID<=0 { return installation,repository,errors.New("enter an installation ID and owner/repository") }
 jwt,err:=c.JWT(); if err!=nil { return installation,repository,err }
 if err=c.request(ctx,"GET",fmt.Sprintf("/app/installations/%d",installationID),jwt,nil,&installation); err!=nil { return installation,repository,err }
 token,err:=c.InstallationToken(ctx,installationID); if err!=nil { return installation,repository,err }
 err=c.request(ctx,"GET","/repos/"+fullName,token,nil,&repository)
 if err==nil && (repository.ID<=0 || repository.Name=="" || repository.Owner.Login=="") { err=errors.New("invalid GitHub repository response") }
 return installation,repository,err
}
type PublishedRelease struct { ID int64 `json:"id"`; HTMLURL string `json:"html_url"`; TagName string `json:"tag_name"`; Body string `json:"body"`; Draft bool `json:"draft"` }
func (c *Client) Publish(ctx context.Context,installationID int64,repository,tag,name,body,publicationKey string) (PublishedRelease,error) {
 var result PublishedRelease
 token,err:=c.InstallationToken(ctx,installationID); if err!=nil { return result,err }
 path:="/repos/"+repository+"/releases"
 marker:="<!-- trackline:"+publicationKey+" -->"
 err=c.request(ctx,"GET",path+"/tags/"+url.PathEscape(tag),token,nil,&result)
 if err==nil {
  if strings.Contains(result.Body,marker) && !result.Draft { return result,nil }
  return result,errors.New("a GitHub Release already uses this tag; choose another version")
 }
 var apiErr *APIError
 if !errors.As(err,&apiErr) || apiErr.Status!=404 { return result,err }
 err=c.request(ctx,"POST",path,token,map[string]any{"tag_name":tag,"name":name,"body":body+"\n\n"+marker,"draft":false,"prerelease":false},&result)
 if err==nil && (result.ID<=0 || result.HTMLURL=="") { return result,errors.New("invalid GitHub Release response") }
 return result,err
}
func VerifySignature(secret,signature string,body []byte) bool {
 if secret=="" || !strings.HasPrefix(signature,"sha256=") { return false }
 supplied,err:=hex.DecodeString(strings.TrimPrefix(signature,"sha256=")); if err!=nil { return false }
 mac:=hmac.New(sha256.New,[]byte(secret)); _,_=mac.Write(body)
 return hmac.Equal(supplied,mac.Sum(nil))
}

var references=regexp.MustCompile(`\b[A-Z][A-Z0-9]{1,9}-[1-9][0-9]*\b`)
func References(value string) []string {
 found:=[]string{}; seen:=map[string]bool{}
 for _,item:=range references.FindAllString(value,-1) { if !seen[item] { found=append(found,item); seen[item]=true } }
 return found
}
