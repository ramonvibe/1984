package layouts

import (
	"context"
	"strings"
	"testing"

	"github.com/ramon/trackline/internal/updatecheck"
)

func TestShellShowsAvailableUpdate(t *testing.T) {
	var output strings.Builder
	data := Data{AppName: "1984", Update: updatecheck.Update{Latest: "0.2.0", URL: "https://github.example/releases/v0.2.0", Available: true}}
	if err := Shell(data, "Teste").Render(context.Background(), &output); err != nil {
		t.Fatal(err)
	}
	for _, value := range []string{"Nova versão disponível", "v0.2.0", `href="https://github.example/releases/v0.2.0"`} {
		if !strings.Contains(output.String(), value) {
			t.Fatalf("missing %q in rendered layout", value)
		}
	}
}
