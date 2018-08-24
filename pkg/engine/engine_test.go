/*
Copyright The Helm Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package engine

import (
	"fmt"
	"sync"
	"testing"

	"k8s.io/helm/pkg/chart"
	"k8s.io/helm/pkg/chartutil"
)

func TestSortTemplates(t *testing.T) {
	tpls := map[string]renderable{
		"/mychart/templates/foo.tpl":                                 {},
		"/mychart/templates/charts/foo/charts/bar/templates/foo.tpl": {},
		"/mychart/templates/bar.tpl":                                 {},
		"/mychart/templates/charts/foo/templates/bar.tpl":            {},
		"/mychart/templates/_foo.tpl":                                {},
		"/mychart/templates/charts/foo/templates/foo.tpl":            {},
		"/mychart/templates/charts/bar/templates/foo.tpl":            {},
	}
	got := sortTemplates(tpls)
	if len(got) != len(tpls) {
		t.Fatal("Sorted results are missing templates")
	}

	expect := []string{
		"/mychart/templates/charts/foo/charts/bar/templates/foo.tpl",
		"/mychart/templates/charts/foo/templates/foo.tpl",
		"/mychart/templates/charts/foo/templates/bar.tpl",
		"/mychart/templates/charts/bar/templates/foo.tpl",
		"/mychart/templates/foo.tpl",
		"/mychart/templates/bar.tpl",
		"/mychart/templates/_foo.tpl",
	}
	for i, e := range expect {
		if got[i] != e {
			t.Errorf("expected %q, got %q at index %d\n\tExp: %#v\n\tGot: %#v", e, got[i], i, expect, got)
		}
	}
}

func TestEngine(t *testing.T) {
	e := New()

	// Forbidden because they allow access to the host OS.
	forbidden := []string{"env", "expandenv"}
	for _, f := range forbidden {
		if _, ok := e.funcMap[f]; ok {
			t.Errorf("Forbidden function %s exists in FuncMap.", f)
		}
	}
}

func TestFuncMap(t *testing.T) {
	fns := FuncMap()
	forbidden := []string{"env", "expandenv"}
	for _, f := range forbidden {
		if _, ok := fns[f]; ok {
			t.Errorf("Forbidden function %s exists in FuncMap.", f)
		}
	}

	// Test for Engine-specific template functions.
	expect := []string{"include", "required", "tpl", "toYaml", "fromYaml", "toToml", "toJson", "fromJson"}
	for _, f := range expect {
		if _, ok := fns[f]; !ok {
			t.Errorf("Expected add-on function %q", f)
		}
	}
}

func TestRender(t *testing.T) {
	c := &chart.Chart{
		Metadata: &chart.Metadata{
			Name:    "moby",
			Version: "1.2.3",
		},
		Templates: []*chart.File{
			{Name: "templates/test1", Data: []byte("{{.outer | title }} {{.inner | title}}")},
			{Name: "templates/test2", Data: []byte("{{.global.callme | lower }}")},
			{Name: "templates/test3", Data: []byte("{{.noValue}}")},
		},
		Values: []byte("outer: DEFAULT\ninner: DEFAULT"),
	}

	vals := []byte(`
outer: spouter
inner: inn
global:
  callme: Ishmael
`)

	e := New()
	v, err := chartutil.CoalesceValues(c, vals)
	if err != nil {
		t.Fatalf("Failed to coalesce values: %s", err)
	}
	out, err := e.Render(c, v)
	if err != nil {
		t.Errorf("Failed to render templates: %s", err)
	}

	expect := "Spouter Inn"
	if out["moby/templates/test1"] != expect {
		t.Errorf("Expected %q, got %q", expect, out["test1"])
	}

	expect = "ishmael"
	if out["moby/templates/test2"] != expect {
		t.Errorf("Expected %q, got %q", expect, out["test2"])
	}
	expect = ""
	if out["moby/templates/test3"] != expect {
		t.Errorf("Expected %q, got %q", expect, out["test3"])
	}

	if _, err := e.Render(c, v); err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
}

func TestRenderInternals(t *testing.T) {
	// Test the internals of the rendering tool.
	e := New()

	vals := chartutil.Values{"Name": "one", "Value": "two"}
	tpls := map[string]renderable{
		"one": {tpl: `Hello {{title .Name}}`, vals: vals},
		"two": {tpl: `Goodbye {{upper .Value}}`, vals: vals},
		// Test whether a template can reliably reference another template
		// without regard for ordering.
		"three": {tpl: `{{template "two" dict "Value" "three"}}`, vals: vals},
	}

	out, err := e.render(nil, tpls)
	if err != nil {
		t.Fatalf("Failed template rendering: %s", err)
	}

	if len(out) != 3 {
		t.Fatalf("Expected 3 templates, got %d", len(out))
	}

	if out["one"] != "Hello One" {
		t.Errorf("Expected 'Hello One', got %q", out["one"])
	}

	if out["two"] != "Goodbye TWO" {
		t.Errorf("Expected 'Goodbye TWO'. got %q", out["two"])
	}

	if out["three"] != "Goodbye THREE" {
		t.Errorf("Expected 'Goodbye THREE'. got %q", out["two"])
	}
}

func TestParallelRenderInternals(t *testing.T) {
	// Make sure that we can use one Engine to run parallel template renders.
	e := New()
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(i int) {
			fname := "my/file/name"
			tt := fmt.Sprintf("expect-%d", i)
			v := chartutil.Values{"val": tt}
			tpls := map[string]renderable{fname: {tpl: `{{.val}}`, vals: v}}
			out, err := e.render(nil, tpls)
			if err != nil {
				t.Errorf("Failed to render %s: %s", tt, err)
			}
			if out[fname] != tt {
				t.Errorf("Expected %q, got %q", tt, out[fname])
			}
			wg.Done()
		}(i)
	}
	wg.Wait()
}

func TestAllTemplates(t *testing.T) {
	ch1 := &chart.Chart{
		Metadata: &chart.Metadata{Name: "ch1"},
		Templates: []*chart.File{
			{Name: "templates/foo", Data: []byte("foo")},
			{Name: "templates/bar", Data: []byte("bar")},
		},
	}
	dep1 := &chart.Chart{
		Metadata: &chart.Metadata{Name: "laboratory mice"},
		Templates: []*chart.File{
			{Name: "templates/pinky", Data: []byte("pinky")},
			{Name: "templates/brain", Data: []byte("brain")},
		},
	}
	ch1.AddDependency(dep1)

	dep2 := &chart.Chart{
		Metadata: &chart.Metadata{Name: "same thing we do every night"},
		Templates: []*chart.File{
			{Name: "templates/innermost", Data: []byte("innermost")},
		},
	}
	dep1.AddDependency(dep2)

	var v chartutil.Values
	tpls := allTemplates(ch1, v)
	if len(tpls) != 5 {
		t.Errorf("Expected 5 charts, got %d", len(tpls))
	}
}

func TestRenderDependency(t *testing.T) {
	e := New()
	deptpl := `{{define "myblock"}}World{{end}}`
	toptpl := `Hello {{template "myblock"}}`
	ch := &chart.Chart{
		Metadata: &chart.Metadata{Name: "outerchart"},
		Templates: []*chart.File{
			{Name: "templates/outer", Data: []byte(toptpl)},
		},
	}
	ch.AddDependency(&chart.Chart{
		Metadata: &chart.Metadata{Name: "innerchart"},
		Templates: []*chart.File{
			{Name: "templates/inner", Data: []byte(deptpl)},
		},
	})

	out, err := e.Render(ch, map[string]interface{}{})
	if err != nil {
		t.Fatalf("failed to render chart: %s", err)
	}

	if len(out) != 2 {
		t.Errorf("Expected 2, got %d", len(out))
	}

	expect := "Hello World"
	if out["outerchart/templates/outer"] != expect {
		t.Errorf("Expected %q, got %q", expect, out["outer"])
	}

}

func TestRenderNestedValues(t *testing.T) {
	e := New()

	innerpath := "templates/inner.tpl"
	outerpath := "templates/outer.tpl"
	// Ensure namespacing rules are working.
	deepestpath := "templates/inner.tpl"
	checkrelease := "templates/release.tpl"

	deepest := &chart.Chart{
		Metadata: &chart.Metadata{Name: "deepest"},
		Templates: []*chart.File{
			{Name: deepestpath, Data: []byte(`And this same {{.Values.what}} that smiles {{.Values.global.when}}`)},
			{Name: checkrelease, Data: []byte(`Tomorrow will be {{default "happy" .Release.Name }}`)},
		},
		Values: []byte(`what: "milkshake"`),
	}

	inner := &chart.Chart{
		Metadata: &chart.Metadata{Name: "herrick"},
		Templates: []*chart.File{
			{Name: innerpath, Data: []byte(`Old {{.Values.who}} is still a-flyin'`)},
		},
		Values: []byte(`who: "Robert"`),
	}
	inner.AddDependency(deepest)

	outer := &chart.Chart{
		Metadata: &chart.Metadata{Name: "top"},
		Templates: []*chart.File{
			{Name: outerpath, Data: []byte(`Gather ye {{.Values.what}} while ye may`)},
		},
		Values: []byte(`
what: stinkweed
who: me
herrick:
    who: time`),
	}
	outer.AddDependency(inner)

	injValues := []byte(`
what: rosebuds
herrick:
  deepest:
    what: flower
global:
  when: to-day`)

	tmp, err := chartutil.CoalesceValues(outer, injValues)
	if err != nil {
		t.Fatalf("Failed to coalesce values: %s", err)
	}

	inject := chartutil.Values{
		"Values": tmp,
		"Chart":  outer.Metadata,
		"Release": chartutil.Values{
			"Name": "dyin",
		},
	}

	t.Logf("Calculated values: %v", inject)

	out, err := e.Render(outer, inject)
	if err != nil {
		t.Fatalf("failed to render templates: %s", err)
	}

	fullouterpath := "top/" + outerpath
	if out[fullouterpath] != "Gather ye rosebuds while ye may" {
		t.Errorf("Unexpected outer: %q", out[fullouterpath])
	}

	fullinnerpath := "top/charts/herrick/" + innerpath
	if out[fullinnerpath] != "Old time is still a-flyin'" {
		t.Errorf("Unexpected inner: %q", out[fullinnerpath])
	}

	fulldeepestpath := "top/charts/herrick/charts/deepest/" + deepestpath
	if out[fulldeepestpath] != "And this same flower that smiles to-day" {
		t.Errorf("Unexpected deepest: %q", out[fulldeepestpath])
	}

	fullcheckrelease := "top/charts/herrick/charts/deepest/" + checkrelease
	if out[fullcheckrelease] != "Tomorrow will be dyin" {
		t.Errorf("Unexpected release: %q", out[fullcheckrelease])
	}
}

func TestRenderBuiltinValues(t *testing.T) {
	inner := &chart.Chart{
		Metadata: &chart.Metadata{Name: "Latium"},
		Templates: []*chart.File{
			{Name: "templates/Lavinia", Data: []byte(`{{.Template.Name}}{{.Chart.Name}}{{.Release.Name}}`)},
			{Name: "templates/From", Data: []byte(`{{.Files.author | printf "%s"}} {{.Files.Get "book/title.txt"}}`)},
		},
		Files: []*chart.File{
			{Name: "author", Data: []byte("Virgil")},
			{Name: "book/title.txt", Data: []byte("Aeneid")},
		},
	}

	outer := &chart.Chart{
		Metadata: &chart.Metadata{Name: "Troy"},
		Templates: []*chart.File{
			{Name: "templates/Aeneas", Data: []byte(`{{.Template.Name}}{{.Chart.Name}}{{.Release.Name}}`)},
		},
	}
	outer.AddDependency(inner)

	inject := chartutil.Values{
		"Values": "",
		"Chart":  outer.Metadata,
		"Release": chartutil.Values{
			"Name": "Aeneid",
		},
	}

	t.Logf("Calculated values: %v", outer)

	out, err := New().Render(outer, inject)
	if err != nil {
		t.Fatalf("failed to render templates: %s", err)
	}

	expects := map[string]string{
		"Troy/charts/Latium/templates/Lavinia": "Troy/charts/Latium/templates/LaviniaLatiumAeneid",
		"Troy/templates/Aeneas":                "Troy/templates/AeneasTroyAeneid",
		"Troy/charts/Latium/templates/From":    "Virgil Aeneid",
	}
	for file, expect := range expects {
		if out[file] != expect {
			t.Errorf("Expected %q, got %q", expect, out[file])
		}
	}

}

func TestAlterFuncMap_include(t *testing.T) {
	c := &chart.Chart{
		Metadata: &chart.Metadata{Name: "conrad"},
		Templates: []*chart.File{
			{Name: "templates/quote", Data: []byte(`{{include "conrad/templates/_partial" . | indent 2}} dead.`)},
			{Name: "templates/_partial", Data: []byte(`{{.Release.Name}} - he`)},
		},
	}

	v := chartutil.Values{
		"Values": "",
		"Chart":  c.Metadata,
		"Release": chartutil.Values{
			"Name": "Mistah Kurtz",
		},
	}

	out, err := New().Render(c, v)
	if err != nil {
		t.Fatal(err)
	}

	expect := "  Mistah Kurtz - he dead."
	if got := out["conrad/templates/quote"]; got != expect {
		t.Errorf("Expected %q, got %q (%v)", expect, got, out)
	}
}

func TestAlterFuncMap_require(t *testing.T) {
	c := &chart.Chart{
		Metadata: &chart.Metadata{Name: "conan"},
		Templates: []*chart.File{
			{Name: "templates/quote", Data: []byte(`All your base are belong to {{ required "A valid 'who' is required" .Values.who }}`)},
			{Name: "templates/bases", Data: []byte(`All {{ required "A valid 'bases' is required" .Values.bases }} of them!`)},
		},
	}

	v := chartutil.Values{
		"Values": chartutil.Values{
			"who":   "us",
			"bases": 2,
		},
		"Chart": c.Metadata,
		"Release": chartutil.Values{
			"Name": "That 90s meme",
		},
	}

	out, err := New().Render(c, v)
	if err != nil {
		t.Fatal(err)
	}

	expectStr := "All your base are belong to us"
	if gotStr := out["conan/templates/quote"]; gotStr != expectStr {
		t.Errorf("Expected %q, got %q (%v)", expectStr, gotStr, out)
	}
	expectNum := "All 2 of them!"
	if gotNum := out["conan/templates/bases"]; gotNum != expectNum {
		t.Errorf("Expected %q, got %q (%v)", expectNum, gotNum, out)
	}
}

func TestAlterFuncMap_tpl(t *testing.T) {
	c := &chart.Chart{
		Metadata: &chart.Metadata{Name: "TplFunction"},
		Templates: []*chart.File{
			{Name: "templates/base", Data: []byte(`Evaluate tpl {{tpl "Value: {{ .Values.value}}" .}}`)},
		},
	}

	v := chartutil.Values{
		"Values": chartutil.Values{
			"value": "myvalue",
		},
		"Chart": c.Metadata,
		"Release": chartutil.Values{
			"Name": "TestRelease",
		},
	}

	out, err := New().Render(c, v)
	if err != nil {
		t.Fatal(err)
	}

	expect := "Evaluate tpl Value: myvalue"
	if got := out["TplFunction/templates/base"]; got != expect {
		t.Errorf("Expected %q, got %q (%v)", expect, got, out)
	}
}

func TestAlterFuncMap_tplfunc(t *testing.T) {
	c := &chart.Chart{
		Metadata: &chart.Metadata{Name: "TplFunction"},
		Templates: []*chart.File{
			{Name: "templates/base", Data: []byte(`Evaluate tpl {{tpl "Value: {{ .Values.value | quote}}" .}}`)},
		},
	}

	v := chartutil.Values{
		"Values": chartutil.Values{
			"value": "myvalue",
		},
		"Chart": c.Metadata,
		"Release": chartutil.Values{
			"Name": "TestRelease",
		},
	}

	out, err := New().Render(c, v)
	if err != nil {
		t.Fatal(err)
	}

	expect := "Evaluate tpl Value: \"myvalue\""
	if got := out["TplFunction/templates/base"]; got != expect {
		t.Errorf("Expected %q, got %q (%v)", expect, got, out)
	}
}

func TestAlterFuncMap_tplinclude(t *testing.T) {
	c := &chart.Chart{
		Metadata: &chart.Metadata{Name: "TplFunction"},
		Templates: []*chart.File{
			{Name: "templates/base", Data: []byte(`{{ tpl "{{include ` + "`" + `TplFunction/templates/_partial` + "`" + ` .  | quote }}" .}}`)},
			{Name: "templates/_partial", Data: []byte(`{{.Template.Name}}`)},
		},
	}
	v := chartutil.Values{
		"Values": chartutil.Values{
			"value": "myvalue",
		},
		"Chart": c.Metadata,
		"Release": chartutil.Values{
			"Name": "TestRelease",
		},
	}

	out, err := New().Render(c, v)
	if err != nil {
		t.Fatal(err)
	}

	expect := "\"TplFunction/templates/base\""
	if got := out["TplFunction/templates/base"]; got != expect {
		t.Errorf("Expected %q, got %q (%v)", expect, got, out)
	}

}
