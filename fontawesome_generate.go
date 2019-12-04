// +build ignore

package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"text/template"
)

var (
	oFlag string

	faVersion = "5.11.2"
)

// These constants are used during generation of SetX functions.
const (
	widthAttrIndex = iota + 2
	heightAttrIndex
	styleAttrIndex
	idAttrIndex
	classAttrIndex
)

func init() {
	flag.StringVar(&oFlag, "o", "", "write output to `file` (default standard output)")
}

func main() {
	flag.Parse()

	err := run()
	if err != nil {
		log.Fatalln(err)
	}
}

func run() error {
	// Get icons
	resp, err := http.Get(fmt.Sprintf("https://raw.githubusercontent.com/FortAwesome/Font-Awesome/%s/metadata/icons.json", faVersion))
	if err != nil {
		return fmt.Errorf("get icons: %v", err)
	}
	defer resp.Body.Close()

	// Generate

	var icons map[string]icon
	err = json.NewDecoder(resp.Body).Decode(&icons)
	if err != nil {
		return fmt.Errorf("decode icons: %v", err)
	}

	var names []string
	for name := range icons {
		names = append(names, name)
	}
	sort.Strings(names)

	var buf bytes.Buffer
	fmt.Fprint(&buf, `package fontawesome

import (
	"html/template"
	"strconv"
	"strings"

	"golang.org/x/net/html"
)

// FontAwesome represents an SVG node
type FontAwesome struct {
	Node *html.Node
}

// XML returns the SVG node as an XML string
func (fa *FontAwesome) XML() string {
	builder := &strings.Builder{}
	if err := html.Render(builder, fa.Node); err != nil {
		return ""
	}
	return builder.String()
}

// HTML returns the SVG node as an HTML template, safe for use in Go templates
func (fa *FontAwesome) HTML() template.HTML {
	return template.HTML(fa.XML())
}

// Size sets the size of a FontAwesome icon
// Short for calling Width and Height with the same int
func (fa *FontAwesome) Size(size int) {
	fa.Width(size)
	fa.Height(size)
}

// Width sets the width of a FontAwesome icon
func (fa *FontAwesome) Width(width int) {
	fa.Node.Attr[`, widthAttrIndex, `].Val = strconv.Itoa(width)
}

// Height sets the height of a FontAwesome icon
func (fa *FontAwesome) Height(height int) {
	fa.Node.Attr[`, heightAttrIndex, `].Val = strconv.Itoa(height)
}

// Style sets the style of a FontAwesome icon
func (fa *FontAwesome) Style(style string) {
	fa.Node.Attr[`, styleAttrIndex, `].Val = style
}

// Id sets the id of a FontAwesome icon
func (fa *FontAwesome) Id(id string) {
	fa.Node.Attr[`, idAttrIndex, `].Val = id
}

// Class sets the class of a FontAwesome icon
func (fa *FontAwesome) Class(class string) {
	fa.Node.Attr[`, classAttrIndex, `].Val = class
}

// Icon returns the named FontAwesome SVG node.
// It returns nil if name is not a valid FontAwesome symbol name.
func Icon(name string) *FontAwesome {
	switch name {
`)
	for _, name := range names {
		ico := icons[name]
		for _, style := range ico.Styles {
			fmt.Fprintf(&buf, "	case %q:\n		return %v()\n", name+"-"+style, kebab(name)+strings.Title(style))
		}
	}
	fmt.Fprint(&buf, `	default:
		return nil
	}
}
`)

	// Write all individual FontAwesome icon functions.
	for _, name := range names {
		generateAndWriteFontAwesome(&buf, icons, name)
	}

	var w io.Writer
	switch oFlag {
	case "":
		w = os.Stdout
	default:
		f, err := os.Create(oFlag)
		if err != nil {
			return err
		}
		defer f.Close()
		w = f
	}

	_, err = w.Write(buf.Bytes())
	return err
}

var faTemplate = template.Must(template.New("fontawesome").Parse(`return &FontAwesome{
	Node: &html.Node{
		FirstChild: &html.Node{
			Type: 3,
			DataAtom: 0,
			Data: "path",
			Namespace: "svg",
			Attr: []html.Attribute{
				{
					Namespace: "",
					Key: "d",
					Val: "{{.Path}}",
				},
			},
		},
		Type: 3,
		DataAtom: 462339,
		Data: "svg",
		Namespace: "svg",
		Attr: []html.Attribute{
			{
				Namespace: "",
				Key: "xmlns",
				Val: "http://www.w3.org/2000/svg",
			},
			{
				Namespace: "",
				Key: "viewbox",
				Val: "{{.VB0}} {{.VB1}} {{.VB2}} {{.VB3}}",
			},
			{
				Namespace: "",
				Key: "width",
				Val: "16",
			},
			{
				Namespace: "",
				Key: "height",
				Val: "16",
			},
			{
				Namespace: "",
				Key: "style",
				Val: "display: inline-block; vertical-align: text-top; fill: currentColor;",
			},
			{
				Namespace: "",
				Key: "id",
				Val: "",
			},
			{
				Namespace: "",
				Key: "class",
				Val: "",
			},
		},
	},
}
`))

type icon struct {
	Styles []string         `json:"styles"`
	SVG    map[string]style `json:"svg"`
}

type style struct {
	ViewBox []string `json:"viewBox"`
	Path    string   `json:"path"`
}

func generateAndWriteFontAwesome(w io.Writer, icons map[string]icon, name string) {
	ico := icons[name]
	for _, style := range ico.Styles {
		fmt.Fprintln(w)
		fmt.Fprintf(w, "// %s returns the %q FontAwesome icon.\n", kebab(name)+strings.Title(style), name+"-"+style)
		fmt.Fprintf(w, "func %s() *FontAwesome {\n", kebab(name)+strings.Title(style))
		st := ico.SVG[style]
		if strings.HasPrefix(st.Path, `<path fill-rule="evenodd" `) {
			st.Path = `<path ` + st.Path[len(`<path fill-rule="evenodd" `):]
		}
		faTemplate.Execute(w, map[string]interface{}{
			"Path": st.Path,
			"VB0":  st.ViewBox[0],
			"VB1":  st.ViewBox[1],
			"VB2":  st.ViewBox[2],
			"VB3":  st.ViewBox[3],
		})
		fmt.Fprintln(w, "}")
	}
}

func kebab(input string) string {
	parts := make([]string, strings.Count(input, "-")+1)
	if _, err := strconv.Atoi(string(input[0])); err == nil {
		input = fmt.Sprintf("fa%s", input)
	}
	for idx, part := range strings.Split(input, "-") {
		parts[idx] = strings.Title(part)
	}
	return strings.Join(parts, "")
}
