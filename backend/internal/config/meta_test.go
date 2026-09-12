package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMetaConfig(t *testing.T) {
	c, e := LoadMeta(func(string) string { return "" })
	if e != nil || c.Enabled {
		t.Fatal("disabled")
	}
	values := map[string]string{"META_GRAPH_VERSION": "v26.0", "META_APP_ID": "111", "META_WABA_ID": "222"}
	for i, key := range []string{"META_ACCESS_TOKEN", "META_APP_SECRET", "META_WEBHOOK_VERIFY_TOKEN"} {
		p := filepath.Join(t.TempDir(), "secret")
		if e = os.WriteFile(p, []byte(fmt.Sprintf("synthetic-distinct-secret-%d", i)), 0600); e != nil {
			t.Fatal(e)
		}
		values[key+"_FILE"] = p
	}
	get := func(k string) string { return values[k] }
	c, e = LoadMeta(get)
	if e != nil || !c.Enabled {
		t.Fatal(e)
	}
	if strings.Contains(fmt.Sprint(c), "synthetic-distinct") {
		t.Fatal("secret formatted")
	}
	values["META_GRAPH_VERSION"] = "v18.0"
	if _, e = LoadMeta(get); e == nil {
		t.Fatal("unreviewed version")
	}
	values["META_GRAPH_VERSION"] = "v26.0"
	values["META_APP_SECRET_FILE"] = values["META_ACCESS_TOKEN_FILE"]
	if _, e = LoadMeta(get); e == nil {
		t.Fatal("shared secret")
	}
	values["META_APP_SECRET_FILE"] = "missing"
	if _, e = LoadMeta(get); e == nil {
		t.Fatal("missing secret")
	}
}
