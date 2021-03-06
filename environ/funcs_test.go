package environ

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"testing"
	"text/template"
)

func TestPlugin(t *testing.T) {
	table := []struct {
		cmd, function string
		ctx           interface{}
		pass          bool
		expect        map[string]interface{}
	}{
		{"nix", "echo", "hello,world", true,
			map[string]interface{}{
				"data": "hello,world",
			},
		},
	}

	for _, v := range table {
		i, err := toJSON(v.ctx)
		if err != nil {
			t.Fatal(err)
		}
		o, err := execJSON(testRunner, v.cmd, i, v.function)
		if err != nil {
			t.Fatal(err)
		}
		for k, e := range o {
			ev := v.expect[k]
			if !reflect.DeepEqual(e, ev) {
				t.Errorf("expected %v got %v", ev, e)
			}
		}
	}

	tpl, err := template.New("plugin").Funcs(
		template.FuncMap{
			"plugin": func(name, function string, ctx interface{}) (interface{}, error) {
				return callPlugin(testRunner, name, function, ctx)
			},
		},
	).Parse(`{{range plugin "nix" "echoPlugin" . }}{{.}}{{end}}`)
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err = tpl.Execute(&buf, "Hello,World"); err != nil {
		t.Fatal(err)
	}
	expect := "Hello,Worldnix echoPlugin"
	got := buf.String()
	if got != expect {
		t.Errorf("expected %s got %s", expect, got)
	}
}

func toJSON(ctx interface{}) ([]byte, error) {
	d := make(map[string]interface{})
	d["data"] = ctx
	return json.Marshal(d)
}

func testRunner(name string, args ...string) *exec.Cmd {
	v := []string{name}
	return helperCMD(append(v, args...)...)
}

func helperCMD(args ...string) *exec.Cmd {
	cmd := exec.Command(os.Args[0], args...)
	cmd.Env = []string{"GO_TEST_ENV=command"}
	return cmd
}

func TestConvert(t *testing.T) {
	t.Run("ordinary use case", func(t *testing.T) {
		expected := []interface{}{"things", "and", "stuff"}

		actual, ok := convert(expected).([]string)
		if !ok {
			t.Error("convert returned an interface that couldn't be converted to []string")
		}

		for i, s := range expected {
			if actual[i] != s {
				t.Errorf("expected %s got %s", s, actual[i])
			}
		}
	})

	t.Run("early exit", func(t *testing.T) {
		expected := []interface{}{"things", nil, "stuff"}
		actual := convert(expected).([]interface{})

		for i, x := range expected {
			if actual[i] != x {
				t.Errorf("expected %v got %v", x, actual[i])
			}
		}
	})
}

func TestMain(t *testing.M) {
	switch os.Getenv("GO_TEST_ENV") {
	case "command":
		args := os.Args[1:]
		switch args[0] {
		case "nix":
			c, err := ioutil.ReadAll(os.Stdin)
			if err != nil {
				log.Fatal(err)
			}
			switch args[1] {
			case "echo":
				fmt.Println(string(c))
			case "echoPlugin":
				d := make(map[string]interface{})
				if err := json.Unmarshal(c, &d); err != nil {
					log.Fatal(err)
				}
				data := d["data"].(string)
				d["data"] = data + "nix echoPlugin"
				v, err := toJSON(d)
				if err != nil {
					log.Fatal(err)
				}
				fmt.Println(string(v))
			}
		}
	default:
		os.Exit(t.Run())
	}
}

func TestAddDirsToPath(t *testing.T) {
	p := os.Args[0]
	dir := filepath.Dir(p)

	name, err := lookUpPlugin([]string{dir}, filepath.Base(p))
	if err != nil {
		t.Fatal(err)
	}
	if name != p {
		t.Errorf("expected %s got %s", p, name)
	}
}
