package task

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/itsabot/abot/core"
	"github.com/itsabot/abot/core/log"
	"github.com/itsabot/abot/shared/datatypes"
	"github.com/itsabot/abot/shared/plugin"
	"github.com/julienschmidt/httprouter"
)

var r *httprouter.Router
var p *dt.Plugin

const (
	keyMem = "mem"
	keyRes = "res"
)

type testT struct {
	Input    string
	Expected string
}

func TestMain(m *testing.M) {
	err := os.Setenv("ABOT_ENV", "test")
	if err != nil {
		log.Fatal("failed to set ABOT_ENV.", err)
	}
	r, err = core.NewServer()
	if err != nil {
		log.Fatal("failed to start abot server.", err)
	}
	p, err = plugin.New("testplugin")
	if err != nil {
		log.Fatal("failed to build test plugin.", err)
	}
	p.Config.Name = "testplugin"
	p.Trigger = &dt.StructuredInput{
		Commands: []string{"get"},
		Objects:  []string{"result"},
	}
	plugin.SetStates(p, [][]dt.State{
		[]dt.State{
			{
				OnEntry: func(in *dt.Msg) string {
					return "entered first"
				},
				OnInput: func(in *dt.Msg) {},
				Complete: func(in *dt.Msg) (bool, string) {
					return true, ""
				},
			},
		},
		Iterate(p, "", OptsIterate{
			IterableMemKey:  keyMem,
			ResultMemKeyIdx: keyRes,
		}),
		[]dt.State{
			{
				OnEntry: func(in *dt.Msg) string {
					return "Great!"
				},
				OnInput: func(in *dt.Msg) {},
				Complete: func(in *dt.Msg) (bool, string) {
					return true, ""
				},
			},
		},
	})
	p.SM.SetOnReset(func(in *dt.Msg) {
		p.DeleteMemory(in, keyMem)
		p.DeleteMemory(in, keyRes)
		ResetIterate(p, in)
	})
	if err = plugin.Register(p); err != nil {
		log.Fatal("failed to register test plugin.", err)
	}
	cleanup()
	exitVal := m.Run()
	cleanup()
	os.Exit(exitVal)
}

func TestIterate(t *testing.T) {
	testEmpty(t)
	testOneResult(t, []string{"item 1"})
	testTwoResults(t, []string{"item 1", "item 2"})
	testThreeResults(t, []string{"item 1", "item 2", "item 3"})
}

func testEmpty(t *testing.T) {
	tests := []testT{
		testT{
			Input:    "get result",
			Expected: "entered first",
		},
		testT{
			Input:    "now go to iterable",
			Expected: "couldn't find any results",
		},
	}
	runner(t, tests, []string{})
}

func testOneResult(t *testing.T, iterable []string) {
	tests := []testT{
		testT{
			Input:    "get result",
			Expected: "entered first",
		},
		testT{
			Input:    "now go to iterable",
			Expected: iterable[0],
		},
		testT{
			Input:    "now go to iterable",
			Expected: "that's all I have",
		},
	}
	runner(t, tests, iterable)
}

func testTwoResults(t *testing.T, iterable []string) {
	tests := []testT{
		testT{
			Input:    "get result",
			Expected: "entered first",
		},
		testT{
			Input:    "now go to iterable",
			Expected: iterable[0],
		},
		testT{
			Input:    "no",
			Expected: iterable[1],
		},
	}
	runner(t, tests, iterable)

	tests = []testT{
		testT{
			Input:    "get result",
			Expected: "entered first",
		},
		testT{
			Input:    "now go to iterable",
			Expected: iterable[0],
		},
		testT{
			Input:    "yes",
			Expected: "Great!",
		},
	}
	runner(t, tests, iterable)
}

func testThreeResults(t *testing.T, iterable []string) {
	tests := []testT{
		testT{
			Input:    "get result",
			Expected: "entered first",
		},
		testT{
			Input:    "now go to iterable",
			Expected: iterable[0],
		},
		testT{
			Input:    "no",
			Expected: iterable[1],
		},
		testT{
			Input:    "yes",
			Expected: "Great!",
		},
	}
	runner(t, tests, iterable)

	tests = []testT{
		testT{
			Input:    "get result",
			Expected: "entered first",
		},
		testT{
			Input:    "now go to iterable",
			Expected: iterable[0],
		},
		testT{
			Input:    "no",
			Expected: iterable[1],
		},
		testT{
			Input:    "no",
			Expected: iterable[2],
		},
		testT{
			Input:    "yes",
			Expected: "Great!",
		},
	}
	runner(t, tests, iterable)

}

func runner(t *testing.T, tests []testT, iterable []string) {
	user := dt.User{FlexID: "0", FlexIDType: 3}
	data := struct {
		FlexIDType dt.FlexIDType
		FlexID     string
		CMD        string
	}{
		FlexIDType: user.FlexIDType,
		FlexID:     user.FlexID,
	}
	u := os.Getenv("ABOT_URL") + "/"
	for i, test := range tests {
		in, err := core.NewMsg(&user, test.Input)
		if err != nil {
			t.Fatal(err)
		}
		if i == 1 {
			p.SetMemory(in, keyMem, iterable)
		}
		data.CMD = test.Input
		byt, err := json.Marshal(data)
		if err != nil {
			t.Fatal("failed to marshal req.", err)
		}
		c, b := request("POST", u, byt)
		if c != http.StatusOK {
			t.Fatal("expected", http.StatusOK, "got", c, b)
		}
		if !strings.Contains(b, test.Expected) {
			t.Fatalf("test %d/%d: expected %q, got %q\n", i+1,
				len(tests), test.Expected, b)
		}
	}
	cleanup()
}

func request(method, path string, data []byte) (int, string) {
	req, err := http.NewRequest(method, path, bytes.NewBuffer(data))
	if err != nil {
		return 0, "err completing request: " + err.Error()
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w.Code, string(w.Body.Bytes())
}

func cleanup() {
	q := `DELETE FROM messages`
	_, err := p.DB.Exec(q)
	if err != nil {
		log.Info("failed to delete messages.", err)
	}
	q = `DELETE FROM states`
	_, err = p.DB.Exec(q)
	if err != nil {
		log.Info("failed to delete messages.", err)
	}
}
