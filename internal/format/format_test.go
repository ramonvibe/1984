package format
import (
 "bytes"
 "context"
 "strings"
 "testing"
)
func TestEscapedMarkdown(t *testing.T) {
 var output bytes.Buffer
 if err:=Markdown("<script>alert(1)</script>\n\n**bold**").Render(context.Background(),&output); err!=nil { t.Fatal(err) }
 if strings.Contains(output.String(),"<script>") || !strings.Contains(output.String(),"<strong>bold</strong>") { t.Fatal("unsafe or broken Markdown",output.String()) }
 if Duration(12300)!="3h 25m" || Duration(-5)!="0m" { t.Fatal("duration formatting failed") }
}
